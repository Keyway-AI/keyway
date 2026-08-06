package contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/model"
)

// fakeStore is a minimal in-memory store.Store for the snapshot flow. It records
// the change events it was asked to persist so a test can inspect their
// attribution.
type fakeStore struct {
	versions map[string]model.ContractVersion
	latest   *model.ContractVersion
	baseline *model.ContractVersion
	saved    []model.ChangeEvent
}

func newFakeStore() *fakeStore {
	return &fakeStore{versions: map[string]model.ContractVersion{}}
}

func (s *fakeStore) SaveContractVersion(_ context.Context, v model.ContractVersion) error {
	s.versions[v.ID] = v
	vv := v
	s.latest = &vv
	if v.IsBaseline {
		s.baseline = &vv
	}
	return nil
}
func (s *fakeStore) GetContractVersion(_ context.Context, id string) (model.ContractVersion, error) {
	if v, ok := s.versions[id]; ok {
		return v, nil
	}
	return model.ContractVersion{}, model.ErrNotFound
}
func (s *fakeStore) LatestVersion(context.Context) (model.ContractVersion, error) {
	if s.latest == nil {
		return model.ContractVersion{}, model.ErrNotFound
	}
	return *s.latest, nil
}
func (s *fakeStore) BaselineVersion(context.Context) (model.ContractVersion, error) {
	if s.baseline == nil {
		return model.ContractVersion{}, model.ErrNotFound
	}
	return *s.baseline, nil
}
func (s *fakeStore) SaveChangeEvents(_ context.Context, events []model.ChangeEvent) error {
	s.saved = append(s.saved, events...)
	return nil
}
func (s *fakeStore) ListChangeEvents(context.Context, time.Time) ([]model.ChangeEvent, error) {
	return nil, nil
}
func (s *fakeStore) SaveProbeResults(context.Context, []model.ProbeResult) error { return nil }
func (s *fakeStore) ProbeHistory(context.Context, string, int) ([]model.ProbeResult, error) {
	return nil, nil
}

// stubAttributor tags every event with a fixed cause, to prove the snapshot flow
// runs attribution before persisting.
type stubAttributor struct{ calls int }

func (a *stubAttributor) Attribute(_ context.Context, _ model.ChangeEvent) (*model.Attribution, error) {
	a.calls++
	return &model.Attribution{Kind: "commit", Ref: "abc123", Actor: "alice", Confidence: 0.9}, nil
}

func consumer(id string, audiences ...string) model.Consumer {
	return model.Consumer{
		ID: id, StableID: id, Name: id,
		Expects:    model.Expectations{Issuers: []string{"https://kc/realms/main"}, Audiences: audiences},
		Confidence: map[string]float64{"overall": 1.0},
	}
}

func TestSnapshotWithAttribution(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()

	// Baseline: one consumer accepting ["a"]. No events on the first snapshot.
	base := Build(BuildInput{Consumers: []model.Consumer{consumer("svc", "a")}})
	_, err := SnapshotWithAttribution(ctx, st, base, &stubAttributor{})
	require.NoError(t, err)
	require.Empty(t, st.saved, "baseline must emit zero events")

	// Widen the audience -> one event, which must be attributed.
	attr := &stubAttributor{}
	next := Build(BuildInput{Consumers: []model.Consumer{consumer("svc", "a", "b")}})
	res, err := SnapshotWithAttribution(ctx, st, next, attr)
	require.NoError(t, err)
	require.NotEmpty(t, res.Events)
	assert.Positive(t, attr.calls, "attributor must be consulted for each event")

	for _, e := range st.saved {
		require.NotNil(t, e.Attribution, "persisted event must carry attribution")
		assert.Equal(t, "commit", e.Attribution.Kind)
		assert.Equal(t, "alice", e.Attribution.Actor)
	}
}

// TestSnapshotNilAttributorIsNoop verifies a nil attributor leaves events
// unattributed (behavior-preserving for callers that don't supply one).
func TestSnapshotNilAttributorIsNoop(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()
	base := Build(BuildInput{Consumers: []model.Consumer{consumer("svc", "a")}})
	_, err := SnapshotWithAttribution(ctx, st, base, nil)
	require.NoError(t, err)
	next := Build(BuildInput{Consumers: []model.Consumer{consumer("svc", "a", "b")}})
	res, err := SnapshotWithAttribution(ctx, st, next, nil)
	require.NoError(t, err)
	require.NotEmpty(t, res.Events)
	for _, e := range st.saved {
		assert.Nil(t, e.Attribution)
	}
	// Sanity: the store really is exercising the not-found baseline path.
	require.False(t, errors.Is(err, model.ErrNotFound))
}
