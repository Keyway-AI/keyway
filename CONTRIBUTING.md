# Contributing to Keyway

Thanks for helping build Keyway. This guide covers the local workflow and the bar for merging.

## Ground rules from the product spec

Keyway has three hard invariants (PRD §1.2). A change that breaks any of them will not be merged:

1. **Keyway never mutates customer configuration.** Discovery and probing are read-only against
   config; the only writes are to Keyway's own Postgres and to a Keyway-controlled canary key.
2. **Keyway never blocks a deploy** and never judges whether an authorization rule is *correct*.
3. **Keyway never requires the user to author a model file.** If your feature needs the user to
   describe their own system, it is out of scope.

Two more that keep the tool safe to run:

4. **Never persist minted tokens.** Log only `jti` and probe ID (PRD OPEN-4).
5. **Production is deny-by-default.** Probing refuses to run unless the host is allowlisted or
   `--i-know-this-is-production` is passed.

## Local setup

```bash
git clone https://github.com/Keyway-AI/keyway
cd keyway
make dev-up          # Postgres + reference Keycloak
make build
make check           # fmt + vet + lint + test — must be green before you push
```

Go 1.25+ and Node 20+ are required.

## Development workflow

1. Open (or claim) an issue describing the change.
2. Branch from `main`: `git checkout -b feat/<short-name>`.
3. Keep the change scoped to one milestone / concern. See [docs/progress.md](docs/progress.md) for where
   things stand and [ARCHITECTURE.md](ARCHITECTURE.md) for the design.
4. Add tests. New discovery adapters, probes and diff rules **must** ship benchmark scenarios in
   `bench/corpus/` and keep the scorecard above the PRD §13.4 thresholds.
5. Run `make check` and `make bench`.
6. Update `docs/progress.md` if you completed or advanced a tracked item.
7. Open a PR using the template. CI must pass.

## Code style

- Standard `gofmt -s`; `golangci-lint` config in `.golangci.yml` is authoritative.
- Package layout follows PRD §3 — put new code in the package that owns the concern.
- `internal/model` has **no dependencies on other internal packages**. Keep it that way.
- Public request/response types live in `pkg/apitypes`.
- Errors are values: wrap with `fmt.Errorf("...: %w", err)`, define sentinels in the package that
  owns them (e.g. `model.ErrNoPrivateKey`).

## Commit messages

Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`). The subject is
imperative and under ~72 chars.

## Reporting security issues

Do **not** open a public issue for vulnerabilities. See [SECURITY.md](SECURITY.md).
