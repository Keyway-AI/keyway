# Architecture review

An independent deep-dive into how Keyway is structured — layering, extension
seams, coupling — and the highest-leverage architectural changes. Written from a
read of the whole tree (~9.5k LOC Go across 33 packages + a React UI + a
benchmark harness).

_Date: 2026-07-25._

## 1. What the system is, in one diagram

```
                 ┌──────────────── transport ────────────────┐
   CLI (cobra) ──┤                                             │
                 │   internal/api (HTTP + SPA)   internal/cli  │
                 └──────┬──────────────────────────────┬───────┘
                        │        (orchestration lives here today)
     ┌──────────────────┴───────── domain (pure, deterministic) ─────────────┐
     │  discovery → contract.Build → diff/classify → blastradius             │
     │  probe (mint + 13 attack tokens)   libdefaults   attribution          │
     └──────────────────┬─────────────────────────────────┬──────────────────┘
                         │            model (leaf)         │
     ┌───────────────────┴──── ports / adapters (infra) ───┴──────────────────┐
     │  Discoverer  issuer.Adapter  store.Store  keystore.Store  Notifier      │
     │  (istio/envoy/k8s/oidc)  (generic/keycloak/k8ssa)  (postgres) (slack)   │
     └──────────────────────────────────────────────────────────────────────┘
```

The core pipeline is a clean, deterministic data flow: **discover** consumers →
**build** a hashed contract version → **diff** against the previous version →
classify each change → compute **blast radius** / grace period → **probe** live
services with minted tokens → **attribute** and **notify**. That flow is the
product, and it is modelled honestly in code.

## 2. Layering — and it is genuinely clean at the bottom

- **`internal/model` is a leaf** (imported by 65 sites, imports nothing internal).
  The domain vocabulary — `Consumer`, `ContractVersion`, `ChangeEvent`, `Key` —
  has zero dependencies. This is the single best structural decision in the repo;
  it keeps the domain uncoupled and is why the diff/contract/blastradius packages
  are unit-testable in isolation.
- **The domain packages are pure and deterministic.** `diff`, `contract`,
  `blastradius`, `libdefaults` take data in and return data out; clocks are
  injected (`Build`/`Resolve` take `now`). This is why mutation testing and the
  offline corpus work at all.
- **No import cycles**; `cli`/`api` sit at the top and are imported only by
  `cmd/` (verified).

## 3. Extension seams — extensibility is a real strength

Eight interfaces are the designed extension points, and they are the right ones:

| Port | Add a new … | Effort |
|---|---|---|
| `discovery.Discoverer` | config source (Consul, AWS ALB, Kong…) | implement one method, register in the aggregator |
| `issuer.Adapter` / `CanaryIssuer` | IdP type (Auth0, Okta, Entra) | implement mint/JWKS/lifecycle |
| `store.Store` | persistence backend | implement the CRUD interface |
| `keystore.Store` | key-at-rest backend (Vault, KMS) | implement Load/Save |
| `notify.Notifier` | alert channel (PagerDuty, webhook) | implement Notify |
| `contract.Attributor` | change-cause source | implement Attribute |

Adding an Istio-like discovery source or a new notifier is genuinely a
"implement-one-interface" job. The `discovery` aggregator (`Run` + `aliasFold`)
and the `issuerregistry` compose these cleanly. **On the axis of "can I plug in a
new adapter", the architecture scores well.**

## 4. Where the architecture is weakest (ranked)

### W1 — No application/use-case layer; orchestration lives in the transport
The snapshot use-case (discover → enrich → build → attribute → store) exists
**twice**: `api.Server.Snapshot` (`handlers.go:45`) and `cli.runSnapshot`
(`discovery.go:177`). They drifted already — the CLI builds its own attributor,
the API injects one. There is no `internal/app`/service layer that both the CLI,
the HTTP API, and the scheduler call. Consequence: business logic is entangled
with transport, flows get duplicated, and the CLI can't reuse API behaviour (and
vice-versa). **This is the #1 thing to fix.**

### W2 — The store abstraction is leaky
`store.Store` is a clean interface, but **seven CLI files import the concrete
`store/postgres`** and call `postgres.Open` directly. So the abstraction that
should let you run against SQLite/in-memory for local dev and tests is bypassed
everywhere except the API. The interface exists; it just isn't the seam anyone
uses outside `api`.

### W3 — Inconsistent API DTO boundary → contract drift
About half the handlers return typed `pkg/apitypes` DTOs; the other half
serialize **raw domain structs** (`model.Consumer`, `model.ContractVersion`,
`blastradius.BlastRadiusResult`) or ad-hoc `map[string]any`. Serializing internal
structs directly is exactly what produced the **AUD-03 bug** (grace period sent
as int64 nanoseconds; proposal echoed in Go PascalCase). Every raw-struct
response is a latent drift with the TypeScript client.

### W4 — Consumer identity is not a first-class concept
`StableID` is computed inline (`discovery/stableid.go`) with a fixed precedence
(service-account > service-name > route > url). The k8s and istio adapters pick
*different* axes for the same workload, so they don't merge (**KI-28**), and
non-`app` selectors fall back to the policy name (**KI-33**). Identity is a domain
concept that deserves its own type (a canonical id **plus aliases**), with
correlation logic in one place rather than smeared across adapters + `aliasFold`.

### W5 — Single-daemon assumptions cap horizontal scale
Idempotency cache, the run index, and canary announce-state live in process
memory. Persistence is Postgres-only. This is documented (KI-05/09) and fine for
v1, but the *architecture* has no seam for "two replicas": no durable
idempotency, no scheduler leader-election. The seams (store, keystore) exist to
close this later; nothing forces it yet.

### W6 — Composition/wiring sprawl
`cli/serve.go` hand-assembles the whole object graph (store, registry, libs,
discoverers, in-cluster client, attributor, keystore, notifier, scheduler) inline.
There is no single composition root; the same wiring is partially repeated across
CLI commands. As adapters multiply this grows super-linearly.

### W7 — Minor: duplicated `Attributor` interface
`contract.Attributor` and `attribution.Attributor` are byte-identical, duplicated
only to dodge an import cycle. A shared port (in `model` or a small `ports`
package) removes the copy.

## 5. Top recommendations (prioritised by leverage ÷ effort)

> **Status (2026-07-25):** recommendations **#1, #2, and #3** are now implemented
> — see `internal/app`, `internal/store/{memory,open}`, and the typed DTOs in
> `pkg/apitypes`. The narrative guide is [ARCHITECTURE.md](../ARCHITECTURE.md).
> #4 (consumer identity, KI-28/KI-33) is now implemented via first-class
> `Aliases` + alias-aware merge. #6 (HA seam) remains open (KI-05/KI-09).

1. ✅ **Introduce `internal/app` (application/use-case layer).** Move the snapshot,
   probe-run, and blast-radius orchestration out of `api.Server`/`cli` into
   use-case types (`SnapshotService`, `ProbeService`, …) that depend only on
   ports (`store.Store`, `discovery.Discoverer`, `issuer` registry, `Attributor`).
   CLI, HTTP, and scheduler become thin callers. Kills W1 and most of W2/W6.
   *High leverage, medium effort.*

2. ✅ **Make `store.Store` the only persistence seam.** Add a `store.Open(dsn)`
   factory and an in-memory/SQLite implementation; have the CLI depend on the
   interface. Enables offline dev, faster tests, and the multi-replica story.
   *High leverage, low-medium effort.*

3. ✅ **One DTO boundary for the whole API (acute cases).** Every handler returns a `pkg/apitypes`
   type; nothing serializes a domain struct raw. Add a tiny contract test that
   fails if a Go response type and its TS counterpart drift (the AUD-03 class).
   *Medium leverage, low effort — and it retires a whole bug class.*

4. ✅ **First-class consumer identity (canonical id + aliases).** Replace the inline
   `StableID` precedence with an identity type that carries alternative keys, and
   put cross-source correlation in one function. Fixes KI-28/KI-33 structurally
   rather than per-adapter. *High leverage for correctness, medium effort.*

5. **A composition root.** One `internal/app.Build(cfg) (*App, error)` (or
   `wire`-style assembly) that constructs the object graph once; `serve` and each
   CLI command ask it for what they need. Shrinks `serve.go`, removes wiring
   duplication. *Medium leverage, low effort once #1 lands.*

6. **Design the HA seam now, implement later.** Define durable-idempotency and
   scheduler-leadership as interfaces (even if the only impl is "single-node")
   so multi-replica is a new adapter, not a rewrite. *Low effort now, saves a lot
   later.*

7. **Unify the `Attributor` port.** One interface, no cycle. *Trivial.*

## 6. Scorecard

| Dimension | Grade | Why |
|---|---|---|
| Domain modelling | **A** | `model` leaf; pure, deterministic core; injected clocks |
| Extensibility (adapters) | **A−** | 8 well-chosen ports; new source/issuer/notifier = one interface |
| Layering (bottom half) | **A−** | clean domain, no cycles |
| Layering (top half) | **B+** | app layer added; store behind the interface; wiring still in `serve` |
| API contract hygiene | **B** | probe/blast/consumer-probe responses now typed DTOs; a few GETs still return the domain model |
| Identity / correlation | **B** | first-class aliases; cross-source merge fixed (KI-28); richer selector naming (KI-33) — selector-less policies remain |
| Scale/HA readiness | **B−** (for v1) | single-daemon by design; seams exist but unused |
| Testing architecture | **A** | mutation + race + offline corpus + adversarial + independent OSS benchmark |

**Overall: a well-modelled, highly extensible core with a thin, slightly
entangled top.** The domain and the ports are the hard part and they are done
well; the highest-value work is introducing an application layer so the CLI, API,
and scheduler stop each re-orchestrating the domain, and tightening the API's DTO
boundary. None of this is a rewrite — it's extraction and consolidation on top of
foundations that are already sound.
