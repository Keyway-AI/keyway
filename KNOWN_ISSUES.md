# Known Issues & Limitations

A living register of everything we know is incomplete, stubbed, deferred, or a
deliberate trade-off. The goal is honesty: nothing here is hidden. Update this
file in the same PR that adds or resolves an item.

**Legend** — Impact: 🔴 blocks a real use case · 🟠 degrades a feature · 🟡 minor / cosmetic · 🔵 design caveat (not a bug).
Status: `open` · `in progress` · `deferred` (intentionally not now) · `resolved`.

_Last updated: 2026-07-24._

---

## Functional gaps (unbuilt or stubbed)

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-01 | Discovery | ~~`discovery/oidcclient` (Keycloak client-registry) stub.~~ **Resolved:** implemented against the Keycloak admin API (admin-cli grant → `GET /admin/realms/{realm}/clients` → audience mappers → consumers), tested with a mocked admin server; wired into `discover`/`snapshot`/`serve` for configured keycloak issuers. Note: OIDC-registry consumers use an `oidc://{realm}/{clientId}` StableID that won't merge with a mesh-discovered id for the same logical service (KI-24). | 🟡 | resolved | `internal/discovery/oidcclient/` |
| KI-24 | Discovery | The same logical consumer discovered both via the mesh (`k8s://…`) and via the OIDC client registry (`oidc://…`) gets two StableIDs and does not merge. | 🟡 | open | `internal/discovery/oidcclient/oidcclient.go` |
| KI-02 | Discovery | All discovery is **file/manifest-based**. The in-cluster `client-go` dynamic path (live CRD/Service reads) is not implemented. | 🟠 | deferred | `internal/discovery/istio/istio.go` header |
| KI-03 | Attribution | Only the **git** attributor exists. K8s-deploy and Keycloak-admin-event sources (PRD OPEN-5) are not built; those changes are `unattributed`. | 🟠 | open | `internal/attribution/attribution.go` |
| KI-04 | API | `GET /v1/probes/runs/{run_id}` returns 404 — probe results are returned inline from the POST and per-consumer from the store, but there is no run_id index. | 🟡 | open | `internal/api/handlers.go:handleGetProbeRun` |
| KI-05 | API | ~~**Idempotency-Key** not enforced.~~ **Resolved:** a POST that repeats an `Idempotency-Key` replays the original response (TTL cache, `Idempotent-Replay: true`), only caching deterministic (<500) results. In-memory today; a multi-replica deployment would back it with Postgres. | 🟡 | resolved | `internal/api/idempotency.go` |
| KI-06 | Scheduler | **Partly resolved:** `keyway serve --snapshot-interval` now periodically snapshots and delivers notifiable change events to the configured notifier. Scheduled **canary tightening** (re-running the canary to tighten measured grace windows, PRD §10.3) is still not automated. | 🟡 | in progress | `internal/cli/serve.go:runScheduler` |
| KI-07 | Contract | Edge derivation is pass-through: `contract.Build` does not yet synthesise issuer→consumer edges from discovery, so `ContractVersion.Edges` is usually empty and blast-radius keys off `Consumer.Expects.Issuers` instead. | 🟡 | open | `internal/contract/build.go` |
| KI-08 | Degraded mode | SaaS IdPs where Keyway does **not** control the private key (Auth0/Okta/Entra) — the PRD §1.3 degraded mode (shadow issuer + library-defaults) — is not implemented. v1 is full-mode only. | 🔵 | deferred | PRD §1.3 |

## Operational / deployment caveats

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-09 | Canary | Canary key material lives in the **daemon's memory** (`issuerregistry`). It does not survive a `keyway serve` restart. A production deployment that needs canary state to persist across restarts should source the operated key from a secret manager. | 🟠 | deferred | `internal/issuerregistry/registry.go` |
| KI-10 | Probing | Real probing requires the target consumers to **trust Keyway's operated key** (staging setup). Against services that only trust the real IdP, minted tokens are rejected as expected — the probe suite assumes the staging-trust model. | 🔵 | by design | `internal/issuer/keycloak/keycloak.go` header |
| KI-11 | Probe concurrency | `EngineConfig.MaxConcurrentPerConsumer` is **reserved but unused**; probes run sequentially per consumer (needed for the baseline-first / consecutive-5xx logic). Only cross-consumer concurrency is active. | 🟡 | by design | `internal/probe/engine.go` |
| KI-12 | Web UI | The committed binary embeds a **placeholder** dashboard; the real React bundle is embedded only in the Docker image (`make web-build` copies `web/dist` → `internal/api/webdist`). Local `keyway serve` shows the placeholder unless you build the web bundle first. | 🟡 | by design | `internal/api/web.go`, `Dockerfile` |
| KI-13 | Repo | Repository is **private**, so the CI / Go-Report / pkg.go.dev badges and the benchmark/real-world badges won't render for anonymous viewers until it's made public. | 🔵 | pending user | `README.md` |

## Benchmark / validation caveats (called out for honesty)

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-14 | Benchmark | The **generated** corpus derives its ground truth from the same field semantics the diff uses, so a perfect score there proves *consistency & regression-safety*, not that the classification rules are the "right" ones (that lives in the §9.2 table + its unit tests). The realistic signal is the file-based scenarios. | 🔵 | by design | `docs/benchmark.md` "Honest limitations" |
| KI-15 | Benchmark | The offline corpus scores **L1 (discovery)** and **L3 (diff)**. **L2 (live probing)** now has a docker-compose rig (`bench/l2/`, `make bench-l2`) that scores it end-to-end against 8 real containerized services — **100% correct verdicts**. **L4 (attribution)** is still covered by package tests only, not a corpus. | 🟡 | resolved (L2) | `bench/l2/`, KI-22 for L4 |
| KI-22 | Benchmark | **L4 (attribution)** has no dedicated corpus; it is exercised only by `internal/attribution` unit tests. | 🟡 | open | `internal/attribution` |
| KI-16 | Real-world | The `bench/realworld` cases are **reproductions** of the cited failure modes, not live scans of the named projects. The citation identifies the real bug class; the code recreates it so detection is deterministic. | 🔵 | by design | `docs/realworld-validation.md` |
| KI-17 | Benchmark | The market-comparison numbers (Snyk/Semgrep/SonarQube/Kiuwan) are **published third-party OWASP results shown for calibration**, not a head-to-head Keyway ran. Different task. | 🔵 | by design | `BENCHMARK.md` "Honesty note" |
| KI-18 | Corpus size | The realistic before/after corpus is **10 scenarios**; the PRD target is 400. FPR/TPR are trustworthy but would harden further with breadth. | 🟡 | open | `bench/corpus/scenarios/` |

## Toolchain

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-23 | Build | `jackc/pgx/v5` v5.10 raises the Go floor to **1.25**, so `go.mod` declares `go 1.25.0`. All Dockerfiles, CI, and docs were on 1.22 (which would have failed `go build`); now aligned to 1.25. Watch this if pinning an older Go. | 🟡 | resolved | `go.mod`, `Dockerfile`, `.github/workflows/*` |

## Detection coverage — known blind spots

| ID | Risk class | Note | Status |
|----|-----------|------|--------|
| KI-19 | Algorithm restrictions | Discovery does **not** populate `Expects.Algorithms` from Istio/Envoy (they don't declare it), so an `alg=none`/algorithm *contract* change is only caught by the **probe** (live), not by the **diff** on manifests. The probe path is covered (RW-01). | 🔵 by design |
| KI-20 | Required claims | Discovery does not derive `Expects.RequiredClaims` from any adapter, so `remove_claim` blast-radius and the missing-claim diff rely on config/probe evidence, not manifest discovery. | 🟠 open |
| KI-21 | Env-var hints | K8s env-var issuer/audience hints are confidence **0.5**, so changes to hint-derived fields classify as `unknown` (below the 0.6 floor) and never page — correct, but means low-confidence consumers get less protection until confirmed by a higher-confidence source. | 🔵 by design |

---

## How to use this file

- Every stub or deferred decision in the code should have a matching `KI-xx` row.
- When you resolve one, set Status to `resolved` (keep the row for history) and
  reference the commit/PR.
- New known issues discovered in review go here first, then get a code `TODO`
  that cites the `KI-xx` id.
