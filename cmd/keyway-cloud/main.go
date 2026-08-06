// Command keyway-cloud is the multi-tenant hosted API ("Keyway Cloud"): accounts,
// projects, and persisted auth-contract analysis over the shared engine. It runs
// alongside — not instead of — the self-hosted `keyway serve`; the live features
// (probing/canary/blast) stay in the customer's environment by design.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Keyway-AI/keyway/cloud"
	"github.com/Keyway-AI/keyway/internal/version"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := cloud.ConfigFromEnv()
	// TODO(hosting): swap NewMemoryStore for the Postgres store when KEYWAY_DB_URL
	// is set. The Store interface is a drop-in; memory is per-process (dev).
	store := cloud.NewMemoryStore()
	srv := cloud.NewServer(cfg, store)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //nolint:contextcheck
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("%s (cloud) listening on %s — frontend %s, github-login=%v",
		version.String(), cfg.Addr, cfg.FrontendURL, cfg.GitHubClientID != "")
	if cfg.GitHubClientID == "" {
		log.Printf("note: GitHub OAuth not configured; set GITHUB_CLIENT_ID/SECRET (or KEYWAY_CLOUD_DEV_LOGIN=1 for local dev)")
	}
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
