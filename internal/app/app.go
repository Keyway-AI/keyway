// Package app is Keyway's application (use-case) layer. It holds the orchestration
// that used to live inside the HTTP handlers and the CLI — discover→build→attribute
// →store, run probes, resolve a blast radius — expressed once, against ports only
// (store.Store, discovery.Discoverer, the issuer registry, an Attributor). The
// HTTP API, the CLI, and the scheduler are thin callers of these use-cases, so the
// same behaviour is shared instead of re-implemented per transport.
package app

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/nometria/keyway/internal/blastradius"
	"github.com/nometria/keyway/internal/contract"
	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/issuerregistry"
	"github.com/nometria/keyway/internal/libdefaults"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/probe"
	"github.com/nometria/keyway/internal/store"
)

// Sentinel errors the transports map to their own status codes / messages.
var (
	ErrNoSnapshot = errors.New("no snapshot yet — take a snapshot first")
	ErrNoIssuer   = errors.New("no issuer configured to mint tokens")
)

// Deps are the ports the use-cases operate on. Everything here is an interface or
// a value type — no transport, no concrete database.
type Deps struct {
	Store       store.Store
	Discoverers []discovery.Discoverer
	Scope       discovery.Scope
	Libs        *libdefaults.DB
	Issuers     *issuerregistry.Registry
	Attributor  contract.Attributor // optional
	Probe       probe.EngineConfig
	Now         func() time.Time // optional; defaults to time.Now
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// Snapshot runs discovery, enriches with library defaults, builds a contract
// version, attributes and stores it via the baseline flow. This is the single
// definition of "take a snapshot", shared by the HTTP API, the CLI, and the
// scheduler.
func (d Deps) Snapshot(ctx context.Context, trigger string) (contract.SnapshotResult, error) {
	var consumers []model.Consumer
	if len(d.Discoverers) > 0 {
		cs, err := discovery.Run(ctx, d.Scope, d.Discoverers...)
		if err != nil {
			return contract.SnapshotResult{}, err
		}
		consumers = cs
	}
	if d.Libs != nil {
		for i := range consumers {
			d.Libs.Enrich(&consumers[i])
		}
	}
	v := contract.Build(contract.BuildInput{Consumers: consumers, TriggerKind: trigger})
	return contract.SnapshotWithAttribution(ctx, d.Store, v, d.Attributor)
}

// ProbeRun mints tokens from the configured issuer and probes the latest
// snapshot's consumers (optionally filtered), persisting the results. Returns
// ErrNoSnapshot / ErrNoIssuer for the caller to map.
func (d Deps) ProbeRun(ctx context.Context, consumerIDs []string) ([]model.ProbeResult, error) {
	v, err := d.Store.LatestVersion(ctx)
	if err != nil {
		return nil, ErrNoSnapshot
	}
	consumers := v.Consumers
	if len(consumerIDs) > 0 {
		consumers = filterByStableID(consumers, consumerIDs)
	}
	iss, ok := d.firstIssuer()
	if !ok {
		return nil, ErrNoIssuer
	}
	issModel, err := iss.Describe(ctx)
	if err != nil {
		return nil, err
	}
	eng := probe.NewEngine(d.Probe)
	results, _, err := eng.Run(ctx, issModel, mintFuncOf(iss), consumers)
	if err != nil {
		return nil, err
	}
	if err := d.Store.SaveProbeResults(ctx, results); err != nil {
		return nil, err
	}
	return results, nil
}

// BlastRadius resolves a proposed change against the latest snapshot, enriched
// with library defaults and recent probe history.
func (d Deps) BlastRadius(ctx context.Context, p blastradius.ChangeProposal) (blastradius.BlastRadiusResult, error) {
	v, err := d.Store.LatestVersion(ctx)
	if err != nil {
		return blastradius.BlastRadiusResult{}, ErrNoSnapshot
	}
	if d.Libs != nil {
		for i := range v.Consumers {
			d.Libs.Enrich(&v.Consumers[i])
		}
	}
	history := map[string][]model.ProbeResult{}
	for _, c := range v.Consumers {
		if h, herr := d.Store.ProbeHistory(ctx, c.StableID, 50); herr == nil && len(h) > 0 {
			history[c.StableID] = h
		}
	}
	return blastradius.Resolve(v, p, history, d.now())
}

// firstIssuer returns a deterministic issuer (sorted by name) to mint with.
func (d Deps) firstIssuer() (issuerregistry.CanaryIssuer, bool) {
	names := d.Issuers.Names()
	if len(names) == 0 {
		return nil, false
	}
	sort.Strings(names)
	return d.Issuers.Get(names[0])
}

func mintFuncOf(iss issuerregistry.CanaryIssuer) probe.MintFunc {
	return func(kid string, claims map[string]any) (string, error) {
		return iss.MintToken(context.Background(), kid, claims)
	}
}

func filterByStableID(consumers []model.Consumer, ids []string) []model.Consumer {
	want := make(map[string]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	var out []model.Consumer
	for _, c := range consumers {
		if want[c.StableID] {
			out = append(out, c)
		}
	}
	return out
}
