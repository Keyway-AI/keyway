package agentauth

import (
	"testing"

	"github.com/nometria/keyway/internal/threats"
)

// TestTaxonomyBridge keeps the analyzer and the threat taxonomy in lockstep: the
// taxonomy must mark as analyzer-covered exactly the threats this analyzer checks
// — no more (no aspirational credit), no fewer (no silent detector).
func TestTaxonomyBridge(t *testing.T) {
	checked := map[string]bool{}
	for _, id := range CheckedThreatIDs() {
		checked[id] = true
	}
	marked := map[string]bool{}
	for _, th := range threats.Catalog() {
		for _, d := range th.Detections {
			if d.Kind == threats.DetAnalyzer {
				marked[th.ID] = true
			}
		}
	}
	for id := range checked {
		if !marked[id] {
			t.Errorf("analyzer checks %s but the taxonomy does not mark it analyzer-covered", id)
		}
	}
	for id := range marked {
		if !checked[id] {
			t.Errorf("taxonomy marks %s analyzer-covered but the analyzer does not check it", id)
		}
	}
}
