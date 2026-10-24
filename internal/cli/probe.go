package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/issuer/generic"
	"github.com/Keyway-AI/keyway/internal/issuer/localkeys"
	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/probe"
	"github.com/Keyway-AI/keyway/internal/store/open"
)

func newProbeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "probe",
		Short: "Run verification probes against probeable consumers",
		RunE:  runProbe,
	}
	cmd.Flags().String("consumer", "", "limit to a single consumer StableID")
	cmd.Flags().String("probe", "", "limit to a single probe ID")
	cmd.Flags().Bool("harness", false, "run the generative attack harness (internal/attack) instead of the fixed probe set")
	cmd.Flags().Bool("dry-run", false, "construct tokens but do not send requests")
	cmd.Flags().Bool("i-know-this-is-production", false, "override the staging-only guard (dangerous)")
	cmd.Flags().StringSlice("allow", nil, "host substrings the probe engine may target (staging allowlist)")
	cmd.Flags().StringSlice("path", nil, "manifest files/directories to discover consumers from (else uses latest snapshot)")
	cmd.Flags().String("consumers", "", "JSON file of model.Consumer targets to probe directly (for staging verifiers not discovered from infra; see bench/verifiers/)")
	cmd.Flags().String("issuer-url", "", "issuer URL to mint synthetic tokens with (Keyway-operated key)")
	cmd.Flags().String("issuer-key", "", "JSON file of persisted keys to mint with (localkeys.Export format), so the harness shares one key with a standalone JWKS server; else a fresh key is generated")
	cmd.Flags().String("serve-jwks", "", "also serve the issuer's JWKS + OIDC discovery at this address (e.g. :8099) so real verifier targets can validate signatures against the same key the harness mints with")
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
	genOpts := generic.Options{Name: "keyway", IssuerURL: issuerURL}
	if kf, _ := cmd.Flags().GetString("issuer-key"); kf != "" {
		keys, kerr := loadIssuerKeys(kf)
		if kerr != nil {
			return kerr
		}
		genOpts.ImportKeys = keys
	}
	iss, err := generic.New(genOpts)
	if err != nil {
		return err
	}
	issModel, err := iss.Describe(ctx)
	if err != nil {
		return err
	}

	if addr, _ := cmd.Flags().GetString("serve-jwks"); addr != "" {
		if serr := serveIssuerJWKS(cmd, addr, issuerURL, iss); serr != nil {
			return serr
		}
	}

	allow, _ := cmd.Flags().GetStringSlice("allow")
	prod, _ := cmd.Flags().GetBool("i-know-this-is-production")
	dry, _ := cmd.Flags().GetBool("dry-run")
	cfg := probe.DefaultEngineConfig()
	cfg.Allowlist = allow
	cfg.AllowProduction = prod
	cfg.DryRun = dry
	eng := probe.NewEngine(cfg)

	mint := func(kid string, claims map[string]any) (string, error) {
		return iss.MintToken(ctx, kid, claims)
	}
	runFn := eng.Run
	if harness, _ := cmd.Flags().GetBool("harness"); harness {
		runFn = eng.RunHarness
	}
	results, outcomes, err := runFn(ctx, issModel, mint, consumers)
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
			st, cleanup, oerr := open.Open(ctx, dsn)
			if oerr == nil {
				defer cleanup()
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
	if manifest, _ := cmd.Flags().GetString("consumers"); manifest != "" {
		return loadConsumerManifest(manifest)
	}
	if len(paths) > 0 {
		scope := buildScope(cmd)
		return discovery.Run(context.Background(), scope, defaultDiscoverers()...)
	}
	// Fall back to the latest snapshot in the store.
	dsn := dbURL(cmd)
	if dsn == "" {
		return nil, nil
	}
	st, cleanup, err := open.Open(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	v, err := st.LatestVersion(context.Background())
	if err != nil {
		return nil, nil
	}
	return v.Consumers, nil
}

// loadIssuerKeys reads a localkeys.Export JSON file (array of PersistedKey) so the
// harness mints with a key shared with a standalone JWKS server.
func loadIssuerKeys(path string) ([]localkeys.PersistedKey, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- operator-supplied eval key file
	if err != nil {
		return nil, fmt.Errorf("read issuer-key: %w", err)
	}
	var keys []localkeys.PersistedKey
	if err := json.Unmarshal(raw, &keys); err != nil {
		return nil, fmt.Errorf("parse issuer-key %s: %w", path, err)
	}
	return keys, nil
}

// serveIssuerJWKS starts a background HTTP server exposing the harness issuer's public
// JWKS and OIDC discovery, so real verifier targets can validate the signatures of the
// tokens the harness mints (both use the same in-process key). Staging/eval only.
func serveIssuerJWKS(cmd *cobra.Command, addr, issuerURL string, iss *generic.Adapter) error {
	ks := iss.KeySet()
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ks.JWKS())
	})
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuerURL,
			"jwks_uri":                              strings.TrimRight(issuerURL, "/") + "/.well-known/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("serve-jwks: %w", err)
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	fmt.Fprintf(cmd.ErrOrStderr(), "serving issuer JWKS at http://%s/.well-known/jwks.json (issuer=%s)\n", addr, issuerURL)
	return nil
}

// loadConsumerManifest reads staging verifier targets from a JSON file. The file is
// either a JSON array of model.Consumer or an object with a "consumers" array. This is
// the path for probing reference verifiers (MCP servers, OAuth middleware) that have no
// Istio/Envoy/k8s config to discover. See bench/verifiers/ for the format and examples.
func loadConsumerManifest(path string) ([]model.Consumer, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- operator-supplied staging target manifest
	if err != nil {
		return nil, fmt.Errorf("read consumers manifest: %w", err)
	}
	var arr []model.Consumer
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	var wrapped struct {
		Consumers []model.Consumer `json:"consumers"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("parse consumers manifest %s: %w", path, err)
	}
	return wrapped.Consumers, nil
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
