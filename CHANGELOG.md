# Changelog

All notable changes to Keyway are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[RELEASING.md](RELEASING.md) for how a release is cut.

## [Unreleased]

## [0.1.0] - 2026-07-31

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

[Unreleased]: https://github.com/Keyway-AI/keyway/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Keyway-AI/keyway/releases/tag/v0.1.0
