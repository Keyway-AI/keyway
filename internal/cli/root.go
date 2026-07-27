// Package cli builds the Keyway command tree, shared by the `keyway` CLI and
// the `keyway-runner` daemon binaries.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nometria/keyway/internal/version"
)

// Options configures the root command tree.
type Options struct {
	// Runner selects the daemon-oriented default command (serve) when true.
	Runner bool
}

// NewRootCmd constructs the full command tree.
func NewRootCmd(opts Options) *cobra.Command {
	root := &cobra.Command{
		Use:   binName(opts.Runner),
		Short: "Know which consumers break before you rotate a key, change an issuer, or drop a claim.",
		Long: "Keyway derives your JWT consumer inventory automatically and verifies it with real\n" +
			"tokens against real staging endpoints, so a key rotation never surprises you.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.String(),
	}

	root.PersistentFlags().String("config", "", "path to config file (default: ./keyway.yaml or $KEYWAY_CONFIG)")
	root.PersistentFlags().String("db", os.Getenv("KEYWAY_DB_URL"), "Postgres connection URL")
	root.PersistentFlags().Bool("json", false, "emit machine-readable JSON where supported")

	root.AddCommand(
		newInitCmd(),
		newIssuerCmd(),
		newDiscoverCmd(),
		newSnapshotCmd(),
		newProbeCmd(),
		newDiffCmd(),
		newBlastRadiusCmd(),
		newCanaryCmd(),
		newThreatsCmd(),
		newReportCmd(),
		newServeCmd(),
		newMigrateCmd(),
		newVersionCmd(),
	)

	if opts.Runner {
		// keyway-runner defaults to serving when invoked with no subcommand.
		root.RunE = func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return newServeCmd().RunE(cmd, args)
			}
			return cmd.Help()
		}
	}

	return root
}

// Execute runs the command tree and maps errors to a process exit code.
func Execute(opts Options) {
	if err := NewRootCmd(opts).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func binName(runner bool) string {
	if runner {
		return "keyway-runner"
	}
	return "keyway"
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version.String())
		},
	}
}
