package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nometria/keyway/internal/blastradius"
	"github.com/nometria/keyway/internal/libdefaults"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/store/open"
	"github.com/spf13/cobra"
)

func newBlastRadiusCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "blast-radius", Short: "Compute who breaks under a proposed change"}

	rotate := &cobra.Command{
		Use:   "rotate-key",
		Short: "Blast radius of rotating a signing key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			issuer, _ := cmd.Flags().GetString("issuer")
			kid, _ := cmd.Flags().GetString("kid")
			return runBlastRadius(cmd, blastradius.ChangeProposal{Kind: blastradius.KindRotateKey, IssuerID: issuer, KID: kid})
		},
	}
	rotate.Flags().String("issuer", "", "issuer ID or name")
	rotate.Flags().String("kid", "", "key ID being rotated")
	rotate.Flags().Bool("verbose", false, "list ready and unknown consumers")
	rotate.Flags().String("output", "table", "output format: json|table")

	removeClaim := &cobra.Command{
		Use:   "remove-claim",
		Short: "Blast radius of removing a claim",
		RunE: func(cmd *cobra.Command, _ []string) error {
			issuer, _ := cmd.Flags().GetString("issuer")
			claim, _ := cmd.Flags().GetString("claim")
			return runBlastRadius(cmd, blastradius.ChangeProposal{Kind: blastradius.KindRemoveClaim, IssuerID: issuer, ClaimName: claim})
		},
	}
	removeClaim.Flags().String("issuer", "", "issuer ID or name")
	removeClaim.Flags().String("claim", "", "claim name being removed")
	removeClaim.Flags().Bool("verbose", false, "list ready and unknown consumers")
	removeClaim.Flags().String("output", "table", "output format: json|table")

	cmd.AddCommand(rotate, removeClaim)
	return cmd
}

func runBlastRadius(cmd *cobra.Command, proposal blastradius.ChangeProposal) error {
	dsn := dbURL(cmd)
	if dsn == "" {
		return fmt.Errorf("no database configured (set --db or KEYWAY_DB_URL)")
	}
	ctx := context.Background()
	st, cleanup, err := open.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer cleanup()

	v, err := st.LatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("no contract snapshot found — run `keyway snapshot` first: %w", err)
	}

	// Enrich consumers with library defaults so library-only signals count.
	if db, derr := libdefaults.Load(); derr == nil {
		for i := range v.Consumers {
			db.Enrich(&v.Consumers[i])
		}
	}

	// Gather recent probe history per consumer.
	history := map[string][]model.ProbeResult{}
	for _, c := range v.Consumers {
		if h, herr := st.ProbeHistory(ctx, c.StableID, 50); herr == nil && len(h) > 0 {
			history[c.StableID] = h
		}
	}

	res, err := blastradius.Resolve(v, proposal, history, time.Now())
	if err != nil {
		return err
	}

	if out, _ := cmd.Flags().GetString("output"); out == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	verbose, _ := cmd.Flags().GetBool("verbose")
	printBlastRadius(cmd, res, verbose)
	return nil
}

func printBlastRadius(cmd *cobra.Command, res blastradius.BlastRadiusResult, verbose bool) {
	out := cmd.OutOrStdout()
	var willBreak, ready, unknown []blastradius.AffectedConsumer
	for _, a := range res.Affected {
		switch a.Verdict {
		case blastradius.VerdictWillBreak:
			willBreak = append(willBreak, a)
		case blastradius.VerdictReady:
			ready = append(ready, a)
		default:
			unknown = append(unknown, a)
		}
	}

	fmt.Fprintf(out, "\n%s affects %d consumers.\n\n", describeProposal(res.Proposal), len(res.Affected))

	if len(willBreak) > 0 {
		fmt.Fprintf(out, "WILL BREAK (%d)\n", len(willBreak))
		for _, a := range willBreak {
			fmt.Fprintf(out, "  %-22s %s   %v\n", a.Consumer.Name, a.Reason, a.Evidence)
			if a.Consumer.OwnerTeam != "" {
				fmt.Fprintf(out, "  %-22s owner: %s\n", "", a.Consumer.OwnerTeam)
			}
		}
		fmt.Fprintln(out)
	}

	fmt.Fprintf(out, "READY (%d)", len(ready))
	if verbose {
		fmt.Fprintln(out)
		for _, a := range ready {
			fmt.Fprintf(out, "  %-22s %s\n", a.Consumer.Name, a.Reason)
		}
	} else if len(ready) > 0 {
		fmt.Fprint(out, "   run with --verbose to list")
	}
	fmt.Fprintln(out)

	fmt.Fprintf(out, "UNKNOWN (%d)", len(unknown))
	if len(unknown) > 0 {
		fmt.Fprint(out, "  insufficient evidence — not probeable")
	}
	fmt.Fprintln(out)

	if res.RecommendedGracePeriod > 0 {
		fmt.Fprintf(out, "\nRECOMMENDED GRACE PERIOD: %s\n", humanDuration(res.RecommendedGracePeriod))
		if res.GraceBasis != "" {
			fmt.Fprintf(out, "  bound by %s\n", res.GraceBasis)
		}
	}
	if len(res.Unknown) > 0 {
		fmt.Fprintf(out, "  NOTE: %d consumer(s) unknown — treat as a lower bound.\n", len(res.Unknown))
	}
}

func describeProposal(p blastradius.ChangeProposal) string {
	switch p.Kind {
	case blastradius.KindRotateKey:
		return fmt.Sprintf("Rotating %s on %s", p.KID, p.IssuerID)
	case blastradius.KindRemoveClaim:
		return fmt.Sprintf("Removing claim %q", p.ClaimName)
	case blastradius.KindChangeIssuer:
		return fmt.Sprintf("Changing issuer to %s", p.NewIssuerURL)
	case blastradius.KindDropAlgorithm:
		return fmt.Sprintf("Dropping algorithm %s", p.Algorithm)
	default:
		return p.Kind
	}
}

func humanDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours > 0 && mins > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}
