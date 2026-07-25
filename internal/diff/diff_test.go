package diff

import (
	"testing"

	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func consumer(stableID string, expects model.Expectations) model.Consumer {
	return model.Consumer{ID: stableID, StableID: stableID, Name: stableID, Expects: expects}
}

func version(id string, cs ...model.Consumer) model.ContractVersion {
	return model.ContractVersion{ID: id, Consumers: cs}
}

// AC-8: an unrelated change that does not touch the contract yields zero events.
func TestComputeNoOp(t *testing.T) {
	c := consumer("api-a", model.Expectations{Audiences: []string{"api-a"}, Algorithms: []string{"RS256"}})
	from := version("v1", c)
	to := version("v2", c) // identical contract
	assert.Empty(t, Compute(from, to))
}

// AC-7: adding an audience produces exactly one widened event.
func TestComputeAudienceAdded(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{Audiences: []string{"api-a"}}))
	to := version("v2", consumer("api-a", model.Expectations{Audiences: []string{"api-a", "api-b"}}))

	events := Compute(from, to)
	require.Len(t, events, 1, "exactly one change event")
	e := events[0]
	assert.Equal(t, FieldAudiences, e.Field)
	assert.Equal(t, model.ChangeWidened, e.Class)
	assert.Equal(t, model.SeverityMedium, e.Severity)
	assert.Equal(t, "api-b", e.NewValue)
	assert.Nil(t, e.OldValue)
	assert.Equal(t, "api-a", e.ConsumerID)
}

func TestComputeAudienceRemoved(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{Audiences: []string{"api-a", "api-b"}}))
	to := version("v2", consumer("api-a", model.Expectations{Audiences: []string{"api-a"}}))

	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, model.ChangeNarrowed, events[0].Class)
	assert.Equal(t, "api-b", events[0].OldValue)
}

func TestComputeAlgNoneCritical(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{Algorithms: []string{"RS256"}}))
	to := version("v2", consumer("api-a", model.Expectations{Algorithms: []string{"RS256", "none"}}))

	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, model.ChangeWidened, events[0].Class)
	assert.Equal(t, model.SeverityCritical, events[0].Severity)
}

func TestComputeRequiredClaimRemovedCritical(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{RequiredClaims: []string{"sub", "dept"}}))
	to := version("v2", consumer("api-a", model.Expectations{RequiredClaims: []string{"sub"}}))

	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, model.ChangeWidened, events[0].Class)
	assert.Equal(t, model.SeverityCritical, events[0].Severity)
}

func TestComputeRefreshUnknownKidHigh(t *testing.T) {
	tr, fa := true, false
	from := version("v1", model.Consumer{StableID: "api-a", JWKSBehavior: model.JWKSBehavior{RefreshesOnUnknownKID: &tr}})
	to := version("v2", model.Consumer{StableID: "api-a", JWKSBehavior: model.JWKSBehavior{RefreshesOnUnknownKID: &fa}})

	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, FieldRefreshUnknown, events[0].Field)
	assert.Equal(t, model.ChangeNarrowed, events[0].Class)
	assert.Equal(t, model.SeverityHigh, events[0].Severity)
}

// Learning a previously-unknown JWKS value (nil -> known) must not page.
func TestComputeRefreshUnknownKidLearned(t *testing.T) {
	fa := false
	from := version("v1", model.Consumer{StableID: "api-a", JWKSBehavior: model.JWKSBehavior{}})
	to := version("v2", model.Consumer{StableID: "api-a", JWKSBehavior: model.JWKSBehavior{RefreshesOnUnknownKID: &fa}})
	assert.Empty(t, Compute(from, to))
}

func TestComputeConsumerAddedRemoved(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{}))
	to := version("v2", consumer("api-b", model.Expectations{}))

	events := Compute(from, to)
	require.Len(t, events, 2)
	// One add (api-b), one remove (api-a); both neutral.
	for _, e := range events {
		assert.Equal(t, model.ChangeNeutral, e.Class)
		assert.Equal(t, FieldConsumer, e.Field)
	}
}

// The following tests exist to pin down branches that the discovery-driven
// corpus cannot reach (Istio/Envoy do not declare clock skew, and the corpus
// never shrinks a cache TTL through this exact code path), so mutation testing
// has coverage of every direction of every numeric/boolean comparison.

func TestComputeClockSkewWidened(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{ClockSkewSec: 60}))
	to := version("v2", consumer("api-a", model.Expectations{ClockSkewSec: 300}))
	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, FieldClockSkew, events[0].Field)
	assert.Equal(t, model.ChangeWidened, events[0].Class)
	assert.Equal(t, model.SeverityMedium, events[0].Severity)
}

func TestComputeClockSkewNarrowed(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{ClockSkewSec: 300}))
	to := version("v2", consumer("api-a", model.Expectations{ClockSkewSec: 30}))
	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, model.ChangeNarrowed, events[0].Class)
	assert.Equal(t, model.SeverityLow, events[0].Severity)
}

func jwksTTL(stableID string, ttl int) model.Consumer {
	return model.Consumer{StableID: stableID, Name: stableID, JWKSBehavior: model.JWKSBehavior{CacheTTLSec: &ttl}}
}

func TestComputeCacheTTLIncreased(t *testing.T) {
	from := version("v1", jwksTTL("api-a", 300))
	to := version("v2", jwksTTL("api-a", 3600))
	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, FieldCacheTTL, events[0].Field)
	assert.Equal(t, model.ChangeNarrowed, events[0].Class)
	assert.Equal(t, model.SeverityMedium, events[0].Severity) // longer cache raises grace need
}

func TestComputeCacheTTLDecreased(t *testing.T) {
	from := version("v1", jwksTTL("api-a", 3600))
	to := version("v2", jwksTTL("api-a", 300))
	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, model.ChangeNarrowed, events[0].Class)
	assert.Equal(t, model.SeverityLow, events[0].Severity)
}

// A cache TTL that is only *learned* (nil -> value) or unchanged must not page.
func TestComputeCacheTTLLearnedAndEqualAreSilent(t *testing.T) {
	learned := version("v2", jwksTTL("api-a", 300))
	fromNil := version("v1", model.Consumer{StableID: "api-a", JWKSBehavior: model.JWKSBehavior{}})
	assert.Empty(t, Compute(fromNil, learned), "learning a TTL is not a change")
	assert.Empty(t, Compute(learned, learned), "equal TTL is not a change")
}

func TestComputeRefreshFalseToTrue(t *testing.T) {
	tr, fa := true, false
	from := version("v1", model.Consumer{StableID: "api-a", JWKSBehavior: model.JWKSBehavior{RefreshesOnUnknownKID: &fa}})
	to := version("v2", model.Consumer{StableID: "api-a", JWKSBehavior: model.JWKSBehavior{RefreshesOnUnknownKID: &tr}})
	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, model.ChangeWidened, events[0].Class, "starting to refresh is a safe widening")
	assert.Equal(t, model.SeverityLow, events[0].Severity)
}

// evidenceFor prefers a provenance Locator, falling back to Source, and emits
// nothing when neither is present.
func TestEvidenceForPrefersLocatorThenSource(t *testing.T) {
	withLocator := consumer("api-a", model.Expectations{Audiences: []string{"a"}})
	withLocator.Provenance = map[string][]model.ProvenanceRecord{
		FieldAudiences: {{Source: "istio", Locator: "file.yaml"}},
	}
	to := consumer("api-a", model.Expectations{Audiences: []string{"a", "b"}})
	to.Provenance = withLocator.Provenance
	ev := Compute(version("v1", withLocator), version("v2", to))
	require.Len(t, ev, 1)
	assert.Equal(t, []string{"file.yaml"}, ev[0].Evidence, "locator preferred")

	// Source-only (no locator) falls back to the source string.
	srcOnly := consumer("api-b", model.Expectations{Audiences: []string{"a"}})
	srcOnly.Provenance = map[string][]model.ProvenanceRecord{FieldAudiences: {{Source: "k8s-env"}}}
	toB := consumer("api-b", model.Expectations{Audiences: []string{"a", "b"}})
	toB.Provenance = srcOnly.Provenance
	ev = Compute(version("v1", srcOnly), version("v2", toB))
	require.Len(t, ev, 1)
	assert.Equal(t, []string{"k8s-env"}, ev[0].Evidence, "source used when no locator")
}

func TestComputeLowConfidenceUnknown(t *testing.T) {
	from := version("v1", consumer("api-a", model.Expectations{Audiences: []string{"api-a"}}))
	toC := consumer("api-a", model.Expectations{Audiences: []string{"api-a", "api-b"}})
	toC.Confidence = map[string]float64{"overall": 0.3}
	to := version("v2", toC)

	events := Compute(from, to)
	require.Len(t, events, 1)
	assert.Equal(t, model.ChangeUnknown, events[0].Class)
	assert.Equal(t, model.SeverityInfo, events[0].Severity)
}
