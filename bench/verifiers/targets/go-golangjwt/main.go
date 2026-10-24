// Verifier target: golang-jwt/jwt/v5 + MicahParks/keyfunc/v3 (JWKS).
// Configured CORRECTLY: RS256 only, require iss, aud=rs-A, and a valid exp. It must
// reject every attack token; a 200 is a finding. Returns 200 on a valid token.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
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
	port := env("PORT", "10001")

	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Fatalf("keyfunc: %v", err)
	}

	http.HandleFunc("/api/protected", func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		_, err := jwt.Parse(raw, k.Keyfunc,
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(aud),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("go-golangjwt listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
