# Keyway Cloud (hosted layer)

`keyway-cloud` is the **multi-tenant hosted API** that wraps the open-source engine
with accounts, projects, and persisted analysis history — so the *static,
config-driven* half of Keyway can run as a SaaS on repositories users connect or
upload. It reuses the exact discovery → contract → diff → threat logic the CLI and
`keyway serve` use (`cloud/analyze.go` calls `discovery.Run` + `contract.Build` +
`diff.Compute`).

**What it does not do — by design.** The *live* half of Keyway (minting tokens,
probing staging endpoints, canary keys, measured blast radius) needs the customer's
real issuers and staging traffic, so it stays **self-hosted** (`keyway serve` in
their environment). The cloud never handles a customer's signing keys.

## Run it

```bash
# local, no GitHub needed (passwordless dev login):
KEYWAY_CLOUD_DEV_LOGIN=1 go run ./cmd/keyway-cloud     # :8090

# with GitHub sign-in:
GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=... \
KEYWAY_CLOUD_BASE_URL=https://cloud.example.com \
KEYWAY_CLOUD_FRONTEND_URL=https://keyway-azure.vercel.app \
go run ./cmd/keyway-cloud
```

### Environment

| Var | Default | Purpose |
|---|---|---|
| `KEYWAY_CLOUD_ADDR` | `:8090` | Listen address |
| `KEYWAY_CLOUD_BASE_URL` | `http://localhost:8090` | Public URL (builds the OAuth redirect; `https://` → Secure cookies) |
| `KEYWAY_CLOUD_FRONTEND_URL` | `http://localhost:5173` | Where to send the user after login |
| `KEYWAY_CLOUD_ALLOWED_ORIGINS` | = frontend URL | CORS allowlist (comma-separated) |
| `KEYWAY_CLOUD_SESSION_SECRET` | random | HMAC key for session cookies (**set in production** or sessions reset on restart) |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | — | GitHub OAuth app |
| `KEYWAY_CLOUD_DEV_LOGIN` | off | Enable passwordless `/v1/auth/dev-login` (local only) |

The GitHub OAuth app's callback URL must be `${KEYWAY_CLOUD_BASE_URL}/v1/auth/github/callback`.

## API

Public:
- `GET  /healthz`
- `GET  /v1/config` — `{github_login, dev_login}` so the UI knows what's enabled
- `GET  /v1/auth/github/login` → GitHub · `GET /v1/auth/github/callback`
- `POST /v1/auth/logout`
- `POST /v1/agent/inspect` — stateless agent/JWT token analysis (real)
- `GET  /v1/threats/coverage` — the cited threat taxonomy

Authenticated (session cookie):
- `GET  /v1/me`
- `GET/POST /v1/projects` · `GET/DELETE /v1/projects/{id}`
- `POST /v1/projects/{id}/analyze` — body `{manifests:{path:content}}` (upload) or empty to sync a connected repo → runs the engine, diffs vs the last analysis, persists
- `GET  /v1/projects/{id}/analyses` — history · `GET /v1/analyses/{id}`

Every project/analysis read is scoped to the requesting user (tenant isolation).

## Hosting notes

- **Persistence.** The default `MemoryStore` is per-process (dev). For hosting,
  implement the `cloud.Store` interface against Postgres — every method is already
  tenant-scoped, so it's a drop-in. (The OSS core already ships a Postgres store
  under `internal/store/postgres` to model after.) GitHub OAuth tokens are kept in
  memory today; a Postgres store should encrypt them at rest.
- **Cross-site cookies.** With an HTTPS `BASE_URL`, cookies are `Secure` +
  `SameSite=None` so the Vercel frontend can authenticate against the API on a
  different host; add the frontend origin to `ALLOWED_ORIGINS`.
- **Container.** Build `./cmd/keyway-cloud` into any container and run behind
  TLS (Fly.io / Railway / Render / your own).
