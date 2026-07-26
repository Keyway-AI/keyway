# Keyway architecture

A guide to how Keyway is built — the layers, the data flow, and where to change
things. If you want the *critique* (what's weak and what to improve), see
[docs/architecture-review.md](docs/architecture-review.md). This document is the
map.

---

## 1. What Keyway does

Keyway answers one question a team usually keeps in one senior engineer's head:
**"who can log into my services, and what just changed about that?"** It

1. **discovers** every service that validates JWTs (from Istio/Envoy/Kubernetes
   manifests or a live cluster, and from an OIDC client registry),
2. **builds** a hashed *contract version* — a normalized snapshot of what each
   service accepts (issuers, audiences, algorithms, required claims, key-cache
   behaviour),
3. **diffs** it against the previous version and **classifies** each change
   (widened / narrowed / neutral, with a severity),
4. computes the **blast radius** of a proposed change (e.g. a key rotation) and a
   safe **grace period**,
5. **probes** live services with synthetic tokens (including attack tokens like
   `alg=none`) to verify behaviour,
6. **attributes** each change to its cause (a commit, a deploy, an IdP admin
   action) and **notifies**.

The whole thing ships as a single Go binary that also serves a React dashboard.

---

## 2. The layered architecture

Keyway follows a **ports-and-adapters** (hexagonal) shape. Dependencies point
**inward**: transport depends on the application layer, which depends on the
domain, which depends only on the model. Infrastructure is reached through
**ports** (interfaces) so it can be swapped.

```
┌─ Transport ───────────────────────────────────────────────┐
│  internal/api (HTTP + embedded SPA)   internal/cli (cobra) │   ← thin; no business logic
└───────────────────────────┬───────────────────────────────┘
                            │ calls use-cases
┌─ Application ─────────────┴────────────────────────────────┐
│  internal/app  —  Snapshot · ProbeRun · BlastRadius        │   ← orchestration, ONE definition
└───────────────────────────┬───────────────────────────────┘
                            │ uses domain + ports
┌─ Domain (pure, deterministic) ─────────────────────────────┐
│  discovery · contract · diff · blastradius · probe          │
│  libdefaults · attribution                                  │   ← data in, data out; clocks injected
└───────────────────────────┬───────────────────────────────┘
                            │ speaks
┌─ Model (leaf) ────────────┴────────────────────────────────┐
│  internal/model — Consumer, ContractVersion, ChangeEvent…  │   ← imports nothing internal
└────────────────────────────────────────────────────────────┘
                            ▲ implemented by
┌─ Ports & adapters (infrastructure) ────────────────────────┐
│  Discoverer(istio/envoy/k8s/oidcclient)  issuer.Adapter     │
│  store.Store(postgres/memory)  keystore.Store  Notifier     │
└────────────────────────────────────────────────────────────┘
```

**The dependency rule:** a package never imports a package from a layer above it.
`model` is a leaf (verified: 65 packages import it; it imports nothing internal).
This is what makes the domain unit-testable and mutation-testable in isolation.

---

## 3. Package map

| Package | Layer | Responsibility |
|---|---|---|
| `internal/model` | model | Domain vocabulary: `Consumer`, `Expectations`, `ContractVersion`, `ChangeEvent`, `Key`, `ProbeResult`, `Attribution`. Pure data. |
| `internal/discovery` | domain | The `Discoverer` port + aggregation (`Run`, `aliasFold`) + `StableID`. |
| `internal/discovery/{istio,envoy,k8s,oidcclient,kube}` | adapter | One discovery source each. |
| `internal/contract` | domain | `Build` (consumers → a hashed version) + `Snapshot` (baseline flow, attribution, persistence). |
| `internal/diff` | domain | `Compute` + `Classify` — the change-classification table (PRD §9.2). Deterministic. |
| `internal/blastradius` | domain | `Resolve` a proposal → affected consumers + a grace period. |
| `internal/probe` | domain | The probe engine + 13 probes; mints real/attack tokens via a `MintFunc`. |
| `internal/libdefaults` | domain | JWKS-behaviour defaults per JWT library+version (enrichment). |
| `internal/attribution` | domain | The `Attributor` chain: git → deploy → Keycloak admin events. |
| `internal/issuer/*` | adapter | Issuer adapters (generic/keycloak/k8ssa) + `localkeys` (key lifecycle, JOSE). |
| `internal/issuerregistry` | adapter | Holds live issuers; wires optional key persistence. |
| `internal/keystore` | adapter | Encrypted key-at-rest (AES-GCM), so canary keys survive a restart. |
| `internal/store` | port | The `store.Store` persistence interface. |
| `internal/store/{postgres,memory,open}` | adapter | Postgres + in-memory implementations + the `Open` factory. |
| `internal/notify` | adapter | The `Notifier` port + Slack. |
| **`internal/app`** | **application** | **The use-cases: `Snapshot`, `ProbeRun`, `BlastRadius`. Depends only on ports.** |
| `internal/api` | transport | HTTP server (chi), auth, idempotency, DTOs, serves the SPA. |
| `internal/cli` | transport | cobra commands; the composition edge that builds adapters from flags/config. |
| `pkg/apitypes` | contract | The wire DTOs shared with the TypeScript client. |
| `web/` | frontend | React dashboard (see §8). |
| `bench/` | tests | Accuracy corpus, mutation harness, adversarial + independent OSS benchmarks. |

---

## 4. The core data flow

A snapshot is the spine of the system:

```
Discoverers ─► discovery.Run ─► []Consumer
                                   │  libdefaults.Enrich (fill JWKS defaults)
                                   ▼
                            contract.Build ─► ContractVersion (deterministic Hash)
                                   │
                            contract.Snapshot(store):
                                   ├─ first ever?  → mark baseline, save, ZERO events
                                   ├─ hash == last? → save, no events
                                   └─ else → diff.Compute(prev, new) ─► []ChangeEvent
                                              │ Attributor.Attribute (who caused it)
                                              ▼  store.SaveChangeEvents
```

The baseline rule (zero events on first snapshot) is load-bearing: it's why a
first run doesn't page a team with a wall of "findings". `diff.Compute` walks the
two versions and runs each field change through `Classify`, which applies the
fixed §9.2 table (e.g. *audience added → widened/medium*, *required claim removed
→ widened/critical*, *stops refreshing on unknown kid → narrowed/high*). A
change below the confidence floor is downgraded to `unknown` and never pages.

Then, on demand: **blast radius** (`blastradius.Resolve` combines the contract,
recent probe history, and library defaults to say who will break and for how
long) and **probing** (`probe.Engine.Run` mints tokens from an issuer and fires
13 probes at each consumer, gated by a staging allowlist).

---

## 5. The application layer (`internal/app`)

`app.Deps` is the seam between transport and domain. It holds only ports:

```go
type Deps struct {
    Store       store.Store
    Discoverers []discovery.Discoverer
    Scope       discovery.Scope
    Libs        *libdefaults.DB
    Issuers     *issuerregistry.Registry
    Attributor  contract.Attributor // optional
    Probe       probe.EngineConfig
}
func (d Deps) Snapshot(ctx, trigger) (contract.SnapshotResult, error)
func (d Deps) ProbeRun(ctx, consumerIDs) ([]model.ProbeResult, error)
func (d Deps) BlastRadius(ctx, proposal) (blastradius.BlastRadiusResult, error)
```

There is **one** definition of each use-case. The HTTP handler, the CLI
`snapshot` command, and the periodic scheduler all call `app.Deps.Snapshot` — so
they cannot drift. Transports only translate: parse the request, call the
use-case, map `app.ErrNoSnapshot`/`ErrNoIssuer` to status codes, serialize a DTO.

**Composition root.** `app.Build(ctx, BuildConfig) (*App, error)` is the single
place that assembles the object graph — store, coordination seams, key store,
issuer registry — and returns the `Deps` plus an `App.Close()`. `serve.go` is now
flag-parsing followed by one `Build` call.

**HA coordination (`internal/coordination`).** Two seams make Keyway multi-replica
without a rewrite: `IdempotencyStore` (durable idempotent-write replay) and
`Leader` (so exactly one replica runs the scheduler). Each has an in-memory
single-node adapter (the default) and a Postgres adapter — idempotency rows shared
across replicas, and a session-level advisory lock for leadership. The scheduler
is leader-gated; the HTTP server takes the shared `IdempotencyStore`.

---

## 6. Extension seams — how to add things

Every outbound dependency is an interface. Adding a capability is implementing
one and registering it at the composition edge (`internal/cli`).

- **A discovery source** (e.g. Consul, AWS ALB): implement
  `discovery.Discoverer` (`Name`, `Discover(ctx, Scope) []Consumer`), add it in
  `cli.allDiscoverers`. See `internal/discovery/envoy` as the smallest example.
- **An issuer type** (e.g. Auth0): implement `issuer.Adapter` (+ `CanaryIssuer`
  for the announce/promote/retire lifecycle), add a case in
  `issuerregistry.build`.
- **A persistence backend**: implement `store.Store`, add a branch in
  `store/open.Open`. (`internal/store/memory` is a complete, readable reference.)
- **A key-at-rest backend** (Vault/KMS): implement `keystore.Store`.
- **An alert channel** (PagerDuty): implement `notify.Notifier`.
- **A change-cause source**: implement `attribution.Attributor`, add it to the
  chain in `cli.buildAttributor`.

---

## 7. Request lifecycle example — `POST /v1/snapshots`

```
HTTP POST /v1/snapshots
  └─ api: authMiddleware (constant-time bearer check)
     └─ api: maxBytes + idempotency middleware
        └─ handleCreateSnapshot
           └─ app.Deps.Snapshot(ctx, "manual")          ← application layer
              ├─ discovery.Run(scope, discoverers…)      ← domain + adapters
              ├─ libdefaults.Enrich(consumers)
              ├─ contract.Build(consumers)               ← deterministic hash
              └─ contract.SnapshotWithAttribution(store, attributor)
                 ├─ diff.Compute(prev, new)              ← classify changes
                 ├─ Attributor.Attribute(events)         ← who caused it
                 └─ store.SaveContractVersion / SaveChangeEvents
           └─ writeJSON(apitypes.SnapshotResponse{…})    ← typed DTO
```

The CLI path (`keyway snapshot`) and the scheduler run the **same**
`app.Deps.Snapshot`; only the edges differ.

---

## 8. Persistence

`store.Store` is the only persistence contract. `store/open.Open(ctx, dsn)`
returns the right implementation:

- `--db memory` (or `memory://`) → `internal/store/memory` — zero setup, used by
  tests and offline development.
- anything else → `internal/store/postgres` (migrations run first).

Canary/signing keys live in the daemon's memory by default; with
`serve --key-store <dir>` they are persisted **encrypted** (AES-256-GCM under
`$KEYWAY_KEY_ENCRYPTION_KEY`) so they survive a restart. Minted tokens are never
persisted — only a `jti` and the probe id (PRD OPEN-4).

---

## 9. Frontend (`web/`)

A React + TypeScript SPA, built by Vite and embedded into the Go binary
(`internal/api/webdist`, refreshed by `make web-build`). It talks to the same
`/v1` API; `pkg/apitypes` is the shared contract and `web/src/api/types.ts`
mirrors it. The signature feature is **`web/src/lib/findings.ts`**, which
translates raw change events into plain-English findings ("Stopped requiring a
permission claim → tokens without it are now accepted"), grouped per service and
prioritised by severity — the layer that makes the data usable by non-specialists.

When there is no backend, the client falls back to sample data (`api/mock.ts`)
so the UI is fully navigable; set `localStorage keyway.live=1` to force live.

---

## 10. Testing architecture

Correctness is defended at several levels, all runnable from `make`:

- **Unit tests** across the domain (deterministic, clock-injected).
- **Accuracy corpus** (`make bench`): a struct-level generated corpus + rendered-
  YAML realistic scenarios run through real discovery, with a CI gate.
- **Mutation testing** (`make mutation`): injects faults into the classifier and
  proves the corpus catches them (anti-overfitting evidence).
- **Adversarial corpus**: deliberately hard cases, scored honestly (not gated).
- **Independent OSS benchmark** (`make bench-oss`, `make bench-oss-study`): real
  configs from 60+ external repos — see [docs/independent-benchmark.md](docs/independent-benchmark.md).
- **Real-world CVE reproductions** (`make validate`).
- `go test -race` for the concurrency-sensitive paths (keystore, idempotency).

---

## 11. "I want to change X" — where to look

| Goal | Start here |
|---|---|
| Change how a change is classified / its severity | `internal/diff/classify.go` (+ its tests) |
| Add / fix a discovery source | `internal/discovery/<source>/` + `cli.allDiscoverers` |
| Change what a snapshot does | `internal/app/app.go` (`Snapshot`) |
| Add an API endpoint | `internal/api/{server.go (route),handlers.go}` + a `pkg/apitypes` DTO |
| Change the grace-period math | `internal/blastradius/graceperiod.go` |
| Add a probe | `internal/probe/probes.go` |
| Change the plain-English findings text | `web/src/lib/findings.ts` |
| Add a persistence backend | implement `store.Store`, branch `store/open.Open` |
| Understand a known limitation | [KNOWN_ISSUES.md](KNOWN_ISSUES.md) |
