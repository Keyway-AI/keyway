// Package drift aggregates auth-contract change direction over a timeline of
// contract versions, to answer Paper A's longitudinal question: do authorization
// contracts widen (accept more) more often than they narrow (accept less), and how
// many changes are risky widenings? It reuses the validated diff classifier
// (internal/diff), which labels each atomic change widened / narrowed / neutral
// with a severity; this package just tallies that direction across a sequence.
//
// The classifier is validated (see bench/harness); feeding it a real commit-history
// timeline is the "drift in the wild" study — this library is the analysis, ready
// for that corpus once it is crawled (G2).
package drift

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Keyway-AI/keyway/internal/diff"
	"github.com/Keyway-AI/keyway/internal/model"
)

// Transition summarizes the classified changes between two consecutive versions.
type Transition struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Widened  int    `json:"widened"`
	Narrowed int    `json:"narrowed"`
	Neutral  int    `json:"neutral"`
	Net      int    `json:"net"`      // Widened - Narrowed (>0 = net widening)
	Breaking int    `json:"breaking"` // widened events at high/critical severity
}

// Report is the drift-direction rollup over a whole timeline.
type Report struct {
	Versions             int          `json:"versions"`
	TotalWidened         int          `json:"total_widened"`
	TotalNarrowed        int          `json:"total_narrowed"`
	TotalNeutral         int          `json:"total_neutral"`
	NetWidened           int          `json:"net_widened"`
	BreakingWidenings    int          `json:"breaking_widenings"`
	WideningTransitions  int          `json:"widening_transitions"`  // transitions with Net > 0
	NarrowingTransitions int          `json:"narrowing_transitions"` // Net < 0
	WidenNarrowRatio     float64      `json:"widen_narrow_ratio"`    // widened / narrowed (0 if none)
	Transitions          []Transition `json:"transitions"`
}

// Analyze runs the diff classifier across each consecutive pair in a
// chronologically-ordered timeline and tallies the widen/narrow direction.
func Analyze(timeline []model.ContractVersion) Report {
	r := Report{Versions: len(timeline)}
	for i := 1; i < len(timeline); i++ {
		t := Transition{From: timeline[i-1].ID, To: timeline[i].ID}
		for _, e := range diff.Compute(timeline[i-1], timeline[i]) {
			switch e.Class {
			case model.ChangeWidened:
				t.Widened++
				if e.Severity == model.SeverityHigh || e.Severity == model.SeverityCritical {
					t.Breaking++
				}
			case model.ChangeNarrowed:
				t.Narrowed++
			case model.ChangeNeutral:
				t.Neutral++
			}
		}
		t.Net = t.Widened - t.Narrowed
		r.TotalWidened += t.Widened
		r.TotalNarrowed += t.Narrowed
		r.TotalNeutral += t.Neutral
		r.BreakingWidenings += t.Breaking
		switch {
		case t.Net > 0:
			r.WideningTransitions++
		case t.Net < 0:
			r.NarrowingTransitions++
		}
		r.Transitions = append(r.Transitions, t)
	}
	r.NetWidened = r.TotalWidened - r.TotalNarrowed
	if r.TotalNarrowed > 0 {
		r.WidenNarrowRatio = float64(r.TotalWidened) / float64(r.TotalNarrowed)
	}
	return r
}

// LoadTimeline reads a directory of contract-version JSON files (one
// model.ContractVersion per file) and returns them ordered by filename, which the
// caller is expected to make chronological (e.g. zero-padded or timestamped names).
func LoadTimeline(dir string) ([]model.ContractVersion, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	out := make([]model.ContractVersion, 0, len(names))
	for _, n := range names {
		b, rerr := os.ReadFile(filepath.Join(dir, n)) // #nosec G304 -- caller-provided timeline dir, read-only
		if rerr != nil {
			return nil, rerr
		}
		var cv model.ContractVersion
		if uerr := json.Unmarshal(b, &cv); uerr != nil {
			return nil, uerr
		}
		out = append(out, cv)
	}
	return out, nil
}
