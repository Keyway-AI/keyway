package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Keyway-AI/keyway/internal/config"
	"github.com/Keyway-AI/keyway/internal/store/postgres"
)

// The command set mirrors PRD §11. Every subcommand is wired to a real handler;
// the full surface is discoverable via `keyway --help`.

// dbURL resolves the store connection string in precedence: the --db flag, the
// KEYWAY_DB_URL environment variable, then db_url in the config file. The
// config-file fallback keeps the one-shot commands (snapshot, discover, diff, …)
// consistent with `serve`, which also honors db_url from config. Use the DSN
// "memory" for an ephemeral in-process store.
func dbURL(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("db"); v != "" {
		return v
	}
	if v := os.Getenv("KEYWAY_DB_URL"); v != "" {
		return v
	}
	cfgPath, _ := cmd.Flags().GetString("config")
	cfg, _ := config.LoadFile(cfgPath)
	return cfg.DBURL
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Write config and test connectivity",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runInit(cmd) },
	}
}

func newIssuerCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "issuer", Short: "Manage issuers"}

	add := &cobra.Command{
		Use:   "add",
		Short: "Register an issuer in the config file",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runIssuerAdd(cmd) },
	}
	add.Flags().String("name", "", "issuer name (defaults to the type)")
	add.Flags().String("type", "", "issuer type: keycloak|k8s_sa|generic_oidc")
	add.Flags().String("url", "", "issuer URL")
	add.Flags().String("admin-credential-env", "", "env var holding admin credentials")
	_ = add.MarkFlagRequired("type")
	_ = add.MarkFlagRequired("url")

	list := &cobra.Command{
		Use:   "list",
		Short: "List registered issuers",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runIssuerList(cmd) },
	}

	cmd.AddCommand(add, list)
	return cmd
}

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Build and store a contract version (first run establishes a baseline)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSnapshot(cmd)
		},
	}
	cmd.Flags().StringSlice("namespace", nil, "Kubernetes namespaces to scan")
	cmd.Flags().StringSlice("path", nil, "manifest files or directories to scan for consumers")
	cmd.Flags().String("kube-context", "", "kube-context for --in-cluster / cluster name used in StableIDs")
	cmd.Flags().Bool("in-cluster", false, "discover live Istio CRDs from the Kubernetes API (client-go)")
	return cmd
}

func short(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Diff two contract versions",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runDiff(cmd) },
	}
	cmd.Flags().String("from", "", "from version ID (default: baseline)")
	cmd.Flags().String("to", "", "to version ID (default: latest)")
	return cmd
}

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "migrate", Short: "Apply or roll back database migrations"}
	up := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dsn := dbURL(cmd)
			if dsn == "" {
				return fmt.Errorf("no database configured — set KEYWAY_DB_URL, pass --db, or add db_url to config (use --db memory for an ephemeral store)")
			}
			if err := postgres.MigrateUp(dsn); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "migrations applied")
			return nil
		},
	}
	down := &cobra.Command{
		Use:   "down",
		Short: "Roll back the most recent migration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dsn := dbURL(cmd)
			if dsn == "" {
				return fmt.Errorf("no database configured — set KEYWAY_DB_URL, pass --db, or add db_url to config (use --db memory for an ephemeral store)")
			}
			if err := postgres.MigrateDown(dsn); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "rolled back one migration")
			return nil
		},
	}
	cmd.AddCommand(up, down)
	return cmd
}
