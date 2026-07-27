package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nometria/keyway/internal/model"
)

func TestMemoryStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := New()

	// Baseline + latest tracking.
	_, err := s.LatestVersion(ctx)
	require.ErrorIs(t, err, model.ErrNotFound)

	base := model.ContractVersion{ID: "v1", Hash: "h1", IsBaseline: true}
	require.NoError(t, s.SaveContractVersion(ctx, base))
	next := model.ContractVersion{ID: "v2", Hash: "h2"}
	require.NoError(t, s.SaveContractVersion(ctx, next))

	latest, err := s.LatestVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "v2", latest.ID)
	b, err := s.BaselineVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "v1", b.ID)
	got, err := s.GetContractVersion(ctx, "v1")
	require.NoError(t, err)
	assert.Equal(t, "h1", got.Hash)

	// Change events with since-filtering.
	old := model.ChangeEvent{ID: "e1", DetectedAt: time.Now().Add(-2 * time.Hour)}
	recent := model.ChangeEvent{ID: "e2", DetectedAt: time.Now()}
	require.NoError(t, s.SaveChangeEvents(ctx, []model.ChangeEvent{old, recent}))
	since, err := s.ListChangeEvents(ctx, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, since, 1)
	assert.Equal(t, "e2", since[0].ID)

	// Probe history: per-consumer, newest first, capped.
	now := time.Now()
	require.NoError(t, s.SaveProbeResults(ctx, []model.ProbeResult{
		{ID: "p1", ConsumerID: "svc", RunAt: now.Add(-time.Hour)},
		{ID: "p2", ConsumerID: "svc", RunAt: now},
		{ID: "p3", ConsumerID: "other", RunAt: now},
	}))
	hist, err := s.ProbeHistory(ctx, "svc", 10)
	require.NoError(t, err)
	require.Len(t, hist, 2)
	assert.Equal(t, "p2", hist[0].ID, "newest first")
	capped, err := s.ProbeHistory(ctx, "svc", 1)
	require.NoError(t, err)
	assert.Len(t, capped, 1)
}
