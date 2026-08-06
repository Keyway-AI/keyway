package attack

import (
	"testing"

	"github.com/Keyway-AI/keyway/internal/threats"
)

// TestTaxonomyBridge keeps the harness and the threat taxonomy in lockstep: every
// threat the harness can detect end-to-end (self-contained) must be marked
// harness-covered in the taxonomy, and every harness-covered threat in the
// taxonomy must actually be exercised by the corpus. This makes the coverage
// report's "harness" credit provably real, not aspirational.
func TestTaxonomyBridge(t *testing.T) {
	ctx, _, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := Corpus(ctx)
	if err != nil {
		t.Fatal(err)
	}

	harnessCovers := map[string]bool{}
	for _, id := range CoveredThreatIDs(corpus) {
		harnessCovers[id] = true
	}

	taxonomyMarks := map[string]bool{}
	for _, th := range threats.Catalog() {
		for _, d := range th.Detections {
			if d.Kind == threats.DetHarness {
				taxonomyMarks[th.ID] = true
			}
		}
	}

	for id := range harnessCovers {
		if !taxonomyMarks[id] {
			t.Errorf("harness detects %s but the taxonomy does not mark it harness-covered", id)
		}
	}
	for id := range taxonomyMarks {
		if !harnessCovers[id] {
			t.Errorf("taxonomy marks %s harness-covered but the corpus does not exercise it self-contained", id)
		}
	}
}
