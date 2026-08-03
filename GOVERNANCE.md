# Keyway governance

Keyway is a young open-source project with a small maintainer group. This document
says how decisions get made so contributors know what to expect. It will grow up
as the community does.

## Roles

- **Users** — anyone running Keyway. Feedback, bug reports, and discussions carry
  real weight in prioritization.
- **Contributors** — anyone who opens a PR, files a well-scoped issue, improves
  docs, or helps in Discussions. No formal status required.
- **Maintainers** — hold merge/release rights, triage issues, and are accountable
  for the invariants below. Listed in [CODEOWNERS](.github/CODEOWNERS).

## How decisions are made

- **Everyday changes** (bug fixes, docs, tests, additive features that respect the
  invariants) — one maintainer approval + green CI is enough to merge.
- **Notable changes** (new detectors, API/CLI surface, dependencies, anything
  user-visible) — open an issue or discussion first so the approach can be agreed
  before code. Lazy consensus: if no maintainer objects within a reasonable window,
  the proposal proceeds.
- **Direction / roadmap** — maintainers decide, informed by
  [Discussions](https://github.com/nometria/keyway/discussions) and issue demand.
- **Disagreement** — discuss in the open; if maintainers can't reach consensus, the
  change waits. Bias is toward *not* shipping something reversible-with-difficulty.

## Non-negotiable invariants

Any change that breaks these will not be merged, regardless of consensus (see
[CONTRIBUTING.md](CONTRIBUTING.md) for the full list): Keyway never mutates customer
config, never blocks a deploy, never requires a user-authored model file, never
persists minted tokens, and probing is deny-by-default in production.

## Becoming a maintainer

Sustained, high-quality contribution — good PRs, helpful reviews, sound judgment on
the invariants — is the path. An existing maintainer nominates; the others agree.
There is no quota.

## Releases

Releases follow [SemVer](https://semver.org) and are cut per
[RELEASING.md](RELEASING.md). Pre-1.0, breaking changes may land in minor versions
and are always called out in [CHANGELOG.md](CHANGELOG.md).

## Code of conduct

Participation is governed by the [Code of Conduct](CODE_OF_CONDUCT.md). Maintainers
are responsible for enforcing it.

## Security

Vulnerabilities follow the coordinated-disclosure process in
[SECURITY.md](SECURITY.md), never a public issue.
