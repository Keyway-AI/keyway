package main

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/bench/mutations"
)

// TestGeneratedCorpusScoresPerfectly verifies the diff/classify layer scores the
// generated corpus with no false positives or negatives (the L3 gate, AC-10).
func TestGeneratedCorpusScores(t *testing.T) {
	scenarios := mutations.Generate(10)
	require.NotEmpty(t, scenarios)

	card := score(scenarios)
	assert.Greater(t, card.TP, 0)
	assert.Greater(t, card.TN, 0)
	assert.Equal(t, 0, card.FP, "no false positives (no-op changes must not alert)")
	assert.Equal(t, 0, card.FN, "no missed detections")
	assert.InDelta(t, 1.0, card.Youden, 0.001)

	// Composition is ~50/50 true/false positives (PRD §13.2).
	total := card.TP + card.TN + card.FP + card.FN
	ratio := float64(card.TP) / float64(total)
	assert.InDelta(t, 0.5, ratio, 0.05, "corpus should be roughly half true positives")
}

// realisticCount is the number of generated realistic scenarios. It defaults to
// 400 but can be lowered via KEYWAY_REALISTIC_N so mutation testing (which reruns
// the whole suite per mutant) stays fast enough to avoid timeouts — coverage of
// every risk class is retained at a much smaller count since the catalog cycles.
func realisticCount() int {
	if v := os.Getenv("KEYWAY_REALISTIC_N"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 400
}

// TestRealisticCorpusScores renders the generated realistic scenarios as real
// YAML, runs them through actual discovery + diff, and asserts a perfect score.
// This is the end-to-end regression guard for KI-18: if a future change to the
// discovery adapters or the classifier breaks any risk class, one of these
// scenarios turns into an FP or FN and this test fails.
func TestRealisticCorpusScores(t *testing.T) {
	n := realisticCount()
	scenarios, cleanup, err := loadGeneratedRealistic(n)
	require.NoError(t, err)
	defer cleanup()
	require.Len(t, scenarios, n, "every generated scenario must discover cleanly")

	card := score(scenarios)
	assert.Greater(t, card.TP, 0)
	assert.Greater(t, card.TN, 0)
	assert.Equal(t, 0, card.FP, "no-op manifest churn must not alert")
	assert.Equal(t, 0, card.FN, "every real contract change must be detected with the right class")
	assert.InDelta(t, 1.0, card.Youden, 0.001)
}

// TestRealisticCorpusDeterministic verifies generation is reproducible (no RNG),
// so CI results are stable.
func TestRealisticCorpusDeterministic(t *testing.T) {
	a := genRealistic(60)
	b := genRealistic(60)
	require.Len(t, a, 60)
	require.Equal(t, len(a), len(b))
	for i := range a {
		assert.Equal(t, a[i].name, b[i].name)
		assert.Equal(t, a[i].before, b[i].before, "scenario %d before manifests must be deterministic", i)
		assert.Equal(t, a[i].after, b[i].after, "scenario %d after manifests must be deterministic", i)
	}
}

// TestFileScenarios exercises the full discovery -> diff pipeline (L1 + L3) on
// the before/after manifest scenarios in the corpus.
func TestFileScenarios(t *testing.T) {
	scenarios, err := loadFileScenarios("../corpus")
	require.NoError(t, err)
	require.NotEmpty(t, scenarios, "expected before/after file scenarios")

	card := score(scenarios)
	// Both seed scenarios must score correctly: the widened one detected, the
	// no-op one silent.
	assert.Equal(t, 0, card.FP, "no-op scenario must not alert")
	assert.Equal(t, 0, card.FN, "widened scenario must be detected")
}
