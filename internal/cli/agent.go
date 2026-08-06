package cli

import (
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
			"(RFC 8707/9728), the delegation act claim (RFC 8693), scope minimization, and\n" +
			"expiry. Each finding maps to a threat in `keyway threats coverage --domain agent`.\n\n" +
			"Pass the token via --token or on stdin.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, _ := cmd.Flags().GetString("token")
			if token == "" {
				b, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				token = strings.TrimSpace(string(b))
			}
			if token == "" {
				return fmt.Errorf("no token provided (use --token or pipe it on stdin)")
			}

			aud, _ := cmd.Flags().GetString("audience")
			reqDel, _ := cmd.Flags().GetBool("require-delegation")
			maxLife, _ := cmd.Flags().GetDuration("max-lifetime")
			scopes, _ := cmd.Flags().GetStringSlice("allowed-scopes")

			findings, err := agentauth.Analyze(token, agentauth.Policy{
				Audience: aud, RequireDelegation: reqDel, MaxLifetime: maxLife,
				AllowedScopes: scopes, Now: time.Now(),
			})
			if err != nil {
				return err
			}

			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
					"findings": findings, "count": len(findings),
				})
			}

			out := cmd.OutOrStdout()
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
			return nil
		},
	}
	cmd.Flags().String("token", "", "the JWT to inspect (default: read from stdin)")
	cmd.Flags().String("audience", "", "expected resource URI/audience the token must be bound to")
	cmd.Flags().Bool("require-delegation", false, "require the delegation `act` claim (on-behalf-of tokens)")
	cmd.Flags().Duration("max-lifetime", 0, "flag tokens whose lifetime exceeds this (e.g. 1h)")
	cmd.Flags().StringSlice("allowed-scopes", nil, "scopes the token may carry; anything else is flagged")
	return cmd
}
