# Benchmark corpus

Each scenario is a `scenario.yaml` (PRD §13.1) plus, eventually, a `docker-compose.yaml`
that stands up the issuer + consumers. The harness runs Keyway's discover → snapshot →
diff pipeline against each and compares to `expected`.

## Composition (PRD §13.2)

Mirror OWASP Benchmark: roughly **50% true positives** (a real contract change) and
**50% no-ops** (dependency bump, comment edit, replica count, unrelated env var). Without
the no-op half the FPR is meaningless.

Target for v1: **400 scenarios** — 200 mutations across every classification-table row,
200 no-ops.

## Naming

`NNNN-short-description/` — `0001`–`0199` reserved for mutations, `0200`+ for no-ops.

Two seed scenarios are included:

- `0042-audience-widened/` — true positive, class `widened`.
- `0201-dependency-bump-noop/` — no-op, expects zero events.

See [docs/progress.md](../../docs/progress.md) M8 for the build-out plan.
