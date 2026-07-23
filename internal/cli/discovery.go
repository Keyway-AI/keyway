package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/architsharma/keyway/internal/contract"
	"github.com/architsharma/keyway/internal/discovery"
	"github.com/architsharma/keyway/internal/discovery/envoy"
	"github.com/architsharma/keyway/internal/discovery/istio"
	"github.com/architsharma/keyway/internal/discovery/k8s"
	"github.com/architsharma/keyway/internal/model"
	"github.com/architsharma/keyway/internal/store/postgres"
	"github.com/spf13/cobra"
)

// buildScope constructs a discovery scope from common flags.
func buildScope(cmd *cobra.Command) discovery.Scope {
	namespaces, _ := cmd.Flags().GetStringSlice("namespace")
	paths, _ := cmd.Flags().GetStringSlice("path")
	kube, _ := cmd.Flags().GetString("kube-context")
	return discovery.Scope{
		KubeContext: kube,
		Namespaces:  namespaces,
		ConfigPaths: paths,
	}
}

// defaultDiscoverers returns the file-based discovery sources.
func defaultDiscoverers() []discovery.Discoverer {
	return []discovery.Discoverer{istio.New(), k8s.New(), envoy.New()}
}

func newDiscoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Derive the consumer inventory from cluster and config sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope := buildScope(cmd)
			if len(scope.ConfigPaths) == 0 {
				return fmt.Errorf("no --path given; provide manifest files or directories to scan")
			}
			consumers, err := discovery.Run(context.Background(), scope, defaultDiscoverers()...)
			if err != nil {
				return err
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(consumers)
			}
			printConsumerTable(cmd, consumers)
			return nil
		},
	}
	cmd.Flags().StringSlice("namespace", nil, "Kubernetes namespaces to scan")
	cmd.Flags().StringSlice("path", nil, "manifest files or directories to scan")
	cmd.Flags().String("kube-context", "", "cluster name used in StableIDs")
	cmd.Flags().String("output", "table", "output format: json|table")
	return cmd
}

func printConsumerTable(cmd *cobra.Command, consumers []model.Consumer) {
	if len(consumers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No consumers discovered.")
		return
	}
	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTABLE ID\tKIND\tAUDIENCES\tPROBEABLE\tCONF")
	for _, c := range consumers {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%v\t%t\t%.2f\n",
			c.Name, c.StableID, c.Kind, c.Expects.Audiences, c.Probeable, c.Confidence["overall"])
	}
	_ = tw.Flush()
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d consumer(s) discovered.\n", len(consumers))
}

// runSnapshot builds a contract from discovery and stores it via the baseline flow.
func runSnapshot(cmd *cobra.Command) error {
	dsn := dbURL(cmd)
	if dsn == "" {
		return fmt.Errorf("no database configured (set --db or KEYWAY_DB_URL)")
	}
	scope := buildScope(cmd)
	var consumers []model.Consumer
	if len(scope.ConfigPaths) > 0 {
		var err error
		consumers, err = discovery.Run(context.Background(), scope, defaultDiscoverers()...)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()
	st, err := postgres.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()

	v := contract.Build(contract.BuildInput{Consumers: consumers, TriggerKind: "manual"})
	res, err := contract.Snapshot(ctx, st, v)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	switch {
	case res.IsBaseline:
		fmt.Fprintf(out, "Baseline established: %d consumers, %d edges (hash %s)\n",
			len(res.Version.Consumers), len(res.Version.Edges), short(res.Version.Hash))
	case res.Unchanged:
		fmt.Fprintf(out, "No change since latest version (hash %s)\n", short(res.Version.Hash))
	default:
		fmt.Fprintf(out, "New version %s: %d change event(s)\n", short(res.Version.Hash), len(res.Events))
		for _, e := range res.Events {
			fmt.Fprintf(out, "  [%s/%s] %s %s: %v -> %v\n", e.Class, e.Severity, e.ConsumerID, e.Field, e.OldValue, e.NewValue)
		}
	}
	return nil
}
