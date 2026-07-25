package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nometria/keyway/internal/api"
	"github.com/nometria/keyway/internal/app"
	"github.com/nometria/keyway/internal/config"
	"github.com/nometria/keyway/internal/issuerregistry"
	"github.com/nometria/keyway/internal/keystore"
	"github.com/nometria/keyway/internal/libdefaults"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/notify"
	"github.com/nometria/keyway/internal/probe"
	"github.com/nometria/keyway/internal/store/postgres"
	"github.com/nometria/keyway/internal/version"
	"github.com/spf13/cobra"
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
	cmd.Flags().String("key-store", "", "directory to persist canary keys (encrypted); requires $KEYWAY_KEY_ENCRYPTION_KEY (32 bytes)")
	return cmd
}

// keyStoreFor builds the durable key store when --key-store is set, sourcing the
// AES key from $KEYWAY_KEY_ENCRYPTION_KEY. Returns (nil, nil) when persistence is
// off, so canary state stays in-memory as before.
func keyStoreFor(cmd *cobra.Command) (keystore.Store, error) {
	dir, _ := cmd.Flags().GetString("key-store")
	if dir == "" {
		return nil, nil
	}
	key, err := keystore.KeyFromEnv("KEYWAY_KEY_ENCRYPTION_KEY")
	if err != nil {
		return nil, err
	}
	return keystore.NewFileStore(dir, key)
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

	// Store + migrations.
	if err := postgres.MigrateUp(cfg.DBURL); err != nil {
		return err
	}
	st, err := postgres.Open(ctx, cfg.DBURL)
	if err != nil {
		return err
	}
	defer st.Close()

	// Issuer registry from config, plus an optional default generic issuer.
	specs := issuerSpecs(cfg)
	if url, _ := cmd.Flags().GetString("issuer-url"); url != "" {
		specs = append(specs, issuerregistry.Spec{Name: "default", Type: model.IssuerGenericOIDC, URL: url})
	}
	// Optional durable key store: when --key-store is set, canary key material is
	// persisted (encrypted) so it survives a restart (KI-09).
	keyStore, err := keyStoreFor(cmd)
	if err != nil {
		return err
	}
	reg, err := issuerregistry.NewRegistryWithStore(specs, keyStore)
	if err != nil {
		return err
	}

	libs, _ := libdefaults.Load()

	scope := buildScope(cmd)
	if len(scope.ConfigPaths) == 0 {
		scope.ConfigPaths = cfg.Discovery.ConfigPaths
	}

	allow, _ := cmd.Flags().GetStringSlice("allow")
	if len(allow) == 0 {
		allow = cfg.ProbeAllowlist
	}
	probeCfg := probe.DefaultEngineConfig()
	probeCfg.Allowlist = allow

	token, _ := cmd.Flags().GetString("token")
	if token == "" {
		token = cfg.APIToken
	}
	addr, _ := cmd.Flags().GetString("addr")

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

	// One application-layer dependency set, shared by the HTTP server and the
	// scheduler so both drive the same use-cases.
	appDeps := app.Deps{
		Store:       st,
		Issuers:     reg,
		Libs:        libs,
		Discoverers: discoverers,
		Scope:       scope,
		Probe:       probeCfg,
		Attributor:  buildAttributor(cfg, attrRoot),
	}
	srv := api.NewServer(api.Config{Addr: addr, Token: token}, appDeps)

	// Optional scheduler: periodically snapshot and notify on change events.
	if interval, _ := cmd.Flags().GetDuration("snapshot-interval"); interval > 0 {
		go runScheduler(ctx, cmd.OutOrStdout(), appDeps, notifierFor(cfg), interval)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s listening on %s (%d issuer(s), UI at /)\n", version.String(), addr, reg.Len())
	return srv.ListenAndServe(ctx)
}

// runScheduler snapshots on a fixed interval and delivers notifiable change
// events to the configured notifier (PRD §10.3 / scheduler; KI-06).
func runScheduler(ctx context.Context, out io.Writer, deps app.Deps, n notify.Notifier, interval time.Duration) {
	fmt.Fprintf(out, "scheduler: snapshotting every %s\n", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
