// Package conformance turns raw harness probe results into a verifier
// conformance report: for each verifier (a JWT/JWKS library, proxy, or gateway
// stood up to validate tokens) it says which adversarial tokens the verifier
// accepted that a correct implementation must reject, and across all verifiers
// how prevalent each weakness is.
//
// This productizes the one-off evaluation in bench/verifiers/: the engine
// (`keyway probe --harness`) already fires the adversarial corpus and scores
// each token; this package renders that into the per-verifier / per-threat
// report a defender (or a reviewer) actually wants, in a stable table and JSON.
//
// It derives everything from []model.ProbeResult alone. The harness emits a
// control token per verifier (ProbeID "harness:CONTROL:*"); its Passed flag says
// whether the verifier trusts the test issuer at all, so a verifier that rejects
// the control is reported as unverified and excluded from prevalence (its
// rejections are meaningless, not secure).
package conformance

import (
	"sort"
	"strings"

	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/threats"
)

const (
	harnessPrefix = "harness:"
	controlThreat = "CONTROL"
)

// Finding is one adversarial token a verifier wrongly accepted.
type Finding struct {
	ThreatID   string `json:"threat_id"`
	Name       string `json:"name"`
	Title      string `json:"title,omitempty"`
	Category   string `json:"category,omitempty"`
	Severity   string `json:"severity,omitempty"`
	StatusCode int    `json:"status_code"`
}

// Verifier is one target's conformance outcome.
type Verifier struct {
	ConsumerID string    `json:"consumer_id"`
	Verified   bool      `json:"verified"` // control accepted → the verifier trusts our issuer
	Attacks    int       `json:"attacks"`  // adversarial tokens fired
	Accepted   int       `json:"accepted"` // adversarial tokens wrongly accepted (len Findings)
	Findings   []Finding `json:"findings"`
}

// Prevalence counts, for one adversarial token, how many verified verifiers
// accepted it.
type Prevalence struct {
	ThreatID  string `json:"threat_id"`
	Name      string `json:"name"`
	Title     string `json:"title,omitempty"`
	Category  string `json:"category,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Accepting int    `json:"accepting"` // verified verifiers that accepted it
	Evaluated int    `json:"evaluated"` // verified verifiers that fired it
}

// Summary is the top-line count.
type Summary struct {
	Verifiers     int `json:"verifiers"`
	Verified      int `json:"verified"`
	Clean         int `json:"clean"`         // verified with zero findings
	WithFindings  int `json:"with_findings"` // verified with ≥1 finding
	TotalFindings int `json:"total_findings"`
}

// Report is the whole conformance result.
type Report struct {
	Verifiers  []Verifier   `json:"verifiers"`
	Prevalence []Prevalence `json:"prevalence"`
	Summary    Summary      `json:"summary"`
}

var severityRank = map[string]int{
	string(model.SeverityCritical): 0,
	string(model.SeverityHigh):     1,
	string(model.SeverityMedium):   2,
	string(model.SeverityLow):      3,
	string(model.SeverityInfo):     4,
	"":                             5,
}

type meta struct{ title, category, severity string }

type token struct{ threat, name string }

// verifierAgg accumulates one verifier's results during the single pass.
type verifierAgg struct {
	verified bool
	fired    map[token]bool
	accepted map[token]bool
	findings []Finding
}

// Build computes the conformance report from harness probe results. Non-harness
// results (ProbeID without the "harness:" prefix) are ignored.
func Build(results []model.ProbeResult) Report {
	index := threatMeta()

	byConsumer := map[string]*verifierAgg{}
	var order []string
	for _, r := range results {
		threat, name, ok := parseProbeID(r.ProbeID)
		if !ok {
			continue
		}
		a := byConsumer[r.ConsumerID]
		if a == nil {
			a = &verifierAgg{fired: map[token]bool{}, accepted: map[token]bool{}}
			byConsumer[r.ConsumerID] = a
			order = append(order, r.ConsumerID)
		}
		if threat == controlThreat {
			a.verified = r.Passed
			continue
		}
		tk := token{threat, name}
		a.fired[tk] = true
		if !r.Passed { // the verifier accepted an attack it should have rejected
			a.accepted[tk] = true
			m := index[threat]
			a.findings = append(a.findings, Finding{
				ThreatID: threat, Name: name,
				Title: m.title, Category: m.category, Severity: m.severity,
				StatusCode: r.StatusCode,
			})
		}
	}
	sort.Strings(order)

	rep := Report{}
	prev := map[token]*Prevalence{}
	for _, id := range order {
		a := byConsumer[id]
		sortFindings(a.findings)
		rep.Verifiers = append(rep.Verifiers, Verifier{
			ConsumerID: id, Verified: a.verified,
			Attacks: len(a.fired), Accepted: len(a.findings), Findings: a.findings,
		})
		rep.Summary.Verifiers++
		if !a.verified {
			continue // an unverified verifier's results are not evidence
		}
		rep.Summary.Verified++
		if len(a.findings) == 0 {
			rep.Summary.Clean++
		} else {
			rep.Summary.WithFindings++
			rep.Summary.TotalFindings += len(a.findings)
		}
		for tk := range a.fired {
			p := prev[tk]
			if p == nil {
				m := index[tk.threat]
				p = &Prevalence{ThreatID: tk.threat, Name: tk.name, Title: m.title, Category: m.category, Severity: m.severity}
				prev[tk] = p
			}
			p.Evaluated++
			if a.accepted[tk] {
				p.Accepting++
			}
		}
	}
	for _, p := range prev {
		rep.Prevalence = append(rep.Prevalence, *p)
	}
	sortPrevalence(rep.Prevalence)
	return rep
}

// parseProbeID splits "harness:<threat>:<name>" into its threat and name parts.
func parseProbeID(id string) (threat, name string, ok bool) {
	rest, found := strings.CutPrefix(id, harnessPrefix)
	if !found {
		return "", "", false
	}
	threat, name, found = strings.Cut(rest, ":")
	if !found || threat == "" {
		return "", "", false
	}
	return threat, name, true
}

// threatMeta indexes the threat catalog by ID for title/category/severity.
func threatMeta() map[string]meta {
	out := map[string]meta{}
	for _, t := range threats.Catalog() {
		out[t.ID] = meta{title: t.Title, category: string(t.Category), severity: string(t.Severity)}
	}
	return out
}

func sortFindings(fs []Finding) {
	sort.Slice(fs, func(i, j int) bool {
		if severityRank[fs[i].Severity] != severityRank[fs[j].Severity] {
			return severityRank[fs[i].Severity] < severityRank[fs[j].Severity]
		}
		if fs[i].ThreatID != fs[j].ThreatID {
			return fs[i].ThreatID < fs[j].ThreatID
		}
		return fs[i].Name < fs[j].Name
	})
}

func sortPrevalence(ps []Prevalence) {
	sort.Slice(ps, func(i, j int) bool {
		// Most-accepted first, then by severity, then stable by ID/name.
		if ps[i].Accepting != ps[j].Accepting {
			return ps[i].Accepting > ps[j].Accepting
		}
		if severityRank[ps[i].Severity] != severityRank[ps[j].Severity] {
			return severityRank[ps[i].Severity] < severityRank[ps[j].Severity]
		}
		if ps[i].ThreatID != ps[j].ThreatID {
			return ps[i].ThreatID < ps[j].ThreatID
		}
		return ps[i].Name < ps[j].Name
	})
}
