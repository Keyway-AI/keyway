// Package realworld validates Keyway against documented, real-world JWT/JWKS
// incidents (CVEs, GitHub issues, postmortems). Each case reproduces the exact
// vulnerable behavior from a cited source and asserts that Keyway flags it — so
// the suite answers "would Keyway have caught this real risk?" rather than only
// testing synthetic scenarios.
package realworld

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/nometria/keyway/internal/issuer/generic"
)

// vuln describes which security checks a reproduced server gets WRONG. Each flag
// corresponds to a documented real-world failure mode.
type vuln struct {
	acceptNone      bool // accepts alg=none tokens (CVE-2022-23540 class)
	acceptHSConfuse bool // verifies HS256 with the RSA public key (CVE-2022-23541 class)
	skipSignature   bool // decodes claims without verifying the RS256 signature
	skipIssuer      bool // does not validate the iss claim
	skipAudience    bool // does not validate the aud claim
	skipExpiry      bool // does not validate exp
	trustHeader     bool // trusts X-User-Id with no bearer token (gateway misconfig)
}

const (
	wantIssuer   = "https://issuer.test"
	wantAudience = "api"
)

var b64 = base64.RawURLEncoding

// vulnServer builds an HTTP handler that reproduces a specific documented flaw.
// Everything not flagged in v is validated correctly, so a probe only fires on
// the reproduced weakness.
func vulnServer(iss *generic.Adapter, v vuln) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			if v.trustHeader && r.Header.Get("X-User-Id") != "" {
				w.WriteHeader(http.StatusOK) // VULNERABLE: trusts an identity header
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

		switch alg {
		case "none":
			if v.acceptNone {
				accept(w, payload, v) // VULNERABLE: unsigned token honored
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		case "HS256":
			if v.acceptHSConfuse && verifyHSWithPublicKey(iss, raw) {
				accept(w, payload, v) // VULNERABLE: public key used as HMAC secret
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		case "RS256":
			if v.skipSignature {
				accept(w, payload, v) // VULNERABLE: claims trusted without verifying the signature
				return
			}
			claims, ok := verifyRS256(iss, raw)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			accept(w, claims, v)
			return
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
}

// accept applies the claim checks the server DOES perform, then responds.
func accept(w http.ResponseWriter, claims map[string]any, v vuln) {
	now := time.Now()
	if !v.skipIssuer {
		if s, _ := claims["iss"].(string); s != wantIssuer {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	if !v.skipAudience {
		if s, _ := claims["aud"].(string); s != wantAudience {
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
	w.WriteHeader(http.StatusOK)
}

// peek decodes a compact JWT's header alg and payload without verifying it.
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

// verifyRS256 verifies a token against the issuer's JWKS and returns its claims.
func verifyRS256(iss *generic.Adapter, raw string) (map[string]any, bool) {
	tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || len(tok.Headers) == 0 {
		return nil, false
	}
	set := iss.KeySet().JWKS()
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

// verifyHSWithPublicKey reproduces the algorithm-confusion flaw: it verifies an
// HS256 token using the issuer's RSA PUBLIC key PEM as the HMAC secret.
func verifyHSWithPublicKey(iss *generic.Adapter, raw string) bool {
	pem, err := iss.KeySet().PublicKeyPEM(iss.KeySet().ActiveKID())
	if err != nil {
		return false
	}
	tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return false
	}
	var claims map[string]any
	return tok.Claims([]byte(pem), &claims) == nil
}
