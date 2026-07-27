package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nometria/keyway/internal/threats"
)

// newThreatsCmd exposes the JWT/JWKS/OIDC threat taxonomy and Keyway's measured
// coverage of it. This is the honest denominator: what fraction of the documented
// threat universe Keyway actually detects, with every gap named and cited.
func newThreatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "threats",
		Short: "Show the JWT threat taxonomy and Keyway's coverage of it",
	}
	cmd.AddCommand(newThreatsCoverageCmd())
	return cmd
}

func newThreatsCoverageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Report detection coverage against the documented JWT threat taxonomy",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cat := threats.Catalog()
			report := threats.Compute(cat)
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(coverageJSON(cat, report))
			}
			fmt.Fprint(cmd.OutOrStdout(), report.Markdown("Regenerate with `keyway threats coverage`."))
			return nil
		},
	}
	return cmd
}

// coverageJSON is the machine-readable shape (for CI gates / dashboards).
func coverageJSON(cat []threats.Threat, r threats.Report) map[string]any {
	cats := make([]map[string]any, 0, len(r.Categories))
	for _, c := range r.Categories {
		cats = append(cats, map[string]any{"category": c.Category, "covered": c.Covered, "total": c.Total})
	}
	return map[string]any{
		"total":      r.Total,
		"covered":    r.Covered,
		"gaps":       len(r.Gaps),
		"percent":    r.Pct(),
		"categories": cats,
		"threats":    cat,
	}
}
