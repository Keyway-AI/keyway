# Keyway Security Audit

_Internal review of the security-sensitive surfaces. Keyway mints synthetic JWTs, operates
signing keys, and probes authentication infrastructure, so it is itself a sensitive tool._

_Date: 2026-07-24 · Scope: full codebase at the M0–M9 completion point._

## Summary

No critical or high-severity issues found. Two defence-in-depth hardening fixes were applied
(constant-time token comparison; scrubbing reflected JWTs from stored responses). The invariants
in [SECURITY.md](../SECURITY.md) hold.

## Findings & verdicts

| # | Area | Finding | Severity | Status |
|---|---|---|---|---|
| 1 | API auth | Bearer token compared with `!=` (timing side-channel) | low | **Fixed** — `crypto/subtle.ConstantTimeCompare` |
| 2 | Token persistence (OPEN-4) | A probed endpoint could reflect the synthetic token into its response body, which is stored (truncated) | low | **Fixed** — response bodies scrubbed of JWT-like strings before storage |
| 3 | Private key at rest | localkeys holds RSA private keys in memory only; never written to Postgres or logs | — | OK by design |
| 4 | Minted token storage | Tokens never persisted; only `jti` (in-memory) + probe ID recorded; `ProbeResult` stores no request material | — | OK (OPEN-4) |
| 5 | Attack tokens (`alg=none`, HS256 confusion) | Generated only to send to probe targets; Keyway never verifies tokens itself, so it cannot be attacked by them | — | OK |
| 6 | Staging guard | Probe engine is default-deny; refuses non-allowlisted hosts unless `--i-know-this-is-production` | — | OK |
| 7 | SQL injection | All queries use pgx parameterisation (`$1`…); no string interpolation | — | OK |
| 8 | Command injection (git attributor) | `exec.CommandContext("git", …)` with an argument array (no shell); the evidence path is passed after `--` so it cannot be read as a flag | — | OK |
| 9 | Secrets in config | `issuer add` stores the **env-var name** for admin credentials, never the secret value | — | OK by design |
| 10 | Web UI XSS | React escapes all rendered API data; no `dangerouslySetInnerHTML` | — | OK |
| 11 | Response size | OIDC/JWKS reads capped at 1 MB; probe response reads capped at 512 B | — | OK |
| 12 | CORS | No CORS headers → same-origin only (UI embedded in the binary) | — | OK |

## Known limitations (documented, not defects)

- **SSRF surface.** Keyway fetches operator-configured OIDC/JWKS URLs and probes discovered
  endpoints. Probe targets are gated by the staging allowlist; issuer URLs come from operator
  config. An operator who can write both the manifests and the allowlist can point Keyway at
  internal hosts — this is inherent to the tool's job and bounded by the allowlist.
- **Idempotency-Key** (PRD §12) is not yet enforced on write endpoints — a correctness/robustness
  gap, not a security one.
- **Canary key material** lives in the daemon's memory. A production deployment that needs canary
  state to survive restarts should supply the operated key from a secret manager (future work).

## Re-run

This audit is reproducible: `go test ./...` covers the auth and scrubbing fixes
(`internal/api`, `internal/probe`). Run `/security-review` against a PR diff for the automated pass.
