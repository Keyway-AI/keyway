package app

import (
	"context"
	"testing"

	"github.com/nometria/keyway/internal/blastradius"
	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/store/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticDiscoverer returns a fixed consumer set — a stand-in adapter so the app
// layer can be tested end-to-end against the in-memory store without any I/O.
type staticDiscoverer struct{ cs []model.Consumer }

func (s staticDiscoverer) Name() string { return "static" }
func (s staticDiscoverer) Discover(context.Context, discovery.Scope) ([]model.Consumer, error) {
	return s.cs, nil
}

func consumer(id string, auds ...string) model.Consumer {
	return model.Consumer{
		ID: id, StableID: id, Name: id,
		Expects:    model.Expectations{Issuers: []string{"https://kc/realms/main"}, Audiences: auds},
		Confidence: map[string]float64{"overall": 1},
	}
}

// TestSnapshotUseCase exercises the whole application use-case against the
// in-memory store: first snapshot is a baseline (no events); a real change on the
// next snapshot produces a classified event.
func TestSnapshotUseCase(t *testing.T) {
	ctx := context.Background()
	st := memory.New()

	d := Deps{Store: st, Discoverers: []discovery.Discoverer{staticDiscoverer{[]model.Consumer{consumer("svc", "a")}}}}
	res, err := d.Snapshot(ctx, "test")
	require.NoError(t, err)
	assert.True(t, res.IsBaseline)
	assert.Empty(t, res.Events, "baseline emits no events")

	// Widen the audience -> one widened event.
	d.Discoverers = []discovery.Discoverer{staticDiscoverer{[]model.Consumer{consumer("svc", "a", "b")}}}
	res, err = d.Snapshot(ctx, "test")
	require.NoError(t, err)
	require.Len(t, res.Events, 1)
	assert.Equal(t, model.ChangeWidened, res.Events[0].Class)
}

// TestUseCaseSentinelErrors verifies the sentinel errors the transports map to
// HTTP status codes when there is no snapshot yet.
func TestUseCaseSentinelErrors(t *testing.T) {
	ctx := context.Background()
	d := Deps{Store: memory.New()}

	_, err := d.ProbeRun(ctx, nil)
	assert.ErrorIs(t, err, ErrNoSnapshot)

	_, err = d.BlastRadius(ctx, blastradius.ChangeProposal{Kind: blastradius.KindRotateKey})
	assert.ErrorIs(t, err, ErrNoSnapshot)
}
