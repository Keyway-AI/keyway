package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Keyway-AI/keyway/internal/app"
	"github.com/Keyway-AI/keyway/internal/attribution"
	"github.com/Keyway-AI/keyway/internal/config"
	"github.com/Keyway-AI/keyway/internal/contract"
	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/discovery/envoy"
	"github.com/Keyway-AI/keyway/internal/discovery/istio"
	"github.com/Keyway-AI/keyway/internal/discovery/k8s"
	"github.com/Keyway-AI/keyway/internal/discovery/kube"
	"github.com/Keyway-AI/keyway/internal/discovery/oidcclient"
	"github.com/Keyway-AI/keyway/internal/libdefaults"
	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/store/open"
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

// configDiscoverers builds the config-driven sources: one Keycloak client-registry
// discoverer per configured keycloak issuer that has admin credentials.
func configDiscoverers(cfg config.Config) []discovery.Discoverer {
	var ds []discovery.Discoverer
	for _, ic := range cfg.Issuers {
		if model.IssuerType(ic.Type) != model.IssuerKeycloak || ic.AdminCredentialEnv == "" {
			continue
		}
		user, pass := splitCreds(os.Getenv(ic.AdminCredentialEnv))
		d, err := oidcclient.New(oidcclient.Options{RealmURL: ic.URL, AdminUser: user, AdminPassword: pass})
		if err == nil {
			ds = append(ds, d)
		}
	}
	return ds
}

// buildAttributor assembles the attribution chain that binds each change event
// to its cause: git blame on the changed manifest, then the K8s deploy
// annotation, then (as a fallback for IdP-side changes) the Keycloak admin-event
// log for each configured Keycloak issuer. File-based sources are more specific,
// so they come first. `root` is the manifest scan root (git repo + deploy base).
func buildAttributor(cfg config.Config, root string) contract.Attributor {
	if root == "" {
		root = "."
	}
	attributors := []attribution.Attributor{
		attribution.NewGit(root),
		attribution.NewK8sDeploy(root),
	}
	for _, ic := range cfg.Issuers {
		if model.IssuerType(ic.Type) != model.IssuerKeycloak || ic.AdminCredentialEnv == "" {
			continue
		}
		user, pass := splitCreds(os.Getenv(ic.AdminCredentialEnv))
		ka, err := attribution.NewKeycloakAdmin(attribution.KeycloakOptions{
			RealmURL: ic.URL, AdminUser: user, AdminPassword: pass,
		})
		if err == nil {
			attributors = append(attributors, ka)
		}
	}
	return attribution.NewChain(attributors...)
}

// inClusterDiscoverers returns the live client-go discovery source when
// --in-cluster is set (KI-02). It errors if the Kubernetes client cannot be
// built, so a misconfigured cluster surfaces loudly rather than silently
// discovering nothing.
func inClusterDiscoverers(cmd *cobra.Command) ([]discovery.Discoverer, error) {
	if on, _ := cmd.Flags().GetBool("in-cluster"); !on {
		return nil, nil
	}
	kctx, _ := cmd.Flags().GetString("kube-context")
	client, err := kube.DynamicClient(kctx)
	if err != nil {
		return nil, err
	}
	return []discovery.Discoverer{istio.NewInCluster(client, kctx)}, nil
}

// allDiscoverers combines the file-based sources with any config-driven ones
// (loading the config file referenced by --config, if present) and the live
// in-cluster source when enabled.
func allDiscoverers(cmd *cobra.Command) ([]discovery.Discoverer, error) {
	ds := defaultDiscoverers()
	if cfg, err := config.Load(configPath(cmd)); err == nil {
		ds = append(ds, configDiscoverers(cfg)...)
	}
	inc, err := inClusterDiscoverers(cmd)
	if err != nil {
		return nil, err
	}
	return append(ds, inc...), nil
}

func splitCreds(s string) (user, pass string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func newDiscoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Derive the consumer inventory from cluster and config sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope := buildScope(cmd)
			inCluster, _ := cmd.Flags().GetBool("in-cluster")
			if len(scope.ConfigPaths) == 0 && !inCluster {
				return fmt.Errorf("no source given; pass --path (manifests) or --in-cluster (live Kubernetes API)")
			}
			ds, err := allDiscoverers(cmd)
			if err != nil {
				return err
			}
			consumers, err := discovery.Run(context.Background(), scope, ds...)
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
	cmd.Flags().String("kube-context", "", "kube-context for --in-cluster / cluster name used in StableIDs")
	cmd.Flags().Bool("in-cluster", false, "discover live Istio CRDs from the Kubernetes API (client-go)")
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
		return fmt.Errorf("no database configured — set KEYWAY_DB_URL, pass --db, or add db_url to config (use --db memory for an ephemeral store)")
	}
	scope := buildScope(cmd)
	// File-based sources no-op without --path; config-driven sources (Keycloak
	// client registry) run regardless, so always invoke discovery.
	ds, err := allDiscoverers(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()
	st, cleanup, err := open.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer cleanup()

	attrRoot := "."
	if len(scope.ConfigPaths) > 0 {
		attrRoot = scope.ConfigPaths[0]
	}
	cfg, _ := config.Load(configPath(cmd))
	libs, _ := libdefaults.Load()
	// Same use-case the HTTP API and scheduler run, so behavior can't drift.
	res, err := app.Deps{
		Store: st, Discoverers: ds, Scope: scope, Libs: libs,
		Attributor: buildAttributor(cfg, attrRoot),
	}.Snapshot(ctx, "manual")
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
