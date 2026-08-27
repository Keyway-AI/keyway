package drift

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Keyway-AI/keyway/internal/model"
)

// version builds a one-consumer contract version with the given expectations.
func version(id string, e model.Expectations) model.ContractVersion {
	return model.ContractVersion{
		ID: id,
		Consumers: []model.Consumer{{
			StableID: "svc-1", Name: "svc-1", Expects: e,
		}},
	}
}

func TestAnalyze_WideningAndNarrowing(t *testing.T) {
	// v0: bound to one audience, requires a claim.
	// v1: adds a second audience (widening) — accepts more.
	// v2: removes the required claim (widening, critical) — accepts more.
	// v3: requires a new claim (narrowing) — accepts less.
	timeline := []model.ContractVersion{
		version("v0", model.Expectations{Audiences: []string{"a"}, RequiredClaims: []string{"groups"}}),
		version("v1", model.Expectations{Audiences: []string{"a", "b"}, RequiredClaims: []string{"groups"}}),
		version("v2", model.Expectations{Audiences: []string{"a", "b"}}),
		version("v3", model.Expectations{Audiences: []string{"a", "b"}, RequiredClaims: []string{"role"}}),
	}
	r := Analyze(timeline)

	if r.Versions != 4 || len(r.Transitions) != 3 {
		t.Fatalf("versions=%d transitions=%d, want 4/3", r.Versions, len(r.Transitions))
	}
	if r.TotalWidened != 2 || r.TotalNarrowed != 1 {
		t.Errorf("widened=%d narrowed=%d, want 2/1", r.TotalWidened, r.TotalNarrowed)
	}
	if r.NetWidened != 1 {
		t.Errorf("net widened=%d, want 1", r.NetWidened)
	}
	// Removing a required claim is a critical widening.
	if r.BreakingWidenings < 1 {
		t.Errorf("expected at least one breaking (high/critical) widening, got %d", r.BreakingWidenings)
	}
	if r.WideningTransitions != 2 || r.NarrowingTransitions != 1 {
		t.Errorf("widening/narrowing transitions = %d/%d, want 2/1",
			r.WideningTransitions, r.NarrowingTransitions)
	}
	if math.Abs(r.WidenNarrowRatio-2.0) > 1e-9 {
		t.Errorf("widen/narrow ratio = %v, want 2.0", r.WidenNarrowRatio)
	}
}

func TestAnalyze_NoNarrowingsRatioZero(t *testing.T) {
	timeline := []model.ContractVersion{
		version("v0", model.Expectations{Audiences: []string{"a"}}),
		version("v1", model.Expectations{Audiences: []string{"a", "b"}}), // widening only
	}
	r := Analyze(timeline)
	if r.TotalNarrowed != 0 || r.WidenNarrowRatio != 0 {
		t.Errorf("expected no narrowings and ratio 0 (undefined), got narrowed=%d ratio=%v",
			r.TotalNarrowed, r.WidenNarrowRatio)
	}
	if r.NetWidened <= 0 {
		t.Errorf("expected net widening, got %d", r.NetWidened)
	}
}

func TestAnalyze_SingleVersionNoTransitions(t *testing.T) {
	r := Analyze([]model.ContractVersion{version("v0", model.Expectations{})})
	if r.Versions != 1 || len(r.Transitions) != 0 {
		t.Errorf("single version should have 0 transitions, got %d", len(r.Transitions))
	}
}

func TestLoadTimeline(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, cv model.ContractVersion) {
		b, _ := json.Marshal(cv)
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Written out of order; LoadTimeline must return them filename-sorted.
	write("002.json", version("v2", model.Expectations{Audiences: []string{"a", "b"}}))
	write("001.json", version("v1", model.Expectations{Audiences: []string{"a"}}))
	write("notes.txt", model.ContractVersion{}) // ignored

	tl, err := LoadTimeline(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tl) != 2 || tl[0].ID != "v1" || tl[1].ID != "v2" {
		t.Fatalf("timeline order wrong: %+v", []string{idOf(tl, 0), idOf(tl, 1)})
	}
	// End-to-end: the loaded timeline analyzes as one widening.
	if r := Analyze(tl); r.TotalWidened != 1 {
		t.Errorf("loaded timeline widened=%d, want 1", r.TotalWidened)
	}
}

func idOf(tl []model.ContractVersion, i int) string {
	if i < len(tl) {
		return tl[i].ID
	}
	return ""
}
