package main

import (
	"testing"

	"github.com/nometria/keyway/bench/mutations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
