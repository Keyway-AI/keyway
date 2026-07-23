package cli

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/architsharma/keyway/internal/discovery"
	"github.com/architsharma/keyway/internal/issuer/generic"
	"github.com/architsharma/keyway/internal/model"
	"github.com/architsharma/keyway/internal/probe"
	"github.com/architsharma/keyway/internal/store/postgres"
	"github.com/spf13/cobra"
)

func newProbeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "probe",
		Short: "Run verification probes against probeable consumers",
		RunE:  runProbe,
	}
	cmd.Flags().String("consumer", "", "limit to a single consumer StableID")
	cmd.Flags().String("probe", "", "limit to a single probe ID")
	cmd.Flags().Bool("dry-run", false, "construct tokens but do not send requests")
	cmd.Flags().Bool("i-know-this-is-production", false, "override the staging-only guard (dangerous)")
	cmd.Flags().StringSlice("allow", nil, "host substrings the probe engine may target (staging allowlist)")
	cmd.Flags().StringSlice("path", nil, "manifest files/directories to discover consumers from (else uses latest snapshot)")
	cmd.Flags().String("issuer-url", "", "issuer URL to mint synthetic tokens with (Keyway-operated key)")
	cmd.Flags().StringSlice("namespace", nil, "Kubernetes namespaces to scan")
	cmd.Flags().String("kube-context", "", "cluster name used in StableIDs")
	return cmd
}

func runProbe(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	consumers, err := loadConsumers(cmd)
	if err != nil {
		return err
	}
	if len(consumers) == 0 {
		return fmt.Errorf("no consumers to probe (use --path or take a snapshot first)")
	}

	if only, _ := cmd.Flags().GetString("consumer"); only != "" {
		consumers = filterConsumers(consumers, only)
	}

	issuerURL, _ := cmd.Flags().GetString("issuer-url")
	if issuerURL == "" {
		issuerURL = firstIssuer(consumers)
	}
	iss, err := generic.New(generic.Options{Name: "keyway", IssuerURL: issuerURL})
	if err != nil {
		return err
	}
	issModel, err := iss.Describe(ctx)
	if err != nil {
		return err
	}

	allow, _ := cmd.Flags().GetStringSlice("allow")
	prod, _ := cmd.Flags().GetBool("i-know-this-is-production")
	dry, _ := cmd.Flags().GetBool("dry-run")
	cfg := probe.DefaultEngineConfig()
	cfg.Allowlist = allow
	cfg.AllowProduction = prod
	cfg.DryRun = dry
	eng := probe.NewEngine(cfg)

	results, outcomes, err := eng.Run(ctx, issModel, func(kid string, claims map[string]any) (string, error) {
		return iss.MintToken(ctx, kid, claims)
	}, consumers)
	if err != nil {
		return err
	}

	if onlyProbe, _ := cmd.Flags().GetString("probe"); onlyProbe != "" {
		results = filterResults(results, onlyProbe)
	}

	printProbeResults(cmd, results, outcomes)

	// Persist results unless dry-run or no DB configured.
	if !dry {
		if dsn := dbURL(cmd); dsn != "" {
			st, oerr := postgres.Open(ctx, dsn)
			if oerr == nil {
				defer st.Close()
				if serr := st.SaveProbeResults(ctx, results); serr != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "warning: could not persist probe results:", serr)
				}
			}
		}
	}
	return nil
}

func loadConsumers(cmd *cobra.Command) ([]model.Consumer, error) {
	paths, _ := cmd.Flags().GetStringSlice("path")
	if len(paths) > 0 {
		scope := buildScope(cmd)
		return discovery.Run(context.Background(), scope, defaultDiscoverers()...)
	}
	// Fall back to the latest snapshot in the store.
	dsn := dbURL(cmd)
	if dsn == "" {
		return nil, nil
	}
	st, err := postgres.Open(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	v, err := st.LatestVersion(context.Background())
	if err != nil {
		return nil, nil
	}
	return v.Consumers, nil
}

func printProbeResults(cmd *cobra.Command, results []model.ProbeResult, outcomes []probe.ConsumerOutcome) {
	out := cmd.OutOrStdout()
	for _, o := range outcomes {
		if o.Skipped {
			fmt.Fprintf(out, "SKIP %s — %s\n", o.ConsumerID, o.Reason)
		}
	}
	if len(results) == 0 {
		fmt.Fprintln(out, "No probe results.")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "CONSUMER\tPROBE\tSTATUS\tRESULT\tLATENCY")
	fails := 0
	for _, r := range results {
		verdict := "pass"
		switch {
		case r.RawResponse == "dry-run":
			verdict = "dry"
		case !r.Passed:
			verdict = "FAIL"
			fails++
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%dms\n", short(r.ConsumerID), r.ProbeID, r.StatusCode, verdict, r.LatencyMs)
	}
	_ = tw.Flush()
	fmt.Fprintf(out, "\n%d probe(s), %d finding(s) at %s\n", len(results), fails, time.Now().Format(time.Kitchen))
}

func filterConsumers(cs []model.Consumer, stableID string) []model.Consumer {
	var out []model.Consumer
	for _, c := range cs {
		if c.StableID == stableID {
			out = append(out, c)
		}
	}
	return out
}

func filterResults(rs []model.ProbeResult, probeID string) []model.ProbeResult {
	var out []model.ProbeResult
	for _, r := range rs {
		if r.ProbeID == probeID {
			out = append(out, r)
		}
	}
	return out
}

func firstIssuer(cs []model.Consumer) string {
	for _, c := range cs {
		if len(c.Expects.Issuers) > 0 {
			return c.Expects.Issuers[0]
		}
	}
	return "https://issuer.keyway.local"
}
