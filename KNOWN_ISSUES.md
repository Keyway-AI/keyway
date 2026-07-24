# Known Issues & Limitations

A living register of everything we know is incomplete, stubbed, deferred, or a
deliberate trade-off. The goal is honesty: nothing here is hidden. Update this
file in the same PR that adds or resolves an item.

**Legend** — Impact: 🔴 blocks a real use case · 🟠 degrades a feature · 🟡 minor / cosmetic · 🔵 design caveat (not a bug).
Status: `open` · `in progress` · `deferred` (intentionally not now) · `resolved`.

_Last updated: 2026-07-24 (security audit v2)._

---

## Functional gaps (unbuilt or stubbed)

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-01 | Discovery | ~~`discovery/oidcclient` (Keycloak client-registry) stub.~~ **Resolved:** implemented against the Keycloak admin API (admin-cli grant → `GET /admin/realms/{realm}/clients` → audience mappers → consumers), tested with a mocked admin server; wired into `discover`/`snapshot`/`serve` for configured keycloak issuers. Note: OIDC-registry consumers use an `oidc://{realm}/{clientId}` StableID that won't merge with a mesh-discovered id for the same logical service (KI-24). | 🟡 | resolved | `internal/discovery/oidcclient/` |
| KI-24 | Discovery | ~~Mesh vs OIDC-registry duplicate.~~ **Resolved:** the aggregator folds an `oidc://` consumer into a mesh consumer when they share BOTH an issuer and an audience (conservative, avoids over-merge). | 🟡 | resolved | `internal/discovery/aggregate.go:aliasFold` |
| KI-02 | Discovery | All discovery is **file/manifest-based**. The in-cluster `client-go` dynamic path (live CRD/Service reads) is not implemented. | 🟠 | deferred | `internal/discovery/istio/istio.go` header |
| KI-03 | Attribution | ~~Only git attribution.~~ **Resolved:** the `Chain` now tries git → K8s deploy annotation → **Keycloak admin events** (`idp_audit`, gated to IdP-side changes, mocked-admin-server tested). Crucially, attribution is now **wired into the live pipeline**: `contract.SnapshotWithAttribution` binds each change event to its cause before persistence, and `serve`/`snapshot` build the chain from config (git+deploy root, Keycloak admin creds). | 🟡 | resolved | `internal/attribution/{keycloak,k8s,chain}.go`, `internal/contract/version.go` |
| KI-04 | API | ~~`GET /v1/probes/runs/{run_id}` had no index.~~ **Resolved:** a bounded in-memory run index (FIFO, 256) returns a run's results by id; durable per-consumer history still in Postgres. | 🟡 | resolved | `internal/api/runs.go` |
| KI-05 | API | ~~**Idempotency-Key** not enforced.~~ **Resolved:** a POST that repeats an `Idempotency-Key` replays the original response (TTL cache, `Idempotent-Replay: true`), only caching deterministic (<500) results. Cache key binds method+path+body and the store is hard-capped (SEC-05). In-memory today; a multi-replica deployment would back it with Postgres. | 🟡 | resolved | `internal/api/idempotency.go` |
| KI-06 | Scheduler | **Resolved (periodic snapshot):** `keyway serve --snapshot-interval` snapshots on an interval and notifies on change events. Automated **measured canary windows** split out as KI-25. | 🟡 | resolved | `internal/cli/serve.go:runScheduler` |
| KI-25 | Canary | **Measured grace windows** (PRD §10.3): the scheduler does not yet re-run the canary probe to record announce→pickup times, so blast-radius grace windows come from cache-TTL/library defaults, not measurement. Needs single-probe execution + announce-time bookkeeping. | 🟠 | deferred | `internal/blastradius/query.go:readyWindow` |
| KI-07 | Contract | ~~Edge derivation was pass-through.~~ **Resolved:** `contract.Build` now synthesises a minimal Issuer per trusted issuer URL and one Edge per (issuer, consumer) trust relationship; deterministic hash preserved. | 🟡 | resolved | `internal/contract/build.go` |
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
| KI-22 | Benchmark | ~~L4 had no corpus.~~ **Resolved:** the harness now scores L4 (git+deploy chain over a temp git repo + annotated manifest), feeding the §13.4 L4 gate (3/3, Youden 1.0). | 🟡 | resolved | `bench/harness/l4.go` |
| KI-16 | Real-world | The `bench/realworld` cases are **reproductions** of the cited failure modes, not live scans of the named projects. The citation identifies the real bug class; the code recreates it so detection is deterministic. | 🔵 | by design | `docs/realworld-validation.md` |
| KI-17 | Benchmark | The market-comparison numbers (Snyk/Semgrep/SonarQube/Kiuwan) are **published third-party OWASP results shown for calibration**, not a head-to-head Keyway ran. Different task. | 🔵 | by design | `BENCHMARK.md` "Honesty note" |
| KI-18 | Corpus size | The realistic before/after corpus was expanded from 10 to **26 scenarios** (15 TP / 11 TN) spanning the Istio, Envoy, and K8s sources and all diff classes (widen/narrow/neutral, incl. issuer/audience/claim/cache-TTL and consumer add/remove) plus new FP traps (list reorder, YAML reformat, GitOps annotations, unrelated resources). Still below the PRD's 400 aspiration, but materially broader; L3-all stays TPR=1.0/FPR=0.0. | 🟡 | in progress | `bench/corpus/scenarios/` |

## Toolchain

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-23 | Build | `jackc/pgx/v5` v5.10 raises the Go floor to **1.25**, so `go.mod` declares `go 1.25.0`. All Dockerfiles, CI, and docs were on 1.22 (which would have failed `go build`); now aligned to 1.25. Watch this if pinning an older Go. | 🟡 | resolved | `go.mod`, `Dockerfile`, `.github/workflows/*` |

## Detection coverage — known blind spots

| ID | Risk class | Note | Status |
|----|-----------|------|--------|
| KI-19 | Algorithm restrictions | Discovery does **not** populate `Expects.Algorithms` from Istio/Envoy (they don't declare it), so an `alg=none`/algorithm *contract* change is only caught by the **probe** (live), not by the **diff** on manifests. The probe path is covered (RW-01). | 🔵 by design |
| KI-20 | Required claims | ~~Discovery did not derive `Expects.RequiredClaims`.~~ **Resolved:** the Istio adapter now parses `AuthorizationPolicy` `when` conditions on `request.auth.claims[...]` and merges the required claims into the matching consumer (confidence 1.0), so `remove_claim` blast-radius and the missing-claim diff work from real config. | 🟡 resolved |
| KI-21 | Env-var hints | K8s env-var issuer/audience hints are confidence **0.5**, so changes to hint-derived fields classify as `unknown` (below the 0.6 floor) and never page — correct, but means low-confidence consumers get less protection until confirmed by a higher-confidence source. | 🔵 by design |

## Security (audit v2 — all resolved)

Full write-up in [docs/security-audit.md](docs/security-audit.md) "Audit v2". Method:
`govulncheck` + `gosec` + three parallel manual reviews.

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| SEC-01 | Probe guard | Staging allowlist used a **substring** match, so `example.com` also allowed `example.com.evil.net`. | 🔴 | resolved | `internal/probe/engine.go:HostAllowed` (exact / dot-suffix) |
| SEC-02 | SSRF | Probe HTTP client **followed redirects**, letting an allowlisted target 302 a minted-token request to an internal host. | 🟠 | resolved | `internal/probe/engine.go` (`CheckRedirect` fail-closed) |
| SEC-03 | SSRF | OIDC discovery/JWKS client followed redirects (issuer-controlled → internal host). | 🟠 | resolved | `internal/issuer/oidc/oidc.go:DefaultClient` |
| SEC-04 | Path traversal | K8s-deploy attributor joined an attacker-influenceable manifest path with **no confinement**; `..`/absolute escaped root. | 🟠 | resolved | `internal/attribution/k8s.go` (test `TestK8sDeployPathTraversal`) |
| SEC-05 | Idempotency | Cache key was the **raw header only** (cross-endpoint/body replay) and the map had **no hard cap** (unbounded growth). | 🟠 | resolved | `internal/api/idempotency.go` (test `TestIdempotencyBoundToBody`) |
| SEC-06 | DoS | Authenticated request bodies had **no size cap**. | 🟡 | resolved | `internal/api/server.go:maxBytes` (1 MiB) |
| SEC-07 | DoS | `http.Server` set only `ReadHeaderTimeout` (slow-loris on body/response). | 🟡 | resolved | `internal/api/server.go` (Read/Write/Idle timeouts) |
| SEC-08 | Cleartext creds | oidcclient would send admin-cli credentials over plain **http**. | 🟡 | resolved | `internal/discovery/oidcclient/oidcclient.go:splitRealmURL` (https required) |
| SEC-09 | Secret capture | `keyway init` / `issuer add` folded ambient `KEYWAY_API_TOKEN` / webhook / DB URL into the **written file**. | 🟡 | resolved | `internal/config/config.go` (`Scaffold`, `LoadFile`) |
| SEC-10 | Dependencies | Two **reachable** CVEs: go-jose JWE panic (GO-2026-4945), x/text infinite loop (GO-2026-5970). | 🟠 | resolved | `go.mod` (go-jose v4.1.4, x/text v0.39.0); `govulncheck` clean |

---

## How to use this file

- Every stub or deferred decision in the code should have a matching `KI-xx` row.
- When you resolve one, set Status to `resolved` (keep the row for history) and
  reference the commit/PR.
- New known issues discovered in review go here first, then get a code `TODO`
  that cites the `KI-xx` id.
