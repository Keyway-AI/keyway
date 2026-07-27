package threats

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nometria/keyway/internal/model"
)

// severityRank orders severities most-severe first for reporting.
var severityRank = map[model.Severity]int{
	model.SeverityCritical: 0,
	model.SeverityHigh:     1,
	model.SeverityMedium:   2,
	model.SeverityLow:      3,
	model.SeverityInfo:     4,
}

// categoryOrder is the display order of categories in the report.
var categoryOrder = []Category{
	CatSignature, CatAlgorithm, CatHeaderKey, CatClaims, CatAuthz, CatKeyMgmt, CatJWKS, CatEncoding,
}

// CategoryStat is per-category coverage.
type CategoryStat struct {
	Category Category
	Total    int
	Covered  int
}

// Report is the computed coverage of the taxonomy.
type Report struct {
	Total       int
	Covered     int
	Categories  []CategoryStat
	Gaps        []Threat // uncovered, most-severe first
	CoveredList []Threat // covered, most-severe first
}

// Pct is the integer coverage percentage (0 when the catalog is empty).
func (r Report) Pct() int {
	if r.Total == 0 {
		return 0
	}
	return int(float64(r.Covered) / float64(r.Total) * 100)
}

// Compute measures coverage across the catalog. A threat counts as covered iff it
// maps to at least one detector; the covered/gap lists are sorted most-severe
// first (then by ID) for stable reporting.
func Compute(cat []Threat) Report {
	r := Report{Total: len(cat)}
	catIndex := map[Category]*CategoryStat{}
	for _, c := range categoryOrder {
		catIndex[c] = &CategoryStat{Category: c}
	}
	for _, t := range cat {
		cs := catIndex[t.Category]
		if cs == nil {
			cs = &CategoryStat{Category: t.Category}
			catIndex[t.Category] = cs
		}
		cs.Total++
		if t.Covered() {
			r.Covered++
			cs.Covered++
			r.CoveredList = append(r.CoveredList, t)
		} else {
			r.Gaps = append(r.Gaps, t)
		}
	}
	for _, c := range categoryOrder {
		if cs := catIndex[c]; cs != nil && cs.Total > 0 {
			r.Categories = append(r.Categories, *cs)
		}
	}
	sortThreats(r.Gaps)
	sortThreats(r.CoveredList)
	return r
}

func sortThreats(ts []Threat) {
	sort.SliceStable(ts, func(i, j int) bool {
		if severityRank[ts[i].Severity] != severityRank[ts[j].Severity] {
			return severityRank[ts[i].Severity] < severityRank[ts[j].Severity]
		}
		return ts[i].ID < ts[j].ID
	})
}

func detString(ds []Detection) string {
	if len(ds) == 0 {
		return "—"
	}
	parts := make([]string, len(ds))
	for i, d := range ds {
		parts[i] = fmt.Sprintf("%s:%s", d.Kind, d.ID)
	}
	return "`" + strings.Join(parts, "`, `") + "`"
}

func srcString(ss []Source) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = fmt.Sprintf("[%s](%s)", s.Ref, s.URL)
	}
	return strings.Join(parts, "; ")
}

// Markdown renders the coverage report. generatedNote is appended as a footer
// (e.g. the command that regenerates it) so the file is clearly a build artifact.
func (r Report) Markdown(generatedNote string) string {
	var b strings.Builder
	b.WriteString("# JWT/JWKS/OIDC threat coverage\n\n")
	b.WriteString("> Auto-generated from the threat taxonomy in `internal/threats`. This is the\n")
	b.WriteString("> **denominator**: coverage is measured against the documented universe of JWT\n")
	b.WriteString("> verifier threats (RFC 8725, OWASP, CVEs, CWE, PortSwigger), not against a\n")
	b.WriteString("> corpus we wrote. Every gap below is a named, cited threat Keyway does not yet\n")
	b.WriteString("> detect — the roadmap, kept honest.\n\n")

	fmt.Fprintf(&b, "**Coverage: %d of %d documented threats (%d%%).** %d gaps remain.\n\n",
		r.Covered, r.Total, r.Pct(), len(r.Gaps))

	b.WriteString("## Coverage by category\n\n")
	b.WriteString("| Category | Covered | Total |\n|---|---|---|\n")
	for _, c := range r.Categories {
		fmt.Fprintf(&b, "| %s | %d | %d |\n", c.Category, c.Covered, c.Total)
	}
	b.WriteString("\n")

	b.WriteString("## Gaps (no detection yet)\n\n")
	if len(r.Gaps) == 0 {
		b.WriteString("_None — every cataloged threat has a detector._\n\n")
	} else {
		b.WriteString("| ID | Severity | Threat | Invariant | Sources |\n|---|---|---|---|---|\n")
		for _, t := range r.Gaps {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				t.ID, t.Severity, t.Title, t.Invariant, srcString(t.Sources))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Covered\n\n")
	b.WriteString("| ID | Severity | Threat | Detector | Sources |\n|---|---|---|---|---|\n")
	for _, t := range r.CoveredList {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			t.ID, t.Severity, t.Title, detString(t.Detections), srcString(t.Sources))
	}
	b.WriteString("\n")

	if generatedNote != "" {
		fmt.Fprintf(&b, "---\n\n_%s_\n", generatedNote)
	}
	return b.String()
}
