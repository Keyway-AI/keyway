package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/Keyway-AI/keyway/internal/agentauth"
)

// newAgentCmd groups agent-auth commands. Today it statically inspects an
// agent/MCP/on-behalf-of token against the agent-auth invariants (the "agent"
// domain of `keyway threats`).
func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Inspect and verify AI-agent / MCP auth tokens and contracts",
	}
	cmd.AddCommand(newAgentInspectCmd())
	return cmd
}

func newAgentInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Statically check an agent/MCP/OBO token against the agent-auth invariants",
		Long: "Inspects a JWT's own hygiene (no signature verification): audience binding\n" +
			"(RFC 8707/9728), the delegation act claim and chain shape (RFC 8693), scope\n" +
			"minimization, and expiry. Each finding maps to a threat in\n" +
			"`keyway threats coverage --domain agent`.\n\n" +
			"Pass the token via --token or on stdin, or run with --demo to see it work on a\n" +
			"built-in sample token. With --fail-on it exits non-zero when a finding meets\n" +
			"that severity, so it gates CI.",
		// A gating failure is a real result, not a usage mistake — don't print help.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			demo, _ := cmd.Flags().GetBool("demo")

			token, _ := cmd.Flags().GetString("token")
			aud, _ := cmd.Flags().GetString("audience")
			reqDel, _ := cmd.Flags().GetBool("require-delegation")
			maxLife, _ := cmd.Flags().GetDuration("max-lifetime")
			scopes, _ := cmd.Flags().GetStringSlice("allowed-scopes")
			maxDepth, _ := cmd.Flags().GetInt("max-delegation-depth")

			switch {
			case demo:
				// Note on stderr so --json keeps a clean stdout.
				ew := cmd.ErrOrStderr()
				fmt.Fprintln(ew, "Inspecting a built-in, deliberately-insecure sample token — no real token or setup needed.")
				fmt.Fprintln(ew, "Run it on your own token:  echo \"$TOKEN\" | keyway agent inspect --audience <resource>")
				fmt.Fprintln(ew)
				token = demoToken()
				aud, reqDel = "https://mcp.example/api", true
			case token == "":
				b, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				token = strings.TrimSpace(string(b))
			}
			if token == "" {
				return fmt.Errorf("no token provided (use --token, pipe it on stdin, or try --demo)")
			}

			findings, err := agentauth.Analyze(token, agentauth.Policy{
				Audience: aud, RequireDelegation: reqDel, MaxLifetime: maxLife,
				AllowedScopes: scopes, MaxDelegationDepth: maxDepth, Now: time.Now(),
			})
			if err != nil {
				return err
			}

			failOn, _ := cmd.Flags().GetString("fail-on")
			if demo {
				failOn = "none" // a demo should never exit non-zero
			}

			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				if err := json.NewEncoder(out).Encode(map[string]any{
					"findings": findings, "count": len(findings),
				}); err != nil {
					return err
				}
				return gateFindings(findings, failOn)
			}

			if len(findings) == 0 {
				fmt.Fprintln(out, "✓ no agent-auth findings for this token")
				return nil
			}
			tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "THREAT\tSEVERITY\tFINDING")
			for _, f := range findings {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", f.ThreatID, f.Severity, f.Message)
			}
			_ = tw.Flush()
			fmt.Fprintf(out, "\n%d finding(s). See `keyway threats coverage --domain agent`.\n", len(findings))
			return gateFindings(findings, failOn)
		},
	}
	cmd.Flags().String("token", "", "the JWT to inspect (default: read from stdin)")
	cmd.Flags().String("audience", "", "expected resource URI/audience the token must be bound to")
	cmd.Flags().Bool("require-delegation", false, "require the delegation `act` claim (on-behalf-of tokens)")
	cmd.Flags().Duration("max-lifetime", 0, "flag tokens whose lifetime exceeds this (e.g. 1h)")
	cmd.Flags().StringSlice("allowed-scopes", nil, "scopes the token may carry; anything else is flagged")
	cmd.Flags().Int("max-delegation-depth", 0, "flag act delegation chains deeper than this (0 = only flag malformed chains)")
	cmd.Flags().String("fail-on", "none", "exit non-zero if any finding is at or above this severity: none|low|medium|high|critical")
	cmd.Flags().Bool("demo", false, "inspect a built-in, deliberately-insecure sample token (no token or setup needed)")
	return cmd
}

// demoToken builds a deliberately-insecure sample JWT (unsigned) so `--demo` shows a
// spread of findings with zero input: no audience (MCP-02), an omnibus scope
// (SCOPE-01), no expiry (SCOPE-02), and a malformed act chain — a link with no sub
// (DEL-02). It is obviously not a real credential.
func demoToken() string {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	pay := base64.RawURLEncoding.EncodeToString([]byte(
		`{"sub":"agent-42","scope":"admin:*","act":{"act":{"sub":"upstream"}}}`))
	return hdr + "." + pay + "."
}

// gateFindings returns a non-zero (error) result when any finding meets the
// --fail-on severity threshold, so `keyway agent inspect` can gate CI. It reuses
// the same severity ordering as `keyway cloud analyze`.
func gateFindings(findings []agentauth.Finding, failOn string) error {
	failOn = strings.ToLower(strings.TrimSpace(failOn))
	if failOn == "" || failOn == "none" {
		return nil
	}
	threshold, ok := severityRank[failOn]
	if !ok {
		return fmt.Errorf("invalid --fail-on %q (use none|low|medium|high|critical)", failOn)
	}
	for _, f := range findings {
		if severityRank[strings.ToLower(string(f.Severity))] >= threshold {
			return fmt.Errorf("agent-auth findings at or above %q severity", failOn)
		}
	}
	return nil
}
