package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/nometria/keyway/internal/api"
	"github.com/nometria/keyway/internal/app"
	"github.com/nometria/keyway/internal/config"
	"github.com/nometria/keyway/internal/coordination"
	"github.com/nometria/keyway/internal/issuerregistry"
	"github.com/nometria/keyway/internal/keystore"
	"github.com/nometria/keyway/internal/libdefaults"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/notify"
	"github.com/nometria/keyway/internal/version"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API and scheduler (serves the web UI too)",
		RunE:  runServe,
	}
	cmd.Flags().String("addr", ":8080", "listen address")
	cmd.Flags().String("token", "", "API bearer token (default: $KEYWAY_API_TOKEN)")
	cmd.Flags().StringSlice("path", nil, "manifest files/directories for discovery")
	cmd.Flags().StringSlice("allow", nil, "host substrings the probe engine may target")
	cmd.Flags().String("issuer-url", "", "register a default generic issuer at this URL")
	cmd.Flags().Duration("snapshot-interval", 0, "if >0, snapshot+notify on this interval (e.g. 1h)")
	cmd.Flags().Bool("in-cluster", false, "discover live Istio CRDs from the Kubernetes API (client-go) instead of, or in addition to, --path")
	cmd.Flags().String("kube-context", "", "kube-context to use for --in-cluster (default: in-cluster SA, else current context)")
	cmd.Flags().String("key-store", "", "directory to persist canary keys (encrypted, single-node); mutually exclusive with --key-store-db")
	cmd.Flags().Bool("key-store-db", false, "persist canary keys (encrypted) in Postgres so they are shared across replicas (requires a Postgres --db)")
	cmd.Flags().String("key-encryption-key-file", "", "read the 32-byte AES key from this file (e.g. a mounted secret) instead of $KEYWAY_KEY_ENCRYPTION_KEY")
	return cmd
}

// keyPersistenceFor resolves how operated (canary) keys are persisted from the
// flags. The AES root key is sourced from a secret manager, in precedence:
// $KEYWAY_KEY_ENCRYPTION_KEY_CMD (a command, e.g. `vault kv get …`) >
// --key-encryption-key-file (a mounted secret) > $KEYWAY_KEY_ENCRYPTION_KEY.
func keyPersistenceFor(cmd *cobra.Command) (app.KeyPersistence, error) {
	dir, _ := cmd.Flags().GetString("key-store")
	db, _ := cmd.Flags().GetBool("key-store-db")
	if dir != "" && db {
		return app.KeyPersistence{}, fmt.Errorf("--key-store and --key-store-db are mutually exclusive")
	}
	src := keystore.KeySource{
		Env:     "KEYWAY_KEY_ENCRYPTION_KEY",
		Command: os.Getenv("KEYWAY_KEY_ENCRYPTION_KEY_CMD"),
	}
	keyFile, _ := cmd.Flags().GetString("key-encryption-key-file")
	src.File = keyFile
	switch {
	case db:
		return app.KeyPersistence{Mode: "postgres", Source: src}, nil
	case dir != "":
		return app.KeyPersistence{Mode: "file", Dir: dir, Source: src}, nil
	default:
		return app.KeyPersistence{Mode: "none"}, nil
	}
}

func runServe(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfgPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	if dsn := dbURL(cmd); dsn != "" {
		cfg.DBURL = dsn
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Issuer specs from config, plus an optional default generic issuer.
	specs := issuerSpecs(cfg)
	if url, _ := cmd.Flags().GetString("issuer-url"); url != "" {
		specs = append(specs, issuerregistry.Spec{Name: "default", Type: model.IssuerGenericOIDC, URL: url})
	}

	scope := buildScope(cmd)
	if len(scope.ConfigPaths) == 0 {
		scope.ConfigPaths = cfg.Discovery.ConfigPaths
	}
	allow, _ := cmd.Flags().GetStringSlice("allow")
	if len(allow) == 0 {
		allow = cfg.ProbeAllowlist
	}
	attrRoot := "."
	if len(scope.ConfigPaths) > 0 {
		attrRoot = scope.ConfigPaths[0]
	}
	discoverers := append(defaultDiscoverers(), configDiscoverers(cfg)...)
	inc, err := inClusterDiscoverers(cmd)
	if err != nil {
		return err
	}
	discoverers = append(discoverers, inc...)

	keyPersistence, err := keyPersistenceFor(cmd)
	if err != nil {
		return err
	}
	libs, _ := libdefaults.Load()

	// The composition root assembles the whole object graph (store, coordination
	// seams, key store, issuer registry) once; the HTTP server and the scheduler
	// are thin callers of the resulting use-cases.
	application, err := app.Build(ctx, app.BuildConfig{
		DBURL:          cfg.DBURL,
		IssuerSpecs:    specs,
		Discoverers:    discoverers,
		Scope:          scope,
		ProbeAllowlist: allow,
		Attributor:     buildAttributor(cfg, attrRoot),
		Libs:           libs,
		KeyPersistence: keyPersistence,
	})
	if err != nil {
		return err
	}
	defer application.Close()

	token, _ := cmd.Flags().GetString("token")
	if token == "" {
		token = cfg.APIToken
	}
	addr, _ := cmd.Flags().GetString("addr")

	// The HTTP server shares the coordinator's idempotency store, so a retried
	// write replays across replicas (not just within one process).
	srv := api.NewServer(api.Config{Addr: addr, Token: token}, application.Deps).
		WithIdempotency(application.Idempotency())

	// Optional scheduler: periodically snapshot and notify on change events. It is
	// leader-gated so exactly one replica snapshots on the interval.
	if interval, _ := cmd.Flags().GetDuration("snapshot-interval"); interval > 0 {
		go runScheduler(ctx, cmd.OutOrStdout(), application.Deps, notifierFor(cfg), interval, application.Leader())
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s listening on %s (%d issuer(s), UI at /)\n", version.String(), addr, application.IssuerCount)
	return srv.ListenAndServe(ctx)
}

// runScheduler snapshots on a fixed interval and delivers notifiable change
// events to the configured notifier (PRD §10.3 / scheduler; KI-06). It is
// leader-gated: when several replicas run, only the one holding leadership
// snapshots on the tick, so the interval fires once cluster-wide.
func runScheduler(ctx context.Context, out io.Writer, deps app.Deps, n notify.Notifier, interval time.Duration, leader coordination.Leader) {
	fmt.Fprintf(out, "scheduler: snapshotting every %s (leader-gated)\n", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !leader.IsLeader(ctx) {
				continue // another replica holds leadership this tick
			}
			res, err := deps.Snapshot(ctx, "scheduled")
			if err != nil {
				fmt.Fprintln(out, "scheduler: snapshot error:", err)
				continue
			}
			switch {
			case res.IsBaseline:
				fmt.Fprintf(out, "scheduler: baseline established (%s)\n", short(res.Version.Hash))
			case len(res.Events) > 0:
				fmt.Fprintf(out, "scheduler: %d change event(s) (%s)\n", len(res.Events), short(res.Version.Hash))
				if n != nil {
					if err := n.Notify(ctx, res.Events); err != nil {
						fmt.Fprintln(out, "scheduler: notify error:", err)
					}
				}
			}
		}
	}
}

// notifierFor builds a notifier from config (Slack if a webhook is set).
func notifierFor(cfg config.Config) notify.Notifier {
	if cfg.Slack.WebhookURL != "" {
		return notify.NewSlack(cfg.Slack.WebhookURL)
	}
	return nil
}

func issuerSpecs(cfg config.Config) []issuerregistry.Spec {
	var specs []issuerregistry.Spec
	for _, ic := range cfg.Issuers {
		specs = append(specs, issuerregistry.Spec{
			Name:         ic.Name,
			Type:         model.IssuerType(ic.Type),
			URL:          ic.URL,
			AdminCredEnv: ic.AdminCredentialEnv,
		})
	}
	return specs
}
