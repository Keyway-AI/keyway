package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// vulnFlags is the validator's configured weakness (from the VULN env var).
type vulnFlags struct {
	acceptNone      bool
	acceptHSConfuse bool
	skipSignature   bool
	skipIssuer      bool
	skipAudience    bool
	skipExpiry      bool
	trustHeader     bool
}

func vulnFromEnv(name string) (vulnFlags, string) {
	switch name {
	case "none":
		return vulnFlags{acceptNone: true}, "accepts alg=none (CVE-2022-23540 class)"
	case "confusion":
		return vulnFlags{acceptHSConfuse: true}, "verifies HS256 with the RSA public key (CVE-2022-23541 class)"
	case "nosig":
		return vulnFlags{skipSignature: true}, "trusts claims without verifying the signature (CWE-347)"
	case "noaud":
		return vulnFlags{skipAudience: true}, "does not validate aud (RFC 8725 §3.9)"
	case "noiss":
		return vulnFlags{skipIssuer: true}, "does not validate iss (RFC 8725 §3.8)"
	case "noexp":
		return vulnFlags{skipExpiry: true}, "does not validate exp (RFC 8725 §3.9)"
	case "header":
		return vulnFlags{trustHeader: true}, "trusts X-User-Id without a token (CWE-290)"
	default:
		return vulnFlags{}, "correctly configured"
	}
}

var b64 = base64.RawURLEncoding

// jwksCache holds the fetched public keys, refreshed periodically.
type jwksCache struct {
	mu  sync.RWMutex
	set jose.JSONWebKeySet
}

func (c *jwksCache) get() jose.JSONWebKeySet {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.set
}

func (c *jwksCache) set0(s jose.JSONWebKeySet) {
	c.mu.Lock()
	c.set = s
	c.mu.Unlock()
}

func runValidator(args []string) error {
	fs := flag.NewFlagSet("validator", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	_ = fs.Parse(args)

	jwksURL := os.Getenv("ISSUER_JWKS_URL")
	if jwksURL == "" {
		jwksURL = "http://issuer:9000/jwks"
	}
	v, desc := vulnFromEnv(os.Getenv("VULN"))

	cache := &jwksCache{}
	if err := refreshJWKS(cache, jwksURL); err != nil {
		// Keep retrying in the background; the issuer may still be starting.
		fmt.Fprintln(os.Stderr, "validator: initial JWKS fetch failed, will retry:", err)
	}
	go func() {
		for {
			time.Sleep(10 * time.Second)
			_ = refreshJWKS(cache, jwksURL)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/", validateHandler(cache, v))

	println("l2 validator listening on", *addr, "— VULN:", desc)
	return (&http.Server{Addr: *addr, Handler: mux}).ListenAndServe()
}

func refreshJWKS(cache *jwksCache, url string) error {
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(body, &set); err != nil {
		return err
	}
	cache.set0(set)
	return nil
}

func validateHandler(cache *jwksCache, v vulnFlags) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			if v.trustHeader && r.Header.Get("X-User-Id") != "" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		alg, payload, ok := peek(raw)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		set := cache.get()
		switch alg {
		case "none":
			if v.acceptNone {
				accept(w, payload, v)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
		case "HS256":
			if v.acceptHSConfuse && verifyHSWithPublicKey(set, raw) {
				accept(w, payload, v)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
		case "RS256":
			if v.skipSignature {
				accept(w, payload, v)
				return
			}
			claims, ok := verifyRS256(set, raw)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			accept(w, claims, v)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func accept(w http.ResponseWriter, claims map[string]any, v vulnFlags) {
	now := time.Now()
	if !v.skipIssuer {
		if s, _ := claims["iss"].(string); s != l2Issuer {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	if !v.skipAudience {
		if s, _ := claims["aud"].(string); s != l2Audience {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	if !v.skipExpiry {
		if exp, ok := claims["exp"].(float64); ok && now.Unix() > int64(exp) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	// nbf (not-before) is honored by every non-broken service, independent of
	// the exp weakness, so the not_yet_valid probe passes everywhere except a
	// service that deliberately ignores it.
	if nbf, ok := claims["nbf"].(float64); ok && now.Unix() < int64(nbf) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func peek(token string) (alg string, payload map[string]any, ok bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", nil, false
	}
	hb, err := b64.DecodeString(parts[0])
	if err != nil {
		return "", nil, false
	}
	var hdr struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(hb, &hdr) != nil {
		return "", nil, false
	}
	pb, err := b64.DecodeString(parts[1])
	if err != nil {
		return "", nil, false
	}
	payload = map[string]any{}
	_ = json.Unmarshal(pb, &payload)
	return hdr.Alg, payload, true
}

func verifyRS256(set jose.JSONWebKeySet, raw string) (map[string]any, bool) {
	tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || len(tok.Headers) == 0 {
		return nil, false
	}
	keys := set.Key(tok.Headers[0].KeyID)
	if len(keys) == 0 {
		return nil, false
	}
	var claims map[string]any
	if err := tok.Claims(keys[0].Key, &claims); err != nil {
		return nil, false
	}
	return claims, true
}

// verifyHSWithPublicKey reproduces algorithm confusion: HMAC-verify with the
// active RSA public key's PEM (the exact bytes Keyway's probe uses as secret).
func verifyHSWithPublicKey(set jose.JSONWebKeySet, raw string) bool {
	if len(set.Keys) == 0 {
		return false
	}
	pub, ok := set.Keys[0].Key.(*rsa.PublicKey)
	if !ok {
		return false
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return false
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return false
	}
	var claims map[string]any
	return tok.Claims([]byte(pemStr), &claims) == nil
}
