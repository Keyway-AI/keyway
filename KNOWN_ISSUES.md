# Known Issues & Limitations

A living register of everything we know is incomplete, stubbed, deferred, or a
deliberate trade-off. The goal is honesty: nothing here is hidden. Update this
file in the same PR that adds or resolves an item.

**Legend** — Impact: 🔴 blocks a real use case · 🟠 degrades a feature · 🟡 minor / cosmetic · 🔵 design caveat (not a bug).
Status: `open` · `in progress` · `deferred` (intentionally not now) · `resolved`.

_Last updated: 2026-07-25 (final project audit)._

## Final audit (2026-07-25)

A whole-codebase audit (four parallel review passes + mutation/race testing) found
and **fixed** these correctness bugs — all were traced and are covered by tests:

| ID | Sev | Bug | Fix |
|----|-----|-----|-----|
| AUD-01 | 🔴 | `measuredWindow` (KI-25) took the *most-recent* fail→pass canary transition, so a flaky-probe blip could shrink the grace period below the real pickup latency → rotation outage. | Take the **maximum** observed gap (worst-case pickup). `internal/blastradius/query.go` |
| AUD-02 | 🔴 | `keystore.FileStore.Save` (KI-09) used a fixed `.tmp` path; concurrent saves on one issuer raced on rename and dropped persisted key state (113 errors / 200 trials). | `os.CreateTemp` unique file + fsync + a store mutex, and a `persistMu` serialising Export→onChange in `localkeys`. |
| AUD-03 | 🔴 | `BlastRadiusResult` wire drift: grace emitted as int64 **nanoseconds** under `recommended_grace_period` (TS expected `_seconds`); `ChangeProposal` had no JSON tags (PascalCase echo). | Added `recommended_grace_period_seconds`; snake_case JSON tags on `ChangeProposal`. |
| AUD-04 | 🟠 | `resolveRemoveClaim`/`resolveDropAlgorithm` ignored the proposal's issuer → over-warned consumers trusting a *different* issuer. | Scope to `usesIssuer` when an issuer is named. |
| AUD-05 | 🟠 | `libdefaults.Match` returned `Versions[0]` as authoritative for any out-of-range/unparseable version → mislabelled e.g. keyfunc `v0.5`/`latest` as the known-bad v1.x behaviour. | Return `ok=false` unless a version constraint (or explicit catch-all) matches. |
| AUD-06 | 🟠 | `firstIssuer()` picked `Names()[0]` off a map (nondeterministic) → probe runs could mint with the wrong issuer's key on multi-issuer setups. | Sort names for a deterministic pick. |
| AUD-07 | 🟠 | Namespace scope filter compared the **raw** namespace, but consumers default empty→`default`, so `--namespace default` silently dropped manifests omitting `metadata.namespace`. | Normalise before filtering (istio + k8s). |
| AUD-08 | 🟠 | The **L4 attribution CI gate** used Precision, which is structurally `1.0` (l4Score records only TP/FN) — the gate could never fail. | Gate on **Recall**. |
| AUD-09 | 🟡 | Consecutive-5xx abort reset on transport errors (status 0), so a service flapping 5xx/connection-refused never aborted. | Count status `≤0` as a failure too. |
| AUD-10 | 🟡 | `contract.Hash` sorted keys/claims on a single field (non-total) → theoretical non-determinism on duplicate KID/claim-name. | Total comparators. |
| AUD-11 | 🟡 | UI: `ConsumerDrawer` confidence sort used a 1-arg comparator; "Accepts more/less" pill was misleading for JWKS-behaviour findings; `types.ts` missing `last_observed_refresh`; mock probe id drift. | Total comparator; new "Rotation risk" pill; type + mock aligned. |
| AUD-12 | 🟠 | **Found by the 60-repo independent study:** real manifests sometimes write Istio `audiences` as a bare string (`audiences: "api"`) instead of a list; a `[]string` field failed to unmarshal and dropped the **entire** RequestAuthentication (issuer included) — issuer recall was 94.4%. | `audiences` now tolerates scalar-or-list (`stringSlice`); issuer recall → 100%. `internal/discovery/istio/istio.go` (test `TestScalarAudiencesTolerated`). See `docs/independent-benchmark.md`. |

Design-level gaps found and **deferred** (tracked as KI-28…KI-32 below).

_Last updated: 2026-07-24 (security audit v2)._

---

## Functional gaps (unbuilt or stubbed)

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-01 | Discovery | ~~`discovery/oidcclient` (Keycloak client-registry) stub.~~ **Resolved:** implemented against the Keycloak admin API (admin-cli grant → `GET /admin/realms/{realm}/clients` → audience mappers → consumers), tested with a mocked admin server; wired into `discover`/`snapshot`/`serve` for configured keycloak issuers. Note: OIDC-registry consumers use an `oidc://{realm}/{clientId}` StableID that won't merge with a mesh-discovered id for the same logical service (KI-24). | 🟡 | resolved | `internal/discovery/oidcclient/` |
| KI-24 | Discovery | ~~Mesh vs OIDC-registry duplicate.~~ **Resolved:** the aggregator folds an `oidc://` consumer into a mesh consumer when they share BOTH an issuer and an audience (conservative, avoids over-merge). | 🟡 | resolved | `internal/discovery/aggregate.go:aliasFold` |
| KI-02 | Discovery | ~~All discovery is file/manifest-based.~~ **Resolved:** an in-cluster `client-go` dynamic path reads live Istio `RequestAuthentication` + `AuthorizationPolicy` CRDs from the Kubernetes API and maps them through the exact same logic as the manifest source. Enabled with `--in-cluster` on `discover`/`snapshot`/`serve` (in-cluster SA config, or a `--kube-context`). Tested with the fake dynamic client. Envoy/K8s live reads can follow the same pattern if needed. | 🟠 | resolved | `internal/discovery/istio/incluster.go`, `internal/discovery/kube/client.go` |
| KI-03 | Attribution | ~~Only git attribution.~~ **Resolved:** the `Chain` now tries git → K8s deploy annotation → **Keycloak admin events** (`idp_audit`, gated to IdP-side changes, mocked-admin-server tested). Crucially, attribution is now **wired into the live pipeline**: `contract.SnapshotWithAttribution` binds each change event to its cause before persistence, and `serve`/`snapshot` build the chain from config (git+deploy root, Keycloak admin creds). | 🟡 | resolved | `internal/attribution/{keycloak,k8s,chain}.go`, `internal/contract/version.go` |
| KI-04 | API | ~~`GET /v1/probes/runs/{run_id}` had no index.~~ **Resolved:** a bounded in-memory run index (FIFO, 256) returns a run's results by id; durable per-consumer history still in Postgres. | 🟡 | resolved | `internal/api/runs.go` |
| KI-05 | API | ~~**Idempotency-Key** not enforced.~~ **Resolved:** a POST that repeats an `Idempotency-Key` replays the original response (TTL cache, `Idempotent-Replay: true`), only caching deterministic (<500) results. Cache key binds method+path+body and the store is hard-capped (SEC-05). Storage is the `coordination.IdempotencyStore` seam: in-memory (single-node) by default, or **Postgres-backed so retried writes replay across replicas** (`--db …`). | 🟢 | resolved | `internal/api/idempotency.go`, `internal/coordination/` |
| KI-06 | Scheduler | **Resolved (periodic snapshot):** `keyway serve --snapshot-interval` snapshots on an interval and notifies on change events. Automated **measured canary windows** split out as KI-25. | 🟡 | resolved | `internal/cli/serve.go:runScheduler` |
| KI-25 | Canary | ~~Grace windows came only from cache-TTL/library defaults.~~ **Resolved (measurement):** `measuredWindow` derives the real announce→pickup latency from a fail→pass transition in canary probe history and the resolver prefers it over the default, labelling the basis "measured pickup on …". Activates whenever canary probes are re-run over time (operator- or scheduler-triggered); the history-driven measurement needs no announce-time store. Auto-scheduling periodic canary re-probes is a further nicety. | 🟠 | resolved | `internal/blastradius/query.go:measuredWindow` |
| KI-07 | Contract | ~~Edge derivation was pass-through.~~ **Resolved:** `contract.Build` now synthesises a minimal Issuer per trusted issuer URL and one Edge per (issuer, consumer) trust relationship; deterministic hash preserved. | 🟡 | resolved | `internal/contract/build.go` |
| KI-08 | Degraded mode | SaaS IdPs where Keyway does **not** control the private key (Auth0/Okta/Entra) — the PRD §1.3 degraded mode (shadow issuer + library-defaults) — is not implemented. v1 is full-mode only. | 🔵 | deferred | PRD §1.3 |

## Operational / deployment caveats

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-09 | Canary | ~~Canary key material lived only in memory and was lost on restart.~~ **Resolved:** an opt-in encrypted key store (`keyway serve --key-store <dir>`, AES-256-GCM under `$KEYWAY_KEY_ENCRYPTION_KEY`) persists operated keys; the registry loads them on startup and writes back on every announce/promote/retire, so canary state survives a restart. Off by default (in-memory as before); private keys are **never** written in plaintext. Now with a **Postgres-backed `keystore.Store`** (`--key-store-db`) so key material is shared/durable across replicas, and the AES root key is sourced from a **secret manager** — env, a mounted-secret file (`--key-encryption-key-file`), or a command (`$KEYWAY_KEY_ENCRYPTION_KEY_CMD`, e.g. `vault kv get …`). | 🟢 | resolved | `internal/keystore/`, `internal/issuer/localkeys/persist.go`, `internal/issuerregistry/registry.go` |
| KI-10 | Probing | Real probing requires the target consumers to **trust Keyway's operated key** (staging setup). Against services that only trust the real IdP, minted tokens are rejected as expected — the probe suite assumes the staging-trust model. | 🔵 | by design | `internal/issuer/keycloak/keycloak.go` header |
| KI-11 | Probe concurrency | `EngineConfig.MaxConcurrentPerConsumer` is **reserved but unused**; probes run sequentially per consumer (needed for the baseline-first / consecutive-5xx logic). Only cross-consumer concurrency is active. | 🟡 | by design | `internal/probe/engine.go` |
| KI-12 | Web UI | ~~The committed binary embedded a placeholder dashboard.~~ **Resolved:** the real Vite bundle is committed under `internal/api/webdist`, so `go install` / `go build` / `keyway serve` all ship the full UI with no Node toolchain. `make web-build` rebuilds and re-embeds it (strips source maps); a test asserts the real bundle is embedded, not the placeholder. UI changes require `make web-build` + committing `webdist`. | 🟡 | resolved | `internal/api/web.go`, `Makefile:web-build` |
| KI-13 | Repo | Repository is **private**, so the CI / Go-Report / pkg.go.dev badges and the benchmark/real-world badges won't render for anonymous viewers until it's made public. | 🔵 | pending user | `README.md` |

## Benchmark / validation caveats (called out for honesty)

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-14 | Benchmark | The **generated struct** corpus (`bench/mutations`) derives its ground truth from the same field semantics the diff uses, so a perfect score there proves *consistency & regression-safety*, not that the classification rules are the "right" ones (that lives in the §9.2 table + its unit tests). The end-to-end signal is the **realistic** corpus (`L3-realistic`, ground truth from the manifest mutation) and the hand-authored file scenarios. | 🔵 | by design | `docs/benchmark.md` "Honest limitations" |
| KI-15 | Benchmark | The offline corpus scores **L1 (discovery)** and **L3 (diff)**. **L2 (live probing)** now has a docker-compose rig (`bench/l2/`, `make bench-l2`) that scores it end-to-end against 8 real containerized services — **100% correct verdicts**. **L4 (attribution)** is still covered by package tests only, not a corpus. | 🟡 | resolved (L2) | `bench/l2/`, KI-22 for L4 |
| KI-22 | Benchmark | ~~L4 had no corpus.~~ **Resolved:** the harness now scores L4 (git+deploy chain over a temp git repo + annotated manifest), feeding the §13.4 L4 gate (3/3, Youden 1.0). | 🟡 | resolved | `bench/harness/l4.go` |
| KI-16 | Real-world | The `bench/realworld` cases are **reproductions** of the cited failure modes, not live scans of the named projects. The citation identifies the real bug class; the code recreates it so detection is deterministic. | 🔵 | by design | `docs/realworld-validation.md` |
| KI-17 | Benchmark | The market-comparison numbers (Snyk/Semgrep/SonarQube/Kiuwan) are **published third-party OWASP results shown for calibration**, not a head-to-head Keyway ran. Different task. | 🔵 | by design | `BENCHMARK.md` "Honesty note" |
| KI-28 | Discovery | ~~Istio↔K8s merge axis divergence.~~ **Resolved:** `model.Consumer` gained `Aliases`, and the k8s adapter now emits its service-name identity as an alias alongside the SA-based primary. `discovery.Merge` correlates on the identity set (`{StableID} ∪ Aliases`), so a workload seen by Istio (service name) and Kubernetes (service account) merges into one record with the readable canonical id. Guarded by `TestCrossSourceMergeByAlias` (+ an over-merge guard). | 🟠 | resolved | `internal/model/consumer.go`, `internal/discovery/aggregate.go`, `internal/discovery/k8s/k8s.go` |
| KI-29 | API | ~~Idempotency had no in-flight coalescing.~~ **Resolved:** a refcounted per-key lock serializes concurrent requests that share an Idempotency-Key — the first executes and caches, the rest block then replay it — so two concurrent identical POSTs no longer both execute. Verified under `-race` (`TestIdempotencyConcurrentCoalescing`: exactly one execution across 24 simultaneous requests). The in-flight lock is same-node; cross-replica replay is now shared via the Postgres `coordination.IdempotencyStore`. | 🟢 | resolved | `internal/api/idempotency.go` |
| KI-30 | Discovery | ~~Selector-less AuthorizationPolicy claims dropped; nested claim keys mis-parsed.~~ **Resolved:** a selector-less (namespace-wide) AuthorizationPolicy now applies its required claims to every consumer in the namespace, and a nested key `request.auth.claims[a][b]` contributes its top-level segment `a`. Raised the 60-repo claim recall path. Test `TestNamespaceWideAndNestedClaims`. | 🟡 | resolved | `internal/discovery/istio/istio.go` |
| KI-31 | Discovery | ~~Envoy discovery was order-nondeterministic and dropped duplicate provider names.~~ **Resolved:** providers are now emitted in sorted order (deterministic), duplicate names keep the first definition deterministically, and per-field provenance/confidence for audiences is populated. (Namespace on a route stays the kube-context label — routes have no k8s namespace.) | 🟡 | resolved | `internal/discovery/envoy/envoy.go` |
| KI-32 | Discovery | ~~`jwksUri` was parsed then discarded.~~ **Resolved (jwksUri):** the rotation endpoint is now captured into `JWKSBehavior.JWKSURI` from both Istio `jwtRules.jwksUri` and Envoy `remote_jwks.http_uri.uri` (verified on all 5 real OSS configs). It is metadata only — not in the contract hash or diff, so no baseline shift. `Expects.Algorithms` / `RefreshIntervalSec` still come only from library-defaults enrichment (Istio/Envoy don't declare them — KI-19). | 🟡 | resolved | `internal/discovery/{istio,envoy}`, `internal/model/consumer.go` |
| KI-33 | Discovery | ~~Consumer naming fell back to the policy name for non-`app` selectors.~~ **Mostly resolved:** `serviceName`/`apServiceName` now recognise more selector conventions (`app.kubernetes.io/name`, `istio`, `k8s-app`), so e.g. a gateway RA selecting `istio: ingressgateway` is named `ingressgateway` (was the policy name). The residual case is a truly **selector-less** (namespace-wide) RequestAuthentication, which is inherently un-nameable from a workload — it still uses the policy name and is modelled as a single consumer (a namespace-wide model is future work). | 🟡 | in progress | `internal/discovery/istio/istio.go:labelValue` |
| KI-26 | Normalization | **Whitespace resolved; trailing slash kept by design.** Discovered issuers/audiences are now whitespace-trimmed (a token's `iss`/`aud` can't carry whitespace), removing that false-positive class (`adv-06`). A **trailing slash** (`…/main` vs `…/main/`) is deliberately *not* normalized: under strict OIDC the `iss` must byte-match, so it is a real contract change and is flagged (`adv-03`) — a documented judgment call, not a bug. Adversarial FPR dropped 0.5→0.25 (Youden 0.75). | 🟡 | in progress | `internal/discovery/{istio,envoy}`, `docs/benchmark-integrity.md` |
| KI-27 | Benchmark integrity | The main-corpus 100% proves self-consistency, not generalisation. Mitigated by **mutation testing** (`make mutation`: 100% mutator coverage, all behaviour-changing faults killed, 2 equivalent-mutant survivors) and a **held-out adversarial corpus** scored honestly at **0.5 Youden** (not gated). See `docs/benchmark-integrity.md`. | 🔵 | by design | `docs/benchmark-integrity.md`, `Makefile:mutation` |
| KI-18 | Corpus size | ~~Only 10 realistic scenarios.~~ **Resolved:** two moves. (1) The hand-authored file corpus went 10→**26** (15 TP / 11 TN). (2) A new **realistic generator** (`bench/harness/realistic.go`) deterministically renders **400** real Istio/Envoy/K8s YAML before/after pairs (240 TP / 160 TN) across parameterized names/issuers/audiences/claims/TTLs and runs them through the **actual discovery pipeline** — meeting the PRD's 400 aspiration with end-to-end (L1+L3) scenarios rather than struct-level mutations. `L3-realistic` scores TPR=1.0/FPR=0.0 and is CI-gated. | 🟡 | resolved | `bench/harness/realistic.go`, `bench/corpus/scenarios/` |

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
| SEC-10 | Dependencies | Reachable CVEs fixed by upgrade: go-jose JWE panic (GO-2026-4945→v4.1.4), x/text infinite loop (GO-2026-5970→v0.39.0). Later, adding client-go (KI-02) pulled reachable chi (GO-2026-5775/5777→v5.3.0) and x/net (GO-2026-4918/5026→v0.55.0) CVEs, also upgraded. | 🟠 | resolved | `go.mod`; `govulncheck` clean |

---

## How to use this file

- Every stub or deferred decision in the code should have a matching `KI-xx` row.
- When you resolve one, set Status to `resolved` (keep the row for history) and
  reference the commit/PR.
- New known issues discovered in review go here first, then get a code `TODO`
  that cites the `KI-xx` id.
