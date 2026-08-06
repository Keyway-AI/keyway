package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Keyway-AI/keyway/internal/contract"
	"github.com/Keyway-AI/keyway/internal/coordination"
	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/issuerregistry"
	"github.com/Keyway-AI/keyway/internal/keystore"
	"github.com/Keyway-AI/keyway/internal/libdefaults"
	"github.com/Keyway-AI/keyway/internal/probe"
	"github.com/Keyway-AI/keyway/internal/store/open"
)

// idempotencyTTL is how long an idempotent write's result is replayable.
const idempotencyTTL = 24 * time.Hour

// KeyPersistence selects how operated (canary) signing keys are persisted.
//   - Mode "" / "none": in-memory only (default; keys reset on restart).
//   - Mode "file":       encrypted files under Dir (survives restart; single node).
//   - Mode "postgres":   encrypted rows shared across replicas (requires a
//     Postgres DBURL).
//
// The AES root key always comes from a secret manager via Source (env / file /
// command), never the config file.
type KeyPersistence struct {
	Mode   string
	Dir    string
	Source keystore.KeySource
}

// BuildConfig is the resolved input to the composition root. The caller (CLI)
// parses flags/config into this; Build assembles the object graph from it.
type BuildConfig struct {
	DBURL          string
	IssuerSpecs    []issuerregistry.Spec
	Discoverers    []discovery.Discoverer
	Scope          discovery.Scope
	ProbeAllowlist []string
	Attributor     contract.Attributor
	Libs           *libdefaults.DB
	KeyPersistence KeyPersistence
}

// App is the assembled application: the use-case Deps plus the coordination
// seams and a Close that releases every owned resource (store pool, coordination
// pool, …) in reverse order.
type App struct {
	Deps        Deps
	Coordinator *coordination.Coordinator
	IssuerCount int
	closes      []func()
}

// Leader gates work that must run on one replica (the scheduler).
func (a *App) Leader() coordination.Leader { return a.Coordinator.Leader() }

// Idempotency is the shared idempotent-write store to hand the HTTP server.
func (a *App) Idempotency() coordination.IdempotencyStore { return a.Coordinator.Idempotency() }

// Close releases all owned resources.
func (a *App) Close() {
	for i := len(a.closes) - 1; i >= 0; i-- {
		a.closes[i]()
	}
}

// Build is the single composition root: it opens the store, the coordination
// seams, and the key store, wires the issuer registry, and returns an App whose
// Deps the HTTP API, the CLI, and the scheduler all drive. On any error it
// unwinds whatever was already opened.
func Build(ctx context.Context, cfg BuildConfig) (*App, error) {
	st, stCleanup, err := open.Open(ctx, cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("app: open store: %w", err)
	}
	closes := []func(){stCleanup}
	unwind := func() {
		for i := len(closes) - 1; i >= 0; i-- {
			closes[i]()
		}
	}

	coord, err := coordination.Open(ctx, cfg.DBURL, idempotencyTTL)
	if err != nil {
		unwind()
		return nil, fmt.Errorf("app: open coordination: %w", err)
	}
	closes = append(closes, coord.Close)

	keyStore, err := buildKeyStore(cfg.KeyPersistence, coord)
	if err != nil {
		unwind()
		return nil, err
	}

	reg, err := issuerregistry.NewRegistryWithStore(cfg.IssuerSpecs, keyStore)
	if err != nil {
		unwind()
		return nil, fmt.Errorf("app: build issuer registry: %w", err)
	}

	probeCfg := probe.DefaultEngineConfig()
	probeCfg.Allowlist = cfg.ProbeAllowlist

	deps := Deps{
		Store:       st,
		Issuers:     reg,
		Libs:        cfg.Libs,
		Discoverers: cfg.Discoverers,
		Scope:       cfg.Scope,
		Probe:       probeCfg,
		Attributor:  cfg.Attributor,
	}
	return &App{Deps: deps, Coordinator: coord, IssuerCount: reg.Len(), closes: closes}, nil
}

// buildKeyStore constructs the operated-key store per the persistence mode,
// sourcing the AES root key from a secret manager. It reuses the coordinator's
// Postgres pool for the "postgres" mode so the key material is shared and durable
// across replicas.
func buildKeyStore(kp KeyPersistence, coord *coordination.Coordinator) (keystore.Store, error) {
	switch kp.Mode {
	case "", "none":
		return nil, nil // in-memory canary state, as before
	case "file":
		key, err := keystore.ResolveKey(kp.Source)
		if err != nil {
			return nil, err
		}
		return keystore.NewFileStore(kp.Dir, key)
	case "postgres":
		pool := coord.Pool()
		if pool == nil {
			return nil, fmt.Errorf("app: key-store=postgres requires a Postgres DB URL")
		}
		key, err := keystore.ResolveKey(kp.Source)
		if err != nil {
			return nil, err
		}
		return keystore.NewPostgresStore(pool, key)
	default:
		return nil, fmt.Errorf("app: unknown key-store mode %q (want file|postgres)", kp.Mode)
	}
}
