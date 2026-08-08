package main

import (
	"context"
	"strings"
	"testing"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/discovery/envoy"
	"github.com/Keyway-AI/keyway/internal/discovery/istio"
	"github.com/Keyway-AI/keyway/internal/discovery/k8s"
)

// TestNegativeControlPrecision is the real precision probe. The independent-parse
// validation can't measure precision honestly (it and Keyway read the same
// syntax, so captures are ~always a subset → precision ≈ 100% by construction).
// Here we plant DISTRACTOR values that a *correct* discoverer must ignore —
// commented-out issuers, issuers in non-auth resources, claims behind a
// non-matching selector, values in annotations — and assert none leak into any
// discovered contract. A leak is a genuine false positive.
func TestNegativeControlPrecision(t *testing.T) {
	consumers, err := discovery.Run(context.Background(),
		discovery.Scope{ConfigPaths: []string{"negcontrol"}},
		istio.New(), k8s.New(), envoy.New(),
	)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}

	var captured []string
	for _, c := range consumers {
		captured = append(captured, c.Expects.Issuers...)
		captured = append(captured, c.Expects.Audiences...)
		captured = append(captured, c.Expects.RequiredClaims...)
	}

	leaks := 0
	for _, v := range captured {
		if strings.Contains(strings.ToUpper(v), "DISTRACTOR") {
			t.Errorf("false positive: discovery captured a planted distractor value %q", v)
			leaks++
		}
	}

	// Sanity: the real values MUST be captured, else the corpus/test is broken and
	// the "no leaks" result would be vacuous.
	joined := strings.Join(captured, " ")
	for _, want := range []string{"https://real-1.example", "https://real-3.example"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected real value %q to be captured (sanity check)", want)
		}
	}
	t.Logf("negative-control precision: %d planted distractors, %d leaked, %d values captured",
		4, leaks, len(captured))
}
