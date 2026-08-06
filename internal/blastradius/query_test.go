package blastradius

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/probe"
)

func intp(i int) *int    { return &i }
func boolp(b bool) *bool { return &b }

func issuerVersion(consumers ...model.Consumer) model.ContractVersion {
	return model.ContractVersion{
		Issuers:   []model.Issuer{{ID: "iss-1", Name: "kc", IssuerURL: "https://kc/realms/main"}},
		Consumers: consumers,
	}
}

func consumerUsing(stableID string, jwks model.JWKSBehavior) model.Consumer {
	return model.Consumer{
		ID: stableID, StableID: stableID, Name: stableID,
		Expects:      model.Expectations{Issuers: []string{"https://kc/realms/main"}, Audiences: []string{stableID}},
		JWKSBehavior: jwks,
	}
}

func TestRotateKeyLibraryWillBreak(t *testing.T) {
	c := consumerUsing("payments-api", model.JWKSBehavior{RefreshesOnUnknownKID: boolp(false), Source: model.SrcLibraryDefault})
	c.Library = &model.LibraryInfo{Name: "MicahParks/keyfunc", Version: "v1.9.0"}
	v := issuerVersion(c)

	res, err := Resolve(v, ChangeProposal{Kind: KindRotateKey, IssuerID: "iss-1", KID: "rsa-1"}, nil, time.Now())
	require.NoError(t, err)
	require.Len(t, res.Affected, 1)
	assert.Equal(t, VerdictWillBreak, res.Affected[0].Verdict)
	assert.InDelta(t, 0.8, res.Affected[0].Confidence, 0.001)
	assert.Contains(t, res.Affected[0].Evidence[0], "keyfunc")
}

func TestRotateKeyCacheReadyWithGrace(t *testing.T) {
	c := consumerUsing("orders-api", model.JWKSBehavior{CacheTTLSec: intp(3600), RefreshesOnUnknownKID: boolp(true), Source: model.SrcConfig})
	v := issuerVersion(c)

	res, err := Resolve(v, ChangeProposal{Kind: KindRotateKey, IssuerID: "iss-1"}, nil, time.Now())
	require.NoError(t, err)
	require.Len(t, res.Affected, 1)
	assert.Equal(t, VerdictReady, res.Affected[0].Verdict)
	// Grace = 3600s * 1.5 = 5400s = 1h30m.
	assert.Equal(t, 90*time.Minute, res.RecommendedGracePeriod)
	assert.Equal(t, "orders-api", res.GraceBasis)
}

func TestRotateKeyCanaryEvidenceWins(t *testing.T) {
	c := consumerUsing("payments-api", model.JWKSBehavior{RefreshesOnUnknownKID: boolp(false), CacheTTLSec: intp(3600)})
	v := issuerVersion(c)
	now := time.Now()
	history := map[string][]model.ProbeResult{
		"payments-api": {{ProbeID: probe.ProbeCanaryKey, Passed: true, RunAt: now.Add(-time.Hour)}},
	}

	res, err := Resolve(v, ChangeProposal{Kind: KindRotateKey, IssuerID: "iss-1"}, history, now)
	require.NoError(t, err)
	require.Len(t, res.Affected, 1)
	// Canary passed -> ready with confidence 1.0, overriding the library signal.
	assert.Equal(t, VerdictReady, res.Affected[0].Verdict)
	assert.InDelta(t, 1.0, res.Affected[0].Confidence, 0.001)
	assert.Contains(t, res.Affected[0].Evidence[0], "probe:")
}

// TestRotateKeyMeasuredWindow verifies that when the canary probe history shows
// a real fail→pass transition, the grace period is based on that MEASURED pickup
// latency rather than the cache-TTL default, and the basis is labeled (KI-25).
func TestRotateKeyMeasuredWindow(t *testing.T) {
	// Cache TTL of 6h would recommend a 9h grace. The measured pickup is tighter.
	c := consumerUsing("payments-api", model.JWKSBehavior{CacheTTLSec: intp(6 * 3600), Source: model.SrcConfig})
	v := issuerVersion(c)
	now := time.Now()
	// Announced key rejected at -4h, accepted at -2h → measured pickup = 2h.
	history := map[string][]model.ProbeResult{
		"payments-api": {
			{ProbeID: probe.ProbeCanaryKey, Passed: false, RunAt: now.Add(-4 * time.Hour)},
			{ProbeID: probe.ProbeCanaryKey, Passed: true, RunAt: now.Add(-2 * time.Hour)},
		},
	}
	res, err := Resolve(v, ChangeProposal{Kind: KindRotateKey, IssuerID: "iss-1"}, history, now)
	require.NoError(t, err)
	require.Len(t, res.Affected, 1)
	assert.Equal(t, VerdictReady, res.Affected[0].Verdict)
	// Grace = measured 2h * 1.5 = 3h, NOT the 9h the 6h cache TTL would give.
	assert.Equal(t, 3*time.Hour, res.RecommendedGracePeriod)
	assert.Contains(t, res.GraceBasis, "measured pickup")
	assert.Contains(t, res.Affected[0].Reason, "measured pickup")
}

func TestMeasuredWindowNoTransition(t *testing.T) {
	now := time.Now()
	// Only passes (already picked up on first probe) → no measurable transition.
	only := []model.ProbeResult{
		{ProbeID: probe.ProbeCanaryKey, Passed: true, RunAt: now.Add(-time.Hour)},
		{ProbeID: probe.ProbeCanaryKey, Passed: true, RunAt: now.Add(-30 * time.Minute)},
	}
	_, ok := measuredWindow(only, now)
	assert.False(t, ok, "no fail→pass transition means no measurement")

	// A real slow pickup (1h) followed later by a flaky-probe blip (5m) must keep
	// the LARGEST gap — grace is sized to the worst observed pickup, so noise
	// cannot shrink it and cause a rotation outage.
	seq := []model.ProbeResult{
		{ProbeID: probe.ProbeCanaryKey, Passed: false, RunAt: now.Add(-3 * time.Hour)},
		{ProbeID: probe.ProbeCanaryKey, Passed: true, RunAt: now.Add(-2 * time.Hour)}, // 1h gap (real pickup)
		{ProbeID: probe.ProbeCanaryKey, Passed: false, RunAt: now.Add(-40 * time.Minute)},
		{ProbeID: probe.ProbeCanaryKey, Passed: true, RunAt: now.Add(-35 * time.Minute)}, // 5m blip
	}
	d, ok := measuredWindow(seq, now)
	require.True(t, ok)
	assert.Equal(t, time.Hour, d, "uses the largest (worst-case) transition, not the most recent")
}

func TestRotateKeyStaleCanaryIgnored(t *testing.T) {
	c := consumerUsing("payments-api", model.JWKSBehavior{RefreshesOnUnknownKID: boolp(false)})
	v := issuerVersion(c)
	now := time.Now()
	history := map[string][]model.ProbeResult{
		"payments-api": {{ProbeID: probe.ProbeCanaryKey, Passed: true, RunAt: now.Add(-48 * time.Hour)}},
	}
	res, err := Resolve(v, ChangeProposal{Kind: KindRotateKey, IssuerID: "iss-1"}, history, now)
	require.NoError(t, err)
	// Stale canary ignored -> falls through to library signal -> will_break.
	assert.Equal(t, VerdictWillBreak, res.Affected[0].Verdict)
}

func TestRemoveClaim(t *testing.T) {
	c := consumerUsing("api", model.JWKSBehavior{})
	c.Expects.RequiredClaims = []string{"sub", "dept"}
	other := consumerUsing("other", model.JWKSBehavior{})
	v := issuerVersion(c, other)

	res, err := Resolve(v, ChangeProposal{Kind: KindRemoveClaim, ClaimName: "dept"}, nil, time.Now())
	require.NoError(t, err)
	require.Len(t, res.Affected, 1)
	assert.Equal(t, "api", res.Affected[0].Consumer.StableID)
	assert.Equal(t, VerdictWillBreak, res.Affected[0].Verdict)
	assert.Len(t, res.Unknown, 1, "consumer not requiring the claim is unknown")
}

func TestChangeIssuer(t *testing.T) {
	c := consumerUsing("api", model.JWKSBehavior{})
	v := issuerVersion(c)
	res, err := Resolve(v, ChangeProposal{Kind: KindChangeIssuer, IssuerID: "iss-1", NewIssuerURL: "https://kc/realms/new"}, nil, time.Now())
	require.NoError(t, err)
	require.Len(t, res.Affected, 1)
	assert.Equal(t, VerdictWillBreak, res.Affected[0].Verdict)
}

func TestDropAlgorithm(t *testing.T) {
	only := consumerUsing("rs256-only", model.JWKSBehavior{})
	only.Expects.Algorithms = []string{"RS256"}
	multi := consumerUsing("multi", model.JWKSBehavior{})
	multi.Expects.Algorithms = []string{"RS256", "ES256"}
	v := issuerVersion(only, multi)

	res, err := Resolve(v, ChangeProposal{Kind: KindDropAlgorithm, Algorithm: "RS256"}, nil, time.Now())
	require.NoError(t, err)
	require.Len(t, res.Affected, 1)
	assert.Equal(t, "rs256-only", res.Affected[0].Consumer.StableID)
}

// TestAC9_Performance verifies a 50-consumer graph resolves well under 10s and
// picks a bounding consumer for the grace period.
func TestAC9_Performance(t *testing.T) {
	var consumers []model.Consumer
	for i := 0; i < 50; i++ {
		ttl := 300 + i*10
		consumers = append(consumers, consumerUsing(
			fmt.Sprintf("svc-%02d", i),
			model.JWKSBehavior{CacheTTLSec: intp(ttl), RefreshesOnUnknownKID: boolp(true), Source: model.SrcConfig},
		))
	}
	v := issuerVersion(consumers...)

	start := time.Now()
	res, err := Resolve(v, ChangeProposal{Kind: KindRotateKey, IssuerID: "iss-1"}, nil, time.Now())
	elapsed := time.Since(start)
	require.NoError(t, err)
	assert.Less(t, elapsed, 10*time.Second)
	assert.Len(t, res.Affected, 50)
	// Bounding consumer is svc-49 (largest TTL = 790s). 790s * 1.5 = 1185s, which
	// is below the 1h floor, so the recommendation floors to 1h (PRD §10.3).
	assert.Equal(t, "svc-49", res.GraceBasis)
	assert.Equal(t, time.Hour, res.RecommendedGracePeriod)
}
