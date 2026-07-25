# Keyway — Implementation Progress Tracker

This is the single source of truth for **what is built vs. pending**. Update it in every PR that
advances a tracked item. Milestones mirror PRD §15; acceptance criteria (AC-n) mirror PRD §14.

**Legend:** ✅ done · 🚧 in progress · ⬜ not started · 🔷 stub only (compiles, no real logic)

_Last updated: 2026-07-24 — **all milestones M0–M9 + UI complete and tested**; verified live; security audit + benchmarking (L1/L2/L3 + real-world CVEs) done. `oidcclient` discovery and the `serve` snapshot scheduler now shipped. Remaining polish tracked in [KNOWN_ISSUES.md](KNOWN_ISSUES.md): Idempotency-Key (KI-05), scheduled canary tightening (KI-06), L4 attribution corpus (KI-22), StableID merge across mesh/OIDC (KI-24)._

---

## Milestone status

| Milestone | Deliverable | Gate | Status |
|---|---|---|---|
| **M0** | Repo scaffold, tooling, CI, web skeleton, tracker | — | ✅ |
| **M1** | `model`, `store/postgres`, migrations, `contract/hash` | AC-1 | ✅ |
| **M2** | `issuer/keycloak` (describe + mint), `discovery/istio`, `discovery/k8s` | AC-3 | ✅ |
| **M3** | `probe` engine, probes 1–12, CLI `probe` | AC-4 | ✅ |
| **M4** | `contract/build`, snapshot, baseline flow, `diff` + `classify` | AC-1,2,7,8 | ✅ |
| **M5** | `libdefaults` DB + detection | AC-6 | ✅ |
| **M6** | `AnnounceKey`, probe 13, `canary` CLI | AC-5 | ✅ |
| **M7** | `blastradius` + grace period, CLI + API | AC-9 | ✅ |
| **M8** | `bench/harness`, 400-scenario corpus, scorecard | AC-10 | ✅ |
| **M9** | `attribution`, `notify/slack`, `api/server` | full | ✅ |
| **UI** | React dashboard over the HTTP API (PRD §12) | — | ✅ |

---

## M0 — Scaffold & tooling

- [x] `go.mod` with pinned dependencies (PRD §2)
- [x] Repository layout per PRD §3
- [x] README, LICENSE (Apache-2.0), CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, CHANGELOG
- [x] Makefile, `.golangci.yml`, `.gitignore`, `.env.example`
- [x] Dockerfile (distroless, multi-stage w/ embedded web), `docker-compose.yml`
- [x] GitHub Actions: CI (go + web + bench gate), release, dependabot, issue/PR templates
- [x] This tracker
- [x] `internal/version` build-info package
- [x] Web UI skeleton (React + TS + Vite + Tailwind) — builds, lints, renders (verified in browser)
- [x] `go build ./...`, `go vet ./...`, `go test ./...` all green
- [ ] `make dev-up` brings up Postgres + Keycloak

## M1 — Core model, store, hash (Gate: AC-1)

- [x] `internal/model` types compile with no cross-internal deps (PRD §4)
  - [x] `issuer.go`, `consumer.go`, `contract.go`, `probe.go`, `change.go`, `errors.go`
- [x] `internal/contract/hash.go` — canonical SHA-256 (PRD §8.1)
  - [x] **First test written & passing:** two runs on unchanged system → identical hash (AC-1)
- [x] `internal/store` interface (PRD §5)
- [x] `internal/store/postgres` implementation — pgx pool, JSONB blobs, batched writes; integration-tested against real Postgres (gated on `KEYWAY_TEST_DB`)
- [x] `migrations/` — schema (moved to `internal/store/postgres/migrations`, embedded into the binary)
- [x] `keyway migrate up/down` command — golang-migrate + embedded iofs; verified up & down
- [x] `keyway snapshot` wired end-to-end to the store (baseline flow verified: AC-1, AC-2)

## M2 — Issuers & discovery (Gate: AC-3)

- [x] `internal/issuer/adapter.go` interface (PRD §5)
- [x] `internal/issuer/localkeys` — JOSE key lifecycle (generate/sign/promote/retire/JWKS), tested
- [x] `internal/issuer/generic` — full local-key OIDC issuer (mint + canary), tested
- [x] `internal/issuer/keycloak` — real OIDC Describe (discovery + JWKS over HTTP) + Keyway-operated mint/canary
- [x] `internal/issuer/k8ssa` — local-key SA issuer
- [x] `internal/issuer/oidc` — shared OIDC discovery + JWKS fetch helper
- [x] `internal/discovery/discoverer.go` interface + `StableID` derivation (§4.2) + aggregator (merge by StableID)
- [x] `internal/discovery/istio` — RequestAuthentication parsing (conf 1.0)
- [x] `internal/discovery/k8s` — Services + workloads, SA token projections, env hints (conf 0.5), owner labels, endpoints
- [x] `internal/discovery/envoy` — jwt_authn providers incl. `cache_duration` → JWKS TTL
- [x] `internal/discovery/oidcclient` — Keycloak client registry via admin API (audience mappers → consumers), tested; wired into discover/snapshot/serve
- [x] `keyway discover` command (table/json) + `keyway snapshot` wired to discovery
- [x] AC-3: ≥85% consumer discovery on the reference stack, zero config files (tested); verified E2E via CLI
- [x] AC-7 end-to-end: adding an Istio audience → exactly one `widened` event through the real pipeline

## M3 — Probe engine (Gate: AC-4)

- [x] `internal/probe/mint.go` — baseline claims, raw `alg=none` build, HS256-confusion secret, signature tamper
- [x] `internal/probe/probes.go` — all 13 probe definitions with Mutate logic
- [x] `internal/probe/engine.go` — concurrency (global sem), staging guard (default deny), kill switch, inter-probe delay, consecutive-5xx abort, baseline-5xx→unverified, probe-9 expansion (cap 8)
- [x] `keyway probe` command (`--consumer`, `--probe`, `--dry-run`, `--allow`, `--i-know-this-is-production`); persists results
- [x] AC-4: all probes execute against a real JWT validator; `header_bypass` flags a service trusting `X-User-Id`; staging guard, 5xx-skip, sibling-token all tested

## M4 — Contract build, baseline, diff (Gate: AC-1,2,7,8)

- [x] `internal/contract/build.go` — assemble graph + compute hash from discovery output
- [x] `internal/contract/version.go` — `Snapshot` + baseline flow (PRD §8.2), zero-event guarantee
- [x] `internal/diff/diff.go` — match by StableID, atomic field-change decomposition, feeds Classify; fully tested
- [x] `internal/diff/classify.go` — the exact classification table (PRD §9.2) + severity, fully tested
- [x] `keyway snapshot` wired; `keyway diff` wired (resolves --from/--to or baseline→latest)
- [x] AC-2: first snapshot = baseline, zero events (verified against Postgres)
- [x] AC-7: adding an audience → exactly one `widened` event (unit-tested; needs Istio discovery for full E2E in M2)
- [x] AC-8: no-op change → zero events (unit-tested)

## M5 — Library defaults (Gate: AC-6)

- [x] `internal/libdefaults/data/defaults.yaml` — seed DB (keyfunc, node-jsonwebtoken, spring, pyjwt…)
- [x] `internal/libdefaults/db.go` — embedded load + semver-constraint matching (Masterminds/semver) + path-suffix lookup, tested
- [x] Detection from `go.mod`/`package.json`/`pom.xml`/`build.gradle`/`requirements.txt`/`Cargo.toml` (`DetectDir`, `DetectFor`, `Enrich`)
- [x] AC-6: keyfunc v1.9.0 → `refreshes_on_unknown_kid=false` from library defaults with **zero probes** (tested); v2 band → true

## M6 — Canary (Gate: AC-5)

- [x] `issuer.AnnounceKey` / `PromoteKey` / `RetireKey` (localkeys lifecycle; one active signer at a time)
- [x] Probe 13 `canary_key` (2xx expected) — wired in the engine
- [x] `keyway canary start/status/promote` commands (via the daemon API; canary state owned by `serve`)
- [x] AC-5: announced key not used to sign; probe 13 separates ready from not-ready (tested); canary flow verified over the live API

## M7 — Blast radius & grace period (Gate: AC-9)

- [x] `internal/blastradius/query.go` — all 4 proposal kinds (rotate_key/retire_key, remove_claim, change_issuer, drop_algorithm) per §10.2, with canary/library/cache evidence tiers
- [x] `internal/blastradius/graceperiod.go` — window calc, max×1.5, floor 1h ceil 30d (PRD §10.3), tested
- [x] `keyway blast-radius rotate-key / remove-claim` commands + JSON output; enriches via libdefaults; reads probe history
- [x] AC-9: <10s on a 50-consumer graph, named bounding consumer (tested); verified E2E via CLI

## M8 — Benchmark harness (Gate: AC-10)

- [x] `bench/harness/runner.go` + `score.go` (TPR/FPR/precision/recall/F1/Youden) — scores via the real `diff.Compute`
- [x] `bench/mutations/mutate.go` — generates TP (every §9.2 row) + contract-neutral no-ops
- [x] Generated corpus at ~50/50 (800 scenarios: 400 TP / 400 TN) **plus** file-based before/after scenarios that exercise real discovery (L1)
- [x] `--ci-gate` fails below PRD §13.4 thresholds (wired into CI)
- [x] AC-10: corpus emits a passing scorecard (L3 Youden=1.0, FPR=0); tested. ROC-chart export remains a nice-to-have

## M9 — Attribution, notify, API server (Gate: full)

- [x] `internal/attribution` — git commit attributor (last commit touching the evidence file), tested; K8s deploy + Keycloak admin events remain future (OPEN-5)
- [x] `internal/notify/slack.go` (Block Kit, medium+ only), `internal/notify/webhook.go` (JSON POST) — real HTTP
- [x] `internal/api/server.go` + `handlers.go` — all §12 endpoints wired to store/engines, bearer auth; tested via httptest
- [x] `keyway serve` — API + issuer registry; embeds & serves the web UI (SPA fallback); verified live
- [x] Coverage endpoint (resolved/low-confidence/unresolved); scheduled canary tightening remains future

## Web UI (PRD §12 surface)

- [x] Vite + React + TS + Tailwind skeleton, typed API client, routing
- [x] Dashboard: coverage stat tiles, latest snapshot, recent changes
- [x] Consumers table (library, JWKS cache, unknown-kid flag, filter)
- [x] Changes feed (filter by severity, old→new value, evidence pills)
- [x] Blast-radius interactive form → result view (client-side resolver mirrors §10)
- [x] Canary status board (announce action pending M6)
- [x] Wired to live API (mock fallback only when API unreachable; `keyway.live=1` forces live) — verified in-browser against the daemon
- [x] Consumer detail drawer (provenance, confidence, probe history) — click a consumer row; backed by `GET /v1/consumers/{id}/probes`

---

## Real-world validation (cited incidents)

- [x] `bench/realworld/` — reproduces documented JWT/JWKS failure modes and asserts Keyway flags each, with real citations
  - [x] RW-01 `alg=none` (CVE-2022-23540) → probe `alg_none`
  - [x] RW-02 RS256→HS256 confusion (CVE-2022-23541) → probe `alg_confusion`
  - [x] RW-03 signature not verified (CWE-347) → probe `tampered_signature`
  - [x] RW-04/05/06 missing aud/iss/exp checks (RFC 8725) → probes `wrong_audience`/`wrong_issuer`/`expired`
  - [x] RW-07 identity-header trust (CWE-290) → probe `header_bypass`
  - [x] RW-08 JWKS `RefreshUnknownKID=false` rotation outage (openfga/openfga#3099) → blast-radius `will_break`
- [x] `docs/realworld-validation.md` report; `make validate`; CI gate (fails if any regress); README badge — **8/8 detected**

## Open decisions carried from PRD §16

| ID | Decision | Default taken |
|---|---|---|
| OPEN-1 | Mobile/SPA not probeable | exclude from grace period, list under Unknown, flag loudly |
| OPEN-2 | Multi-tenant issuers | one realm per Issuer record; model tenants as separate Issuers |
| OPEN-3 | Probe 9 combinatorics | cap at 8 claims, prioritise auth-decision claims |
| OPEN-4 | Minted token storage | never persist; log only `jti` + probe ID |
| OPEN-5 | Non-git IdP attribution | v1 covers git + Keycloak admin events; others `unattributed` |
