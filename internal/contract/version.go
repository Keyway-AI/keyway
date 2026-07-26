package contract

import (
	"context"
	"errors"

	"github.com/nometria/keyway/internal/diff"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/ports"
	"github.com/nometria/keyway/internal/store"
)

// SnapshotResult reports the outcome of storing a contract version.
type SnapshotResult struct {
	Version    model.ContractVersion
	IsBaseline bool
	Unchanged  bool // hash matched the latest version; no events emitted
	Events     []model.ChangeEvent
}

// Attributor binds a change event to its cause (a commit, a deploy, an IdP admin
// action). It is an alias of the shared ports.Attributor, so the contract
// package (consumer) and the attribution package (producer) share one interface
// without importing each other.
type Attributor = ports.Attributor

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
	return SnapshotWithAttribution(ctx, st, v, nil)
}

// SnapshotWithAttribution is Snapshot plus a step that attributes each emitted
// change event to its cause before persistence. A nil attributor is a no-op, so
// this is behaviour-preserving for callers that don't supply one.
func SnapshotWithAttribution(ctx context.Context, st store.Store, v model.ContractVersion, attr Attributor) (SnapshotResult, error) {
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
	if attr != nil {
		attributeEvents(ctx, attr, events)
	}
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

// attributeEvents fills in each event's Attribution in place. A failure or an
// unattributed result for one event never blocks the others — attribution is
// best-effort enrichment, not a gate on recording the change.
func attributeEvents(ctx context.Context, attr Attributor, events []model.ChangeEvent) {
	for i := range events {
		a, err := attr.Attribute(ctx, events[i])
		if err != nil || a == nil || a.Kind == "unattributed" {
			continue
		}
		events[i].Attribution = a
	}
}
