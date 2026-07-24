package contract

import (
	"context"
	"errors"

	"github.com/nometria/keyway/internal/diff"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/store"
)

// SnapshotResult reports the outcome of storing a contract version.
type SnapshotResult struct {
	Version    model.ContractVersion
	IsBaseline bool
	Unchanged  bool // hash matched the latest version; no events emitted
	Events     []model.ChangeEvent
}

// Snapshot implements the mandatory baseline flow (PRD §8.2):
//
//	if no baseline exists:
//	    mark baseline, save, emit ZERO change events
//	else:
//	    if hash == latest.hash: save, no events
//	    else: events = diff(prev, new); save version + events
//
// Violating this produces a wall of findings on first run — the documented
// pilot-killer — so the zero-events-on-baseline guarantee is load-bearing.
func Snapshot(ctx context.Context, st store.Store, v model.ContractVersion) (SnapshotResult, error) {
	// Ensure the hash reflects the current graph.
	v.Hash = Hash(v)

	_, err := st.BaselineVersion(ctx)
	switch {
	case errors.Is(err, model.ErrNotFound):
		// First ever snapshot: establish the baseline, emit nothing.
		v.IsBaseline = true
		if err := st.SaveContractVersion(ctx, v); err != nil {
			return SnapshotResult{}, err
		}
		return SnapshotResult{Version: v, IsBaseline: true}, nil
	case err != nil:
		return SnapshotResult{}, err
	}

	prev, err := st.LatestVersion(ctx)
	if err != nil {
		return SnapshotResult{}, err
	}
	if v.Hash == prev.Hash {
		// Nothing changed: persist the observation, emit no events.
		if err := st.SaveContractVersion(ctx, v); err != nil {
			return SnapshotResult{}, err
		}
		return SnapshotResult{Version: v, Unchanged: true}, nil
	}

	events := diff.Compute(prev, v)
	if err := st.SaveContractVersion(ctx, v); err != nil {
		return SnapshotResult{}, err
	}
	if len(events) > 0 {
		if err := st.SaveChangeEvents(ctx, events); err != nil {
			return SnapshotResult{}, err
		}
	}
	return SnapshotResult{Version: v, Events: events}, nil
}
