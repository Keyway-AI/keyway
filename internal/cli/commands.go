package cli

import (
	"fmt"
	"os"

	"github.com/architsharma/keyway/internal/store/postgres"
	"github.com/spf13/cobra"
)

// The command set mirrors PRD §11. Handlers are wired milestone by milestone
// (see PROGRESS.md); until then they return a clear notImplemented error so the
// full CLI surface is discoverable via --help from day one.

// dbURL resolves the Postgres connection string from the --db flag, falling
// back to the KEYWAY_DB_URL environment variable.
func dbURL(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("db"); v != "" {
		return v
	}
	return os.Getenv("KEYWAY_DB_URL")
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Write config and test connectivity",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M1", "init")
		},
	}
}

func newIssuerCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "issuer", Short: "Manage issuers"}

	add := &cobra.Command{
		Use:   "add",
		Short: "Register an issuer",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M2", "issuer add")
		},
	}
	add.Flags().String("type", "", "issuer type: keycloak|k8s_sa|generic_oidc")
	add.Flags().String("url", "", "issuer URL")
	add.Flags().String("admin-credential-env", "", "env var holding admin credentials")
	_ = add.MarkFlagRequired("type")
	_ = add.MarkFlagRequired("url")

	list := &cobra.Command{
		Use:   "list",
		Short: "List registered issuers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M2", "issuer list")
		},
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
	cmd.Flags().String("kube-context", "", "cluster name used in StableIDs")
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M4", "diff")
		},
	}
	cmd.Flags().String("from", "", "from version (default: baseline)")
	cmd.Flags().String("to", "", "to version (default: latest)")
	return cmd
}

func newBlastRadiusCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "blast-radius", Short: "Compute who breaks under a proposed change"}

	rotate := &cobra.Command{
		Use:   "rotate-key",
		Short: "Blast radius of rotating a signing key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M7", "blast-radius rotate-key")
		},
	}
	rotate.Flags().String("issuer", "", "issuer ID")
	rotate.Flags().String("kid", "", "key ID being rotated")
	rotate.Flags().Bool("verbose", false, "list ready and unknown consumers")

	removeClaim := &cobra.Command{
		Use:   "remove-claim",
		Short: "Blast radius of removing a claim",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M7", "blast-radius remove-claim")
		},
	}
	removeClaim.Flags().String("issuer", "", "issuer ID")
	removeClaim.Flags().String("claim", "", "claim name being removed")

	cmd.AddCommand(rotate, removeClaim)
	return cmd
}

func newCanaryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "canary", Short: "Manage canary keys (announce without signing)"}

	start := &cobra.Command{
		Use:   "start",
		Short: "Announce a key in JWKS without using it to sign",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M6", "canary start")
		},
	}
	start.Flags().String("issuer", "", "issuer ID")
	start.Flags().String("alg", "RS256", "key algorithm")

	status := &cobra.Command{
		Use:   "status",
		Short: "Show which consumers have picked up the canary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M6", "canary status")
		},
	}
	status.Flags().String("issuer", "", "issuer ID")

	promote := &cobra.Command{
		Use:   "promote",
		Short: "Promote an announced key to active signing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M6", "canary promote")
		},
	}
	promote.Flags().String("issuer", "", "issuer ID")
	promote.Flags().String("kid", "", "key ID to promote")

	cmd.AddCommand(start, status, promote)
	return cmd
}

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Summarise recent changes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M9", "report")
		},
	}
	cmd.Flags().String("since", "7d", "time window, e.g. 7d")
	cmd.Flags().String("format", "md", "output format: json|md")
	return cmd
}

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API and scheduler",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented("M9", "serve")
		},
	}
	cmd.Flags().String("addr", ":8080", "listen address")
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
				return fmt.Errorf("no database configured (set --db or KEYWAY_DB_URL)")
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
				return fmt.Errorf("no database configured (set --db or KEYWAY_DB_URL)")
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
