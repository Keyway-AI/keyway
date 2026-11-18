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
	"github.com/Keyway-AI/keyway/cloud/postgres"
	"github.com/Keyway-AI/keyway/internal/version"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := cloud.ConfigFromEnv()
	// Persistence: Postgres when KEYWAY_DB_URL is set (hosting), else the
	// in-memory store for local/dev. The Store interface is a drop-in.
	store, cleanup, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
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
		return err
	}
	return nil
}

// openStore returns the cloud store for the environment plus a cleanup func.
// KEYWAY_DB_URL selects the persistent Postgres backend (migrations are applied
// first); without it, the in-memory store backs local/dev and loses data on
// restart.
func openStore(ctx context.Context) (cloud.Store, func(), error) {
	dsn := os.Getenv("KEYWAY_DB_URL")
	if dsn == "" {
		log.Printf("note: KEYWAY_DB_URL not set — using the in-memory store (data is lost on restart)")
		st := cloud.NewMemoryStore()
		return st, func() { _ = st.Close() }, nil
	}
	if err := postgres.MigrateUp(dsn); err != nil {
		return nil, nil, err
	}
	st, err := postgres.Open(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("persistence: Postgres (KEYWAY_DB_URL)")
	return st, func() { _ = st.Close() }, nil
}
