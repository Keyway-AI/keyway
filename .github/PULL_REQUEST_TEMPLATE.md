## What & why

<!-- What does this change and why? Link the issue and the PRD section / milestone. -->

Closes #

## Checklist

- [ ] `make check` passes (fmt + vet + lint + test)
- [ ] Added/updated tests
- [ ] Added/updated benchmark scenarios in `bench/corpus/` if this touches discovery, probes, or diff
- [ ] `make bench` scorecard still meets PRD §13.4 thresholds
- [ ] Updated `docs/progress.md` / `CHANGELOG.md` if a tracked item advanced
- [ ] Respects the invariants: no config mutation, no deploy blocking, no user-authored model file, no token persistence
