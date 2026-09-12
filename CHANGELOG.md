# Changelog

All notable changes to Keyway are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[RELEASING.md](RELEASING.md) for how a release is cut.

## [Unreleased]

## [0.2.0] - 2026-09-12

Adoption-focused release: get from zero to a real result in one command, and close
the last gap between the shipped analyzer and the agent-auth threat model.

### Added
- **One-line installer** — `curl -fsSL https://raw.githubusercontent.com/Keyway-AI/keyway/main/install.sh | sh`
  detects your OS/arch, downloads the matching release, verifies its SHA-256, and
  installs `keyway`. Documented alongside every other method in
  [docs/install.md](docs/install.md).
- **Windows binaries** — releases now ship `windows/amd64` and `windows/arm64`
  (`.zip`) in addition to Linux/macOS.
- **`keyway agent inspect --demo`** — inspect a built-in, deliberately-insecure
  sample token so the very first run shows findings with zero input.
- **`keyway agent inspect --fail-on none|low|medium|high|critical`** — exit
  non-zero when a finding meets the threshold, so it gates CI.
- **`agent-inspect` GitHub Action** (`Keyway-AI/keyway/actions/agent-inspect@v0`)
  with a copy-paste [example workflow](examples/github-actions/agent-token.yml).
- **Browser playground** (`playground/`) — the agent analyzer compiled to
  WebAssembly as a static page: paste a token, see the findings, and the token
  never leaves the browser. Same code as the CLI; build with `make playground`.
- **Static DEL-02 detection** — the agent analyzer now flags a malformed or
  over-deep delegation `act` chain (`--max-delegation-depth`), so it covers all
  six statically-checkable agent-auth threats (MCP-01/02, DEL-01/02, SCOPE-01/02).

### Changed
- README leads with "Who is this for?", an install block, and the low-friction
  agent/MCP token check before the heavier discover→probe→blast-radius flow.

### Research / benchmarks (developer-facing)
- Paper A measurement instrument: prevalence with Wilson CIs, kind-aware recall,
  a non-circular negative-control precision test, Helm/kustomize resolution
  (`--resolve-templates`), canonical dedup + a near-duplicate diagnostic, weakness
  co-occurrence, the static-vs-runtime frontier (RQ4), and drift-direction analysis.
- **`make reproduce`** — one command reproduces the papers' offline-verifiable
  claims into `artifact/`; see [REPRODUCE.md](REPRODUCE.md).

## [0.1.0] - 2026-08-07

First public release.

### Added
- Repository scaffolding: Go module, package layout (PRD §3), CI, container build, and
  the React + TypeScript web dashboard.
- Core data model (`internal/model`): issuers, keys, consumers, contract versions, probes, change
  events (PRD §4), and the component interfaces `issuer.Adapter`, `discovery.Discoverer`,
  `store.Store` (PRD §5).
- **M1** — PostgreSQL store (`internal/store/postgres`) on pgx with JSONB blobs and batched writes;
  `keyway migrate up/down` with migrations embedded in the binary; `keyway snapshot` wired
  end-to-end with the mandatory baseline flow (PRD §8.2).
- **M2–M9** — issuers (local JOSE key lifecycle, Keycloak/OIDC describe), file-based and in-cluster
  discovery (Istio/Envoy/K8s + StableID + merge), the 13-probe engine with a staging guard,
  library-defaults detection, the diff walker + widened/narrowed classification, the canary key
  flow, blast radius + measured grace period, git attribution, Slack/webhook notifiers, and the
  full HTTP API + `keyway serve` daemon that also serves the web dashboard. Every acceptance
  criterion (AC-1…AC-10) is covered by tests.
- **Threat coverage** — a cited threat taxonomy and coverage report (`keyway threats coverage`,
  `GET /v1/threats/coverage`) that measures detection against the documented universe of JWT and
  agent-auth threats (RFC 8725, OWASP, CVEs) rather than a self-authored corpus.
- **Generative attack harness** (`internal/attack`, `keyway probe --harness`) that mints
  adversarial tokens — `alg=none`, HS/RS key confusion, `jku`/`kid` injection, expiry and forged
  claims — and proves a correct verifier rejects each one.
- **AI-agent auth** — a static agent-token analyzer (`keyway agent inspect`,
  `POST /v1/agent/inspect`) checking MCP audience binding, on-behalf-of delegation (`act`), scope
  minimization and expiry, plus an agent-auth threat taxonomy and live attack corpus.
- **Web experience** — an "ink & emerald" design system; a marketing site with real product
  mockups and a SaaS shell (login, signup, pricing, features, contact); a responsive app with a
  persistent sidebar; the agent inspector and coverage surfaces; and a dashboard
  verification-coverage panel. First web test suite (Vitest) and a real health-polling hook.
- **Zero-config demo** — `keyway serve` and the container image fall back to an in-memory store
  (with a warning) when no database is configured, so the app and UI run out of the box;
  `make demo` and the "Try it" quickstart.
- **Keyway Cloud** (`cloud/`, `keyway-cloud`) — a multi-tenant hosted layer that runs the static
  half of Keyway (discovery → contract → drift → threat coverage) on connected or uploaded repos.
  Reuses the exact engine; cookie-session auth, GitHub OAuth, tenant isolation. The live half
  (probing, canary, blast radius) stays self-hosted by design — the cloud never handles signing
  keys. In-memory store by default with a Postgres drop-in via the `cloud.Store` interface.
  Frontend: sign-in, projects dashboard, and per-project contract/drift/history. See
  [docs/cloud.md](docs/cloud.md).
- **CI: CLI + GitHub Action** — `keyway cloud analyze` derives the contract from repo manifests and
  either reports to a Keyway Cloud API (hosted or self-hosted) or runs fully offline against a
  committed baseline, failing the build on drift at/above `--fail-on`. Composite GitHub Action
  (`uses: Keyway-AI/keyway@v0`). Long-lived CI bearer tokens via `POST /v1/tokens`. See
  [docs/ci.md](docs/ci.md).

### Design
- A luminous, swishX-inspired treatment (warm "aurora" grounds, frosted glass panels, glowing
  accent CTAs, heavier display type) layered onto the ink-&-emerald identity across the marketing
  hero and the Cloud surfaces.

### Changed
- Migrations embed into the binary (`internal/store/postgres/migrations`) for in-VPC deployment.
- CLI resolves the store DSN in precedence `--db` → `KEYWAY_DB_URL` → `db_url` in the config file,
  matching `serve` (previously the one-shot commands ignored the config file).

### Security
- Constant-time API bearer-token comparison (`crypto/subtle`) to remove a timing side-channel.
- Stored probe response bodies are scrubbed of JWT-like strings so no token material is ever
  persisted, even if a probed endpoint reflects the synthetic token (defence-in-depth, OPEN-4).
- The probe engine is deny-by-default with a staging allowlist and a hard production guard.
- Security audit: [docs/security-audit.md](docs/security-audit.md).

[Unreleased]: https://github.com/Keyway-AI/keyway/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Keyway-AI/keyway/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Keyway-AI/keyway/releases/tag/v0.1.0
