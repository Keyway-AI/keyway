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
| KI-05 | API | ~~**Idempotency-Key** not enforced.~~ **Resolved:** a POST that repeats an `Idempotency-Key` replays the original response (TTL cache, `Idempotent-Replay: true`), only caching deterministic (<500) results. Cache key binds method+path+body and the store is hard-capped (SEC-05). In-memory today; a multi-replica deployment would back it with Postgres. | 🟡 | resolved | `internal/api/idempotency.go` |
| KI-06 | Scheduler | **Resolved (periodic snapshot):** `keyway serve --snapshot-interval` snapshots on an interval and notifies on change events. Automated **measured canary windows** split out as KI-25. | 🟡 | resolved | `internal/cli/serve.go:runScheduler` |
| KI-25 | Canary | ~~Grace windows came only from cache-TTL/library defaults.~~ **Resolved (measurement):** `measuredWindow` derives the real announce→pickup latency from a fail→pass transition in canary probe history and the resolver prefers it over the default, labelling the basis "measured pickup on …". Activates whenever canary probes are re-run over time (operator- or scheduler-triggered); the history-driven measurement needs no announce-time store. Auto-scheduling periodic canary re-probes is a further nicety. | 🟠 | resolved | `internal/blastradius/query.go:measuredWindow` |
| KI-07 | Contract | ~~Edge derivation was pass-through.~~ **Resolved:** `contract.Build` now synthesises a minimal Issuer per trusted issuer URL and one Edge per (issuer, consumer) trust relationship; deterministic hash preserved. | 🟡 | resolved | `internal/contract/build.go` |
| KI-08 | Degraded mode | SaaS IdPs where Keyway does **not** control the private key (Auth0/Okta/Entra) — the PRD §1.3 degraded mode (shadow issuer + library-defaults) — is not implemented. v1 is full-mode only. | 🔵 | deferred | PRD §1.3 |

## Operational / deployment caveats

| ID | Area | Issue | Impact | Status | Pointer |
|----|------|-------|--------|--------|---------|
| KI-09 | Canary | ~~Canary key material lived only in memory and was lost on restart.~~ **Resolved:** an opt-in encrypted key store (`keyway serve --key-store <dir>`, AES-256-GCM under `$KEYWAY_KEY_ENCRYPTION_KEY`) persists operated keys; the registry loads them on startup and writes back on every announce/promote/retire, so canary state survives a restart. Off by default (in-memory as before); private keys are **never** written in plaintext. A cloud secret-manager backend can implement the same `keystore.Store` interface. | 🟠 | resolved | `internal/keystore/`, `internal/issuer/localkeys/persist.go`, `internal/issuerregistry/registry.go` |
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
| KI-28 | Discovery | **Istio↔K8s merge axis divergence:** the k8s adapter keys a workload by service-account name when a projected SA-token audience exists (`k8s://…/{sa}`), but Istio always keys by service name (`app` label → `k8s://…/{app}`). When SA name ≠ app name, the same workload yields two un-merged consumers. Needs a reconciliation step (emit both ID candidates, or standardise on service name + carry SA as metadata). | 🟠 | open | `internal/discovery/{k8s,istio}`, `stableid.go` |
| KI-29 | API | **Idempotency has no in-flight coalescing:** the cache is populated only after the handler finishes, so two *concurrent* identical POSTs both execute (e.g. two snapshots). Sequential retries are deduped; concurrent ones need a per-key singleflight. | 🟡 | open | `internal/api/idempotency.go` |
| KI-30 | Discovery | **Selector-less / namespace-wide AuthorizationPolicy** required-claims are dropped: `apServiceName` falls back to the policy's own name, matching no consumer, so a namespace-wide (no `spec.selector`) or mesh-root policy never applies its claims. Nested claim keys (`claims[a][b]`) also mis-parse. | 🟡 | open | `internal/discovery/istio/istio.go` |
| KI-31 | Discovery | **Envoy discovery is order-nondeterministic** (ranges a map) and duplicate provider names in different filter blocks overwrite (last wins); provenance/confidence are only partially populated; namespace is set to the kube-context. | 🟡 | open | `internal/discovery/envoy/envoy.go` |
| KI-32 | Discovery | Several model fields are never populated by discovery: `jwtRules[].jwksUri` is parsed then **discarded**, and `Expects.Algorithms`, `JWKSBehavior.RefreshIntervalSec`, and `RefreshesOnUnknownKID` are only ever set by library-defaults enrichment, not adapters. Surfacing `jwksUri` (the rotation endpoint) is the highest-value gap. | 🟡 | open | `internal/discovery/*`, `internal/model/consumer.go` |
| KI-33 | Discovery | **Consumer naming falls back to the policy name** when the workload isn't derivable: a selector that uses a non-`app` label (e.g. `istio: ingressgateway`) or a selector-less (namespace-wide) RequestAuthentication names the consumer after the `RequestAuthentication` (`metadata.name`) instead of the workload. Surfaced by the independent OSS benchmark (2/5 real configs). Contract fields stay correct and the ID is stable, but the name is off and it feeds the KI-28 merge-axis gap. `serviceName()` should recognise more selector conventions and model selector-less policies as namespace-wide. | 🟡 | open | `internal/discovery/istio/istio.go:serviceName`, `docs/independent-benchmark.md` |
| KI-26 | Normalization | Issuer URLs are compared **verbatim** — no normalization of trailing slashes or whitespace. So `…/realms/main` vs `…/realms/main/` reads as an issuer change and pages. Surfaced by the held-out adversarial corpus (`adv-03`, `adv-06`), where it costs 2 false positives (FPR 0.5). Defensible under strict OIDC exact-match `iss` semantics, but a normalization pass would cut these FPs. | 🟡 | open | `internal/diff/diff.go`, `bench/corpus/adversarial/` |
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
