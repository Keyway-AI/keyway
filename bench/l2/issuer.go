package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"

	"github.com/Keyway-AI/keyway/internal/issuer/generic"
)

// runIssuer serves a minimal OIDC issuer plus a test-only /mint endpoint the
// scorer uses as its MintFunc. It holds the signing key (via Keyway's generic
// issuer) so validators can fetch a real JWKS and the scorer can request real
// signed tokens.
func runIssuer(args []string) error {
	fs := flag.NewFlagSet("issuer", flag.ExitOnError)
	addr := fs.String("addr", ":9000", "listen address")
	_ = fs.Parse(args)

	iss, err := generic.New(generic.Options{Name: "l2", IssuerURL: l2Issuer})
	if err != nil {
		return err
	}
	ctx := context.Background()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Standard OIDC discovery so a real Keyway issuer adapter could point here.
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		base := "http://" + r.Host
		writeJSON(w, map[string]any{
			"issuer":   l2Issuer,
			"jwks_uri": base + "/jwks",
		})
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, iss.KeySet().JWKS())
	})

	// /describe returns Keyway's Issuer model (keys + kids + PEM) so the scorer
	// can resolve the active key for the algorithm-confusion probe.
	mux.HandleFunc("/describe", func(w http.ResponseWriter, _ *http.Request) {
		desc, derr := iss.Describe(ctx)
		if derr != nil {
			http.Error(w, derr.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, desc)
	})

	// /mint is the scorer's MintFunc (test rig only — never ship this).
	mux.HandleFunc("/mint", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			KID    string         `json:"kid"`
			Claims map[string]any `json:"claims"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		tok, merr := iss.MintToken(ctx, req.KID, req.Claims)
		if merr != nil {
			http.Error(w, merr.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"token": tok})
	})

	srv := &http.Server{Addr: *addr, Handler: mux}
	println("l2 issuer listening on", *addr)
	return srv.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
