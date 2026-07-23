# Changelog

All notable changes to Keyway are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial repository scaffolding: Go module, package layout (PRD §3), CI, container build, and
  the React + TypeScript web dashboard skeleton.
- Core data model (`internal/model`): issuers, keys, consumers, contract versions, probes, change
  events (PRD §4).
- Component interfaces: `issuer.Adapter`, `discovery.Discoverer`, `store.Store` (PRD §5).
- `PROGRESS.md` implementation tracker mapped to milestones M1–M9.
- **M1** — PostgreSQL store (`internal/store/postgres`) on pgx with JSONB blobs and batched writes;
  `keyway migrate up/down` with migrations embedded in the binary (golang-migrate + iofs);
  `keyway snapshot` wired end-to-end with the mandatory baseline flow (PRD §8.2). Integration-tested
  against a real database (gated on `KEYWAY_TEST_DB`).
- **M4** — diff walker (`internal/diff/diff.go`): matches consumers by StableID, decomposes field
  changes into atomic operations, and classifies each via the §9.2/§9.3 tables. Adding one audience
  yields exactly one `widened` event (AC-7); no-op changes yield zero (AC-8).

### Changed
- Migrations moved from `/migrations` to `internal/store/postgres/migrations` so they embed into the
  single binary for in-VPC deployment.

[Unreleased]: https://github.com/architsharma/keyway/commits/main
