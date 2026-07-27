package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/store/open"
)

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Summarize recent contract changes",
		RunE:  runReport,
	}
	cmd.Flags().String("since", "7d", "time window, e.g. 7d or 24h")
	cmd.Flags().String("format", "md", "output format: json|md")
	return cmd
}

func runReport(cmd *cobra.Command, _ []string) error {
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

	sinceStr, _ := cmd.Flags().GetString("since")
	events, err := st.ListChangeEvents(ctx, parseSinceDur(sinceStr))
	if err != nil {
		return err
	}

	if format, _ := cmd.Flags().GetString("format"); format == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(events)
	}
	printReportMarkdown(cmd, events, sinceStr)
	return nil
}

func printReportMarkdown(cmd *cobra.Command, events []model.ChangeEvent, since string) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "# Keyway change report (last %s)\n\n", since)
	if len(events) == 0 {
		fmt.Fprintln(out, "No changes in this window. ✅")
		return
	}
	bySev := map[model.Severity][]model.ChangeEvent{}
	for _, e := range events {
		bySev[e.Severity] = append(bySev[e.Severity], e)
	}
	for _, sev := range []model.Severity{model.SeverityCritical, model.SeverityHigh, model.SeverityMedium, model.SeverityLow, model.SeverityInfo} {
		list := bySev[sev]
		if len(list) == 0 {
			continue
		}
		fmt.Fprintf(out, "## %s (%d)\n\n", sev, len(list))
		for _, e := range list {
			attr := ""
			if e.Attribution != nil && e.Attribution.Kind != "unattributed" {
				attr = fmt.Sprintf(" — by %s (%s)", e.Attribution.Actor, short(e.Attribution.Ref))
			}
			fmt.Fprintf(out, "- **%s** `%s` %s: `%v` → `%v` [%s]%s\n",
				e.Class, e.ConsumerID, e.Field, e.OldValue, e.NewValue, e.DetectedAt.Format(time.DateOnly), attr)
		}
		fmt.Fprintln(out)
	}
}

// parseSinceDur mirrors the API's since parsing (supports a 7d shorthand).
func parseSinceDur(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d)
	}
	if n := len(s); n > 1 && s[n-1] == 'd' {
		if hours, err := time.ParseDuration(s[:n-1] + "h"); err == nil {
			return time.Now().Add(-hours * 24)
		}
	}
	return time.Time{}
}
