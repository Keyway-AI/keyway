// Verifier target: go-jose/go-jose/v4 (jwt). Correct config: RS256 only, verify
// signature against the JWKS, require iss + aud=rs-A, validate exp. Returns 200 valid.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	jwksURL := env("JWKS_URL", "http://localhost:8099/.well-known/jwks.json")
	issuer := env("ISSUER", "http://localhost:8099")
	aud := env("AUDIENCE", "rs-A")
	port := env("PORT", "10010")

	var keySet jose.JSONWebKeySet
	resp, err := http.Get(jwksURL) // #nosec G107 -- staging eval issuer
	if err != nil {
		log.Fatalf("fetch jwks: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err := json.Unmarshal(body, &keySet); err != nil {
		log.Fatalf("parse jwks: %v", err)
	}

	http.HandleFunc("/api/protected", func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		keys := keySet.Key(tok.Headers[0].KeyID)
		if len(keys) == 0 {
			http.Error(w, "unknown kid", http.StatusUnauthorized)
			return
		}
		var cl jwt.Claims
		if err := tok.Claims(keys[0].Key, &cl); err != nil { // signature check
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := cl.Validate(jwt.Expected{
			Issuer:      issuer,
			AnyAudience: jwt.Audience{aud},
			Time:        time.Now(),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("go-gojose listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
