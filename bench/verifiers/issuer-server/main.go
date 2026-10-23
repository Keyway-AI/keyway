// Standalone test issuer for the empirical eval. Generates one RS256 key, writes it
// (localkeys.Export JSON) to --key-out so `keyway probe --issuer-key` mints with the
// SAME key, and serves the public JWKS + OIDC discovery at --addr. Start this BEFORE
// the verifier targets so their startup JWKS fetch succeeds. Test-only; never a real key.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Keyway-AI/keyway/internal/issuer/localkeys"
	"github.com/Keyway-AI/keyway/internal/model"
)

func main() {
	addr := flag.String("addr", ":8099", "listen address for JWKS + discovery")
	issuer := flag.String("issuer", "http://localhost:8099", "issuer URL (iss claim / discovery)")
	keyOut := flag.String("key-out", "/tmp/keyway-eval-issuerkey.json", "where to write the persisted key for --issuer-key")
	flag.Parse()

	ks := localkeys.NewKeySet()
	if _, err := ks.Generate("RS256", model.KeyActive, time.Now()); err != nil {
		log.Fatalf("generate key: %v", err)
	}
	blob, err := json.MarshalIndent(ks.Export(), "", "  ")
	if err != nil {
		log.Fatalf("export key: %v", err)
	}
	if err := os.WriteFile(*keyOut, blob, 0o600); err != nil {
		log.Fatalf("write key-out: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ks.JWKS())
	})
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                *issuer,
			"jwks_uri":                              strings.TrimRight(*issuer, "/") + "/.well-known/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	log.Printf("issuer-server: serving JWKS at http://%s/.well-known/jwks.json (iss=%s); key written to %s", *addr, *issuer, *keyOut)
	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
