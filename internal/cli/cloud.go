package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Keyway-AI/keyway/cloud"
	"github.com/Keyway-AI/keyway/internal/model"
)

// severityRank orders change severities so --fail-on can gate on a threshold.
var severityRank = map[string]int{
	"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4,
}

func newCloudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Analyze auth config in CI and report to Keyway Cloud (hosted or self-hosted)",
		Long: "Run Keyway's static analysis (contract discovery + drift) from your repo or CI.\n\n" +
			"With --server it uploads the config to a Keyway Cloud API — the hosted service or\n" +
			"your own self-hosted keyway-cloud — for history and a shared report. Without a\n" +
			"server it runs fully offline, diffing against a committed baseline file, so the\n" +
			"open-source path needs no account and no network.",
	}
	cmd.AddCommand(newCloudAnalyzeCmd())
	return cmd
}

func newCloudAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Discover the token contract from config and report drift (CI-friendly)",
		Long: "Reads YAML manifests under --path, derives the token contract, and reports drift.\n\n" +
			"Modes:\n" +
			"  hosted/self-hosted  --server <url> --token <tok> --project <id>\n" +
			"                      uploads the config; the server diffs vs the last run and stores history\n" +
			"  local (offline)     (no --server) diffs vs --baseline and can --write-baseline\n\n" +
			"Exits non-zero when a change at or above --fail-on is detected, so it gates a PR.",
		RunE: runCloudAnalyze,
	}
	f := cmd.Flags()
	f.StringSlice("path", []string{"."}, "manifest files or directories to scan")
	f.String("server", os.Getenv("KEYWAY_CLOUD_URL"), "Keyway Cloud API base URL (env KEYWAY_CLOUD_URL); empty = offline local mode")
	f.String("token", os.Getenv("KEYWAY_TOKEN"), "API token for --server (env KEYWAY_TOKEN)")
	f.String("project", os.Getenv("KEYWAY_PROJECT"), "project id on the server (env KEYWAY_PROJECT)")
	f.String("baseline", "", "local mode: baseline contract file to diff against (JSON)")
	f.String("write-baseline", "", "local mode: write the derived contract to this file (create/update the baseline)")
	f.String("fail-on", "high", "exit non-zero if any change is at or above this severity: none|low|medium|high|critical")
	f.Bool("json", false, "emit the analysis as JSON")
	return cmd
}

func runCloudAnalyze(cmd *cobra.Command, _ []string) error {
	paths, _ := cmd.Flags().GetStringSlice("path")
	server, _ := cmd.Flags().GetString("server")
	asJSON, _ := cmd.Flags().GetBool("json")
	failOn, _ := cmd.Flags().GetString("fail-on")

	manifests, err := collectManifests(paths)
	if err != nil {
		return err
	}
	if len(manifests) == 0 {
		return fmt.Errorf("no YAML manifests found under %s", strings.Join(paths, ", "))
	}

	var (
		version     model.ContractVersion
		changes     []model.ChangeEvent
		baselineNew bool
	)
	if server != "" {
		version, changes, err = analyzeViaServer(cmd, server, manifests)
	} else {
		version, changes, baselineNew, err = analyzeLocal(cmd, manifests)
	}
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{
			"hash": version.Hash, "consumers": len(version.Consumers),
			"issuers": len(version.Issuers), "changes": changes,
		}); err != nil {
			return err
		}
	} else {
		printAnalysis(cmd, version, changes, baselineNew)
	}

	return gate(changes, failOn)
}

// collectManifests walks the given paths for YAML manifests, returning a
// repo-relative-path → content map suitable for the analyze API and engine.
func collectManifests(paths []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("path %q: %w", p, err)
		}
		if !info.IsDir() {
			if isYAML(p) {
				b, err := os.ReadFile(p) // #nosec G304 -- user-supplied config path, read-only
				if err != nil {
					return nil, err
				}
				out[filepath.Base(p)] = string(b)
			}
			continue
		}
		err = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if skipDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isYAML(path) {
				return nil
			}
			// #nosec G304 G122 -- reading the user's own repo config in their CI, read-only; no privilege boundary to TOCTOU across.
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, relErr := filepath.Rel(p, path)
			if relErr != nil {
				rel = path
			}
			out[filepath.ToSlash(rel)] = string(b)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func isYAML(p string) bool {
	l := strings.ToLower(p)
	return strings.HasSuffix(l, ".yaml") || strings.HasSuffix(l, ".yml")
}

func skipDir(name string) bool {
	switch name {
	case "node_modules", "vendor", ".git":
		return true
	}
	return false
}

// analyzeLocal runs the engine offline and diffs against an optional baseline
// file, optionally (re)writing it. Returns whether a new baseline was written.
func analyzeLocal(cmd *cobra.Command, manifests map[string]string) (model.ContractVersion, []model.ChangeEvent, bool, error) {
	baselinePath, _ := cmd.Flags().GetString("baseline")
	writePath, _ := cmd.Flags().GetString("write-baseline")

	var prev *model.ContractVersion
	if baselinePath != "" {
		b, err := os.ReadFile(baselinePath) // #nosec G304 -- user-supplied baseline path
		switch {
		case err == nil:
			var v model.ContractVersion
			if jerr := json.Unmarshal(b, &v); jerr != nil {
				return model.ContractVersion{}, nil, false, fmt.Errorf("baseline %q: %w", baselinePath, jerr)
			}
			prev = &v
		case os.IsNotExist(err):
			// First run: no baseline yet — treated as a baseline establishment.
		default:
			return model.ContractVersion{}, nil, false, err
		}
	}

	version, changes, err := cloud.Analyze(context.Background(), manifests, prev)
	if err != nil {
		return model.ContractVersion{}, nil, false, err
	}

	wrote := false
	if writePath != "" {
		b, mErr := json.MarshalIndent(version, "", "  ")
		if mErr != nil {
			return model.ContractVersion{}, nil, false, mErr
		}
		if err := os.WriteFile(writePath, b, 0o600); err != nil {
			return model.ContractVersion{}, nil, false, err
		}
		wrote = true
	}
	return version, changes, wrote, nil
}

// analyzeViaServer uploads manifests to a Keyway Cloud API and returns the
// server-computed contract + drift.
func analyzeViaServer(cmd *cobra.Command, server string, manifests map[string]string) (model.ContractVersion, []model.ChangeEvent, error) {
	token, _ := cmd.Flags().GetString("token")
	project, _ := cmd.Flags().GetString("project")
	if token == "" {
		return model.ContractVersion{}, nil, fmt.Errorf("--token (or KEYWAY_TOKEN) is required with --server")
	}
	if project == "" {
		return model.ContractVersion{}, nil, fmt.Errorf("--project (or KEYWAY_PROJECT) is required with --server")
	}

	body, _ := json.Marshal(map[string]any{"manifests": manifests})
	url := strings.TrimRight(server, "/") + "/v1/projects/" + project + "/analyze"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return model.ContractVersion{}, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 60 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return model.ContractVersion{}, nil, fmt.Errorf("reaching %s: %w", server, err)
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(payload))
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(payload, &e) == nil && e.Error != "" {
			msg = e.Error
		}
		return model.ContractVersion{}, nil, fmt.Errorf("server returned %d: %s", res.StatusCode, msg)
	}

	var a struct {
		Version model.ContractVersion `json:"version"`
		Changes []model.ChangeEvent   `json:"changes"`
	}
	if err := json.Unmarshal(payload, &a); err != nil {
		return model.ContractVersion{}, nil, fmt.Errorf("decoding server response: %w", err)
	}
	return a.Version, a.Changes, nil
}

func printAnalysis(cmd *cobra.Command, v model.ContractVersion, changes []model.ChangeEvent, baselineNew bool) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Contract %s — %d consumer(s), %d issuer(s)\n",
		short(v.Hash), len(v.Consumers), len(v.Issuers))
	switch {
	case baselineNew && len(changes) == 0:
		fmt.Fprintln(out, "Baseline written. No previous contract to compare.")
	case len(changes) == 0:
		fmt.Fprintln(out, "No drift — the contract held. ✓")
	default:
		fmt.Fprintf(out, "\n%d change(s):\n", len(changes))
		for _, e := range sortedChanges(changes) {
			fmt.Fprintf(out, "  [%s/%s] %s %s: %v → %v\n",
				e.Severity, e.Class, shortConsumerID(e.ConsumerID), e.Field, e.OldValue, e.NewValue)
		}
	}
}

// gate returns a non-nil error (non-zero exit) when any change meets the
// --fail-on severity threshold. "none" never fails.
func gate(changes []model.ChangeEvent, failOn string) error {
	failOn = strings.ToLower(strings.TrimSpace(failOn))
	if failOn == "" || failOn == "none" {
		return nil
	}
	threshold, ok := severityRank[failOn]
	if !ok {
		return fmt.Errorf("invalid --fail-on %q (use none|low|medium|high|critical)", failOn)
	}
	worst, worstName := -1, ""
	for _, e := range changes {
		if r := severityRank[strings.ToLower(string(e.Severity))]; r > worst {
			worst, worstName = r, strings.ToLower(string(e.Severity))
		}
	}
	if worst >= threshold {
		return fmt.Errorf("drift at severity %q meets the --fail-on %q gate", worstName, failOn)
	}
	return nil
}

func sortedChanges(changes []model.ChangeEvent) []model.ChangeEvent {
	out := append([]model.ChangeEvent(nil), changes...)
	sort.SliceStable(out, func(i, j int) bool {
		return severityRank[strings.ToLower(string(out[i].Severity))] > severityRank[strings.ToLower(string(out[j].Severity))]
	})
	return out
}

func shortConsumerID(id string) string {
	parts := strings.Split(strings.Trim(id, "/"), "/")
	if n := len(parts); n >= 2 {
		return strings.Join(parts[n-2:], "/")
	}
	return id
}
