package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Keyway-AI/keyway/internal/threats"
)

// newThreatsCmd exposes the threat taxonomy (JWT/JWKS/OIDC and AI-agent auth) and
// Keyway's measured coverage of it. This is the honest denominator: what fraction
// of the documented threat universe Keyway actually detects, with every gap named
// and cited.
func newThreatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "threats",
		Short: "Show the threat taxonomy (JWT + agent auth) and Keyway's coverage of it",
	}
	cmd.AddCommand(newThreatsCoverageCmd())
	return cmd
}

func newThreatsCoverageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Report detection coverage against the documented threat taxonomy",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cat := threats.Catalog()
			if d, _ := cmd.Flags().GetString("domain"); d != "" {
				cat = filterDomain(cat, threats.Domain(d))
				if len(cat) == 0 {
					return fmt.Errorf("no threats in domain %q (want jwt or agent)", d)
				}
			}
			report := threats.Compute(cat)
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(coverageJSON(cat, report))
			}
			fmt.Fprint(cmd.OutOrStdout(), report.Markdown("Regenerate with `make coverage` (`keyway threats coverage`)."))
			return nil
		},
	}
	cmd.Flags().String("domain", "", "limit to a threat domain: jwt or agent")
	return cmd
}

func filterDomain(cat []threats.Threat, d threats.Domain) []threats.Threat {
	var out []threats.Threat
	for _, t := range cat {
		if t.Domain == d {
			out = append(out, t)
		}
	}
	return out
}

// coverageJSON is the machine-readable shape (for CI gates / dashboards).
func coverageJSON(cat []threats.Threat, r threats.Report) map[string]any {
	cats := make([]map[string]any, 0, len(r.Categories))
	for _, c := range r.Categories {
		cats = append(cats, map[string]any{"category": c.Category, "covered": c.Covered, "total": c.Total})
	}
	doms := make([]map[string]any, 0, len(r.Domains))
	for _, d := range r.Domains {
		doms = append(doms, map[string]any{"domain": d.Domain, "covered": d.Covered, "total": d.Total, "percent": d.Pct()})
	}
	return map[string]any{
		"total":      r.Total,
		"covered":    r.Covered,
		"gaps":       len(r.Gaps),
		"percent":    r.Pct(),
		"domains":    doms,
		"categories": cats,
		"threats":    cat,
	}
}
