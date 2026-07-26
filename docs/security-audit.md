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
  gap, not a security one. _(Addressed later: enforced via SEC-05, and now backed by the
  `coordination.IdempotencyStore` seam — in-memory single-node, or Postgres-shared across replicas.)_
- **Canary key material** lives in the daemon's memory. A production deployment that needs canary
  state to survive restarts should supply the operated key from a secret manager (future work).
  _(Addressed later: a Postgres-backed `keystore.Store` shares/persists the encrypted material
  across replicas, and the AES root key is sourced from a secret manager — env, a mounted-secret
  file, or a `vault`/`aws`-style command — never the config file. See "HA infrastructure" below.)_

## Re-run

This audit is reproducible: `go test ./...` covers the auth and scrubbing fixes
(`internal/api`, `internal/probe`). Run `/security-review` against a PR diff for the automated pass.

---

# Audit v2 — deep pass

_Date: 2026-07-24 · Scope: full codebase at the post-M9 / KI-backlog point._
_Method: automated (`govulncheck` reachable-CVE scan + `gosec` static analysis) plus three
parallel manual reviews across the HTTP/API surface, the issuer/probe/crypto surface, and the
discovery/attribution/store surface._

## Summary

Two **reachable** dependency CVEs were found and fixed by upgrade. Six code-level hardening
issues were found and fixed — one high (staging-guard bypass), two medium (SSRF via redirects,
K8s-attributor path traversal), and three low/medium (idempotency key binding + memory cap,
request-body limits, server timeouts), plus two low config-hygiene fixes (admin creds require
https; `init` no longer captures ambient env secrets). No issue allowed forging a token Keyway
would trust, and the [SECURITY.md](../SECURITY.md) invariants continue to hold.

## Dependency CVEs (reachable → fixed)

| CVE | Package | Impact | Fix |
|---|---|---|---|
| GO-2026-4945 | `github.com/go-jose/go-jose/v4` | JWE parsing panic (DoS) — reachable via JWKS parsing | Upgrade v4.0.5 → **v4.1.4** |
| GO-2026-5970 | `golang.org/x/text` | Infinite-loop DoS — reachable via transitive use | Upgrade v0.31.0 → **v0.39.0** |

`govulncheck ./...` now reports **no vulnerabilities**.

## Findings & verdicts

| # | Area | Finding | Severity | Status |
|---|---|---|---|---|
| SEC-01 | Staging guard | `HostAllowed` used a substring match, so an allowlist of `example.com` also permitted `example.com.attacker.net` (and `notexample.com`) | **high** | **Fixed** — exact match or dot-boundary suffix (`host == a` or `strings.HasSuffix(host, "."+a)`), `internal/probe/engine.go` |
| SEC-02 | SSRF (probe) | The probe HTTP client followed redirects, so an allowlisted target could 302 a minted-token request to an internal address | medium | **Fixed** — `CheckRedirect` returns `http.ErrUseLastResponse` (fail-closed), `internal/probe/engine.go` |
| SEC-03 | SSRF (issuer) | The OIDC discovery/JWKS client followed redirects, so a compromised/misconfigured issuer could redirect a fetch to an internal host | medium | **Fixed** — same fail-closed `CheckRedirect`, `internal/issuer/oidc/oidc.go` |
| SEC-04 | Path traversal | `K8sDeployAttributor` joined an evidence path (which can carry an attacker-influenced manifest name) under its root with no confinement; an absolute path or `..` escaped it and read arbitrary files | medium | **Fixed** — reject absolute paths and `..`, verify containment after join, `internal/attribution/k8s.go` (test `TestK8sDeployPathTraversal`) |
| SEC-05 | Idempotency | The cache key was the raw `Idempotency-Key` header only, so the same key on a different endpoint/body could replay an unrelated response; and the map only swept expired entries (no hard cap → unbounded growth) | medium | **Fixed** — key = `sha256(key‖method‖path‖body)`; hard FIFO cap at 4096, `internal/api/idempotency.go` (test `TestIdempotencyBoundToBody`) |
| SEC-06 | Resource limits | Authenticated request bodies had no size cap (memory-exhaustion DoS) | low | **Fixed** — `maxBytes(1 MiB)` middleware via `http.MaxBytesReader`, `internal/api/server.go` |
| SEC-07 | Resource limits | `http.Server` set only `ReadHeaderTimeout` (slow-loris on body/response) | low | **Fixed** — added `ReadTimeout`/`WriteTimeout`/`IdleTimeout`, `internal/api/server.go` |
| SEC-08 | Cleartext creds | The Keycloak client-registry discoverer would send admin-cli credentials over plain `http` | low | **Fixed** — realm URL must be `https` (localhost exempted), `internal/discovery/oidcclient/oidcclient.go` |
| SEC-09 | Secret capture | `keyway init` / `issuer add` round-tripped `config.Default()`/`Load()`, folding an ambient `KEYWAY_API_TOKEN` / Slack webhook / DB URL into the written file | low | **Fixed** — `config.Scaffold()` (no env secrets) for `init`; `config.LoadFile()` (no env overlay) for read-modify-write, `internal/config/config.go` |

## Surfaces reviewed and cleared (no change needed)

- **SQL injection** — every query is pgx-parameterised; no string interpolation anywhere in `internal/store/postgres`.
- **Command injection (git attributor)** — `exec.CommandContext("git", …)` with an arg array, evidence passed after `--`; no shell.
- **Private keys** — held in memory only; JWKS/serialisation emit public material exclusively.
- **Minted / attack tokens** — never persisted (only `jti` + probe ID); Keyway never verifies tokens itself, so `alg=none`/HS256-confusion tokens cannot be turned against it (OPEN-4).
- **notify / oidcclient endpoints** — operator-controlled (Slack webhook, realm URL); bounded by config.
- **SPA / web UI** — React escapes all API data; no `dangerouslySetInnerHTML`; assets embedded, same-origin only.
- **Auth** — bearer token compared in constant time (`crypto/subtle`).

## Re-run (v2)

```
go get github.com/go-jose/go-jose/v4@v4.1.4 golang.org/x/text@v0.39.0 && go mod tidy
go build ./... && go vet ./... && go test ./...
govulncheck ./...   # expect: No vulnerabilities found.
```

Regression tests: `TestK8sDeployPathTraversal` (`internal/attribution`) and
`TestIdempotencyBoundToBody` (`internal/api`) lock in SEC-04 and SEC-05.

---

# HA infrastructure upgrade

_Date: 2026-07-26 · Scope: the audits' documented future-work / known-limitation
cases (canary key material in memory; idempotency durability) plus the
architecture review's open scale/HA recommendations (#5–#7)._

The audits found no open defects here — these were **documented limitations**.
The infrastructure now accommodates them:

| Case (source) | Upgrade |
|---|---|
| Idempotency state in-process only (audit v1 known-limitation; KI-05) | Storage is the `coordination.IdempotencyStore` seam. Default in-memory (single-node); `--db <postgres>` shares idempotent-write replays across replicas via a `keyway_idempotency` table. Same-node in-flight coalescing (KI-29) is unchanged. |
| Canary key material in daemon memory (audit v1/v2 known-limitation; KI-09) | New Postgres-backed `keystore.Store` (`--key-store-db`) stores AES-256-GCM ciphertext in `keyway_operated_keys`, so canary state is durable **and shared** across replicas. |
| Root key should come from a secret manager (audit v2 future-work) | `keystore.ResolveKey` sources the 32-byte AES key from a secret manager, in precedence: `$KEYWAY_KEY_ENCRYPTION_KEY_CMD` (a command, e.g. `vault kv get -field key secret/keyway`) > `--key-encryption-key-file` (a mounted secret) > `$KEYWAY_KEY_ENCRYPTION_KEY`. It fails closed. |
| Single-daemon scheduler could double-fire under multiple replicas (arch W5/#6) | `coordination.Leader` gates the scheduler. In-memory = always leader (single node); Postgres uses a session-level advisory lock (`pg_try_advisory_lock`) held on a dedicated connection, so exactly one replica snapshots per tick and a standby takes over automatically if the leader dies. |
| Wiring sprawl in `serve.go` (arch W6/#5) | `internal/app.Build(cfg)` is the single composition root; `serve.go` is flag-parsing + one `Build` call. |
| Duplicated `Attributor` interface (arch W7) | One `ports.Attributor`; `contract.Attributor` and `attribution.Attributor` are aliases. |

**Security posture.** No new externally-reachable surface: the coordination and
key-store backends are operator-configured (a DSN, a secret-manager reference).
Private key material is still never written in plaintext — the Postgres keystore
encrypts exactly as `FileStore` does, under a root key that now lives in a secret
manager rather than only an env var. The advisory-lock leader and idempotency
rows carry no secrets. `govulncheck ./...` remains clean.
