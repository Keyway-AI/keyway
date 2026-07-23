package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/architsharma/keyway/internal/api"
	"github.com/architsharma/keyway/internal/config"
	"github.com/architsharma/keyway/internal/issuerregistry"
	"github.com/architsharma/keyway/internal/libdefaults"
	"github.com/architsharma/keyway/internal/model"
	"github.com/architsharma/keyway/internal/probe"
	"github.com/architsharma/keyway/internal/store/postgres"
	"github.com/architsharma/keyway/internal/version"
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
	return cmd
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
	reg, err := issuerregistry.NewRegistry(specs)
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

	srv := api.NewServer(api.Config{Addr: addr, Token: token}, api.Deps{
		Store:       st,
		Issuers:     reg,
		Libs:        libs,
		Discoverers: defaultDiscoverers(),
		Scope:       scope,
		Probe:       probeCfg,
	})

	fmt.Fprintf(cmd.OutOrStdout(), "%s listening on %s (%d issuer(s), UI at /)\n", version.String(), addr, reg.Len())
	return srv.ListenAndServe(ctx)
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
