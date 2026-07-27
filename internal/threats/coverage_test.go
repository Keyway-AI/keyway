package threats

import (
	"testing"

	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/probe"
)

// TestCatalogIntegrity locks in the taxonomy's basic well-formedness so entries
// can't rot into an unusable state.
func TestCatalogIntegrity(t *testing.T) {
	cat := Catalog()
	if len(cat) < 20 {
		t.Fatalf("catalog implausibly small (%d); the taxonomy should enumerate the documented threat space", len(cat))
	}
	validSev := map[model.Severity]bool{
		model.SeverityCritical: true, model.SeverityHigh: true, model.SeverityMedium: true,
		model.SeverityLow: true, model.SeverityInfo: true,
	}
	seen := map[string]bool{}
	for _, tr := range cat {
		if tr.ID == "" {
			t.Errorf("threat with empty ID: %q", tr.Title)
		}
		if seen[tr.ID] {
			t.Errorf("duplicate threat ID %q", tr.ID)
		}
		seen[tr.ID] = true
		if tr.Title == "" || tr.Description == "" || tr.Invariant == "" {
			t.Errorf("%s: title/description/invariant must all be set", tr.ID)
		}
		if !validSev[tr.Severity] {
			t.Errorf("%s: invalid severity %q", tr.ID, tr.Severity)
		}
		if len(tr.Sources) == 0 {
			t.Errorf("%s: every threat must carry at least one external citation", tr.ID)
		}
		for _, s := range tr.Sources {
			if s.Kind == "" || s.Ref == "" || s.URL == "" {
				t.Errorf("%s: source is missing kind/ref/url: %+v", tr.ID, s)
			}
		}
	}
}

// TestProbeDetectionsExist is the honesty cross-check: every threat that claims a
// probe detector must reference a probe ID that actually exists in the registry.
// This makes "covered" mean a real detector, not a wishful annotation.
func TestProbeDetectionsExist(t *testing.T) {
	known := map[string]bool{}
	for _, p := range probe.Definitions() {
		known[p.ID] = true
	}
	for _, tr := range Catalog() {
		for _, d := range tr.Detections {
			if d.Kind == DetProbe && !known[d.ID] {
				t.Errorf("%s references probe %q which does not exist in probe.Definitions()", tr.ID, d.ID)
			}
		}
	}
}

// TestCoverageIsHonest sanity-checks the computed report: it must have real gaps
// (a security tool claiming 100% coverage of the whole JWT threat space would be
// the exact overclaim this taxonomy exists to prevent), and covered+gaps must
// partition the catalog.
func TestCoverageIsHonest(t *testing.T) {
	cat := Catalog()
	r := Compute(cat)
	if r.Total != len(cat) {
		t.Fatalf("report total %d != catalog size %d", r.Total, len(cat))
	}
	if r.Covered+len(r.Gaps) != r.Total {
		t.Fatalf("covered(%d) + gaps(%d) != total(%d)", r.Covered, len(r.Gaps), r.Total)
	}
	if len(r.Gaps) == 0 {
		t.Fatal("expected real coverage gaps; a 100% claim over the full threat space is the overclaim we are guarding against")
	}
	if r.Covered == 0 {
		t.Fatal("expected some covered threats (the existing probes)")
	}
}
