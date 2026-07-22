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

[Unreleased]: https://github.com/architsharma/keyway/commits/main
