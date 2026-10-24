// Verifier target: lestrrat-go/jwx/v2 (jwt.Parse with a cached JWKS).
// Configured CORRECTLY: validate signature against the JWKS, require iss, aud=rs-A,
// and a valid exp (jwx validates exp when WithValidate(true)). Returns 200 on valid.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
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
	port := env("PORT", "10002")

	ctx := context.Background()
	cache := jwk.NewCache(ctx)
	if err := cache.Register(jwksURL); err != nil {
		log.Fatalf("jwk register: %v", err)
	}

	http.HandleFunc("/api/protected", func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		set, err := cache.Get(ctx, jwksURL)
		if err != nil {
			http.Error(w, "jwks: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = jwt.Parse([]byte(raw),
			jwt.WithKeySet(set),
			jwt.WithValidate(true),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(aud),
			jwt.WithRequiredClaim("exp"),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("go-jwx listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
