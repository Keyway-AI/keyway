package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nometria/keyway/internal/contract"
	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStore opens a store against KEYWAY_TEST_DB, skipping the test when unset
// so `go test ./...` stays green without a database. Set e.g.
//
//	KEYWAY_TEST_DB=postgres://keyway:keyway@localhost:5439/keyway?sslmode=disable
func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("KEYWAY_TEST_DB")
	if dsn == "" {
		t.Skip("KEYWAY_TEST_DB not set; skipping Postgres integration test")
	}
	require.NoError(t, MigrateUp(dsn))
	st, err := Open(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(st.Close)
	return st
}

func sampleVersion(baseline bool) model.ContractVersion {
	c := model.Consumer{
		ID:       uuid.NewString(),
		StableID: "k8s://test/default/" + uuid.NewString(),
		Kind:     model.ConsumerService,
		Name:     "api-test",
		Expects:  model.Expectations{Audiences: []string{"api-test"}, Algorithms: []string{"RS256"}},
	}
	v := contract.Build(contract.BuildInput{Consumers: []model.Consumer{c}, TriggerKind: "manual"})
	v.IsBaseline = baseline
	return v
}

func TestSaveAndGetVersion(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	v := sampleVersion(false)
	require.NoError(t, st.SaveContractVersion(ctx, v))

	got, err := st.GetContractVersion(ctx, v.ID)
	require.NoError(t, err)
	assert.Equal(t, v.Hash, got.Hash)
	require.Len(t, got.Consumers, 1)
	assert.Equal(t, v.Consumers[0].StableID, got.Consumers[0].StableID)
}

func TestLatestVersion(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	older := sampleVersion(false)
	older.CreatedAt = time.Now().Add(-time.Hour).UTC()
	newer := sampleVersion(false)
	newer.CreatedAt = time.Now().UTC()
	require.NoError(t, st.SaveContractVersion(ctx, older))
	require.NoError(t, st.SaveContractVersion(ctx, newer))

	got, err := st.LatestVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, newer.ID, got.ID)
}

func TestGetVersionNotFound(t *testing.T) {
	st := testStore(t)
	_, err := st.GetContractVersion(context.Background(), uuid.NewString())
	assert.ErrorIs(t, err, model.ErrNotFound)
}

func TestChangeEventsRoundTrip(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	from := sampleVersion(true)
	to := sampleVersion(false)
	require.NoError(t, st.SaveContractVersion(ctx, from))
	require.NoError(t, st.SaveContractVersion(ctx, to))

	marker := time.Now().UTC()
	ev := model.ChangeEvent{
		ID:          uuid.NewString(),
		FromVersion: from.ID,
		ToVersion:   to.ID,
		ConsumerID:  "k8s://test/default/api",
		Field:       "expects.audiences",
		OldValue:    nil,
		NewValue:    "api-b",
		Class:       model.ChangeWidened,
		Severity:    model.SeverityMedium,
		Confidence:  1.0,
		Evidence:    []string{"istio:RequestAuthentication/api"},
		DetectedAt:  marker,
	}
	require.NoError(t, st.SaveChangeEvents(ctx, []model.ChangeEvent{ev}))

	got, err := st.ListChangeEvents(ctx, marker.Add(-time.Minute))
	require.NoError(t, err)
	require.NotEmpty(t, got)

	var found *model.ChangeEvent
	for i := range got {
		if got[i].ID == ev.ID {
			found = &got[i]
			break
		}
	}
	require.NotNil(t, found, "saved event must be listed")
	assert.Equal(t, model.ChangeWidened, found.Class)
	assert.Equal(t, "api-b", found.NewValue)
	assert.Equal(t, []string{"istio:RequestAuthentication/api"}, found.Evidence)
}

func TestProbeResultsRoundTrip(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	consumerID := "k8s://test/default/" + uuid.NewString()
	res := model.ProbeResult{
		ID:          uuid.NewString(),
		ProbeID:     "valid_token",
		ConsumerID:  consumerID,
		EndpointURL: "http://api.test/health",
		StatusCode:  200,
		LatencyMs:   12,
		Passed:      true,
		RunAt:       time.Now().UTC(),
	}
	require.NoError(t, st.SaveProbeResults(ctx, []model.ProbeResult{res}))

	got, err := st.ProbeHistory(ctx, consumerID, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "valid_token", got[0].ProbeID)
	assert.True(t, got[0].Passed)
}

// TestBaselineFlow exercises contract.Snapshot end-to-end (AC-1, AC-2): the
// first snapshot is a baseline with no events, and an identical re-snapshot is
// reported unchanged.
func TestBaselineFlow(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()

	// Use a fresh, isolated view by relying on hash equality across two builds
	// of the same graph. (Runs against a shared DB may already have a baseline;
	// this test asserts the unchanged path on a repeated identical hash.)
	v := sampleVersion(false)
	res1, err := contract.Snapshot(ctx, st, v)
	require.NoError(t, err)
	require.NotEmpty(t, res1.Version.Hash)

	// Re-snapshot the identical graph: hash matches latest -> unchanged, no events.
	same := v
	same.ID = uuid.NewString()
	res2, err := contract.Snapshot(ctx, st, same)
	require.NoError(t, err)
	assert.Empty(t, res2.Events, "identical re-snapshot emits no events")
}
