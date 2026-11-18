package conformance

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/model"
)

func ctrl(consumer string, passed bool) model.ProbeResult {
	return model.ProbeResult{ProbeID: "harness:CONTROL:valid_token", ConsumerID: consumer, Passed: passed, StatusCode: 200}
}
func attack(consumer, threat, name string, passed bool, status int) model.ProbeResult {
	return model.ProbeResult{ProbeID: "harness:" + threat + ":" + name, ConsumerID: consumer, Passed: passed, StatusCode: status}
}

// TestBuild_FindingsAndPrevalence covers the three verifier states — a finding, a
// clean verifier, and an unverified one — and the prevalence maths that must
// exclude the unverified target.
func TestBuild_FindingsAndPrevalence(t *testing.T) {
	results := []model.ProbeResult{
		// v1: trusts issuer, wrongly accepts a no-exp token (SCOPE-02), rejects alg:none.
		ctrl("v1", true),
		attack("v1", "SCOPE-02", "missing_exp", false, 200),
		attack("v1", "SIG-01", "alg_none", true, 401),
		// v2: trusts issuer, rejects everything → clean.
		ctrl("v2", true),
		attack("v2", "SCOPE-02", "missing_exp", true, 401),
		attack("v2", "SIG-01", "alg_none", true, 401),
		// v3: rejects the control → unverified, excluded from prevalence.
		ctrl("v3", false),
		attack("v3", "SCOPE-02", "missing_exp", true, 401),
		attack("v3", "SIG-01", "alg_none", true, 401),
	}

	rep := Build(results)

	assert.Equal(t, 3, rep.Summary.Verifiers)
	assert.Equal(t, 2, rep.Summary.Verified, "v3 rejected the control, so it is unverified")
	assert.Equal(t, 1, rep.Summary.Clean)
	assert.Equal(t, 1, rep.Summary.WithFindings)
	assert.Equal(t, 1, rep.Summary.TotalFindings)

	// SCOPE-02 is accepted by 1 of the 2 verified verifiers (v3 excluded).
	var scope *Prevalence
	for i := range rep.Prevalence {
		if rep.Prevalence[i].ThreatID == "SCOPE-02" {
			scope = &rep.Prevalence[i]
		}
	}
	require.NotNil(t, scope, "SCOPE-02 must appear in prevalence")
	assert.Equal(t, 1, scope.Accepting)
	assert.Equal(t, 2, scope.Evaluated, "only the two verified verifiers count toward evaluated")
	assert.NotEmpty(t, scope.Title, "prevalence is enriched from the threat catalog")

	// The finding carries enriched metadata.
	var v1 Verifier
	for _, v := range rep.Verifiers {
		if v.ConsumerID == "v1" {
			v1 = v
		}
	}
	require.Len(t, v1.Findings, 1)
	assert.Equal(t, "SCOPE-02", v1.Findings[0].ThreatID)
	assert.Equal(t, 2, v1.Attacks, "both attack tokens were fired at v1")
}

// TestBuild_IgnoresNonHarness ensures fixed-probe results (no "harness:" prefix)
// never leak into the conformance report.
func TestBuild_IgnoresNonHarness(t *testing.T) {
	results := []model.ProbeResult{
		{ProbeID: "alg_none", ConsumerID: "v1", Passed: false},
		ctrl("v1", true),
	}
	rep := Build(results)
	require.Len(t, rep.Verifiers, 1)
	assert.Zero(t, rep.Verifiers[0].Accepted, "a non-harness probe must not count as a conformance finding")
}

// TestRenderTable smoke-tests the human renderer surfaces the headline counts and
// a finding row.
func TestRenderTable(t *testing.T) {
	rep := Build([]model.ProbeResult{
		ctrl("lib-x", true),
		attack("lib-x", "SCOPE-02", "missing_exp", false, 200),
	})
	var buf bytes.Buffer
	rep.WriteTable(&buf)
	out := buf.String()
	assert.Contains(t, out, "Verifier conformance report")
	assert.Contains(t, out, "lib-x")
	assert.Contains(t, out, "SCOPE-02")
	assert.Contains(t, out, "1 FINDING(S)")
}

// TestWriteJSON round-trips.
func TestWriteJSON(t *testing.T) {
	rep := Build([]model.ProbeResult{ctrl("v", true), attack("v", "SIG-01", "alg_none", false, 200)})
	var buf bytes.Buffer
	require.NoError(t, rep.WriteJSON(&buf))
	assert.True(t, strings.Contains(buf.String(), "\"summary\""))
	assert.True(t, strings.Contains(buf.String(), "SIG-01"))
}
