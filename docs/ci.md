# Keyway in CI — CLI & GitHub Action

Run Keyway's static analysis (contract discovery + drift) from your repo or
pipeline. It derives what your services expect from JWTs / agent tokens, compares
against the previous run, and **fails the build on breaking drift** — so a widened
audience or a dropped claim is caught in review, not in production.

Two ways to run, same command underneath:

| Mode | What it does | Needs |
|---|---|---|
| **Hosted / self-hosted** | Uploads config to a Keyway Cloud API; the server diffs vs the last run and keeps history + a shared report | `--server`, `--token`, `--project` |
| **Offline (local)** | Diffs against a baseline file committed to your repo — no account, no network | `--baseline` (and `--write-baseline` to create it) |

The server can be **our hosted Keyway Cloud** or **your own `keyway-cloud`** — it's
the same API. Point `--server` at whichever you run.

## CLI

```bash
go install github.com/Keyway-AI/keyway/cmd/keyway@latest
```

### Report to a Keyway Cloud server

Create a token in the project's **Connect CI / CLI** panel (or `POST /v1/tokens`),
and grab the project id from the same panel.

```bash
keyway cloud analyze \
  --server https://cloud.example.com \
  --token "$KEYWAY_TOKEN" \
  --project prj_123 \
  --path deploy/ \
  --fail-on high
```

`--server`, `--token`, and `--project` also read `KEYWAY_CLOUD_URL`,
`KEYWAY_TOKEN`, and `KEYWAY_PROJECT` from the environment.

### Offline, no account

Establish a baseline once and commit it, then diff against it in CI:

```bash
# once, and commit the result:
keyway cloud analyze --path deploy/ --write-baseline .keyway/baseline.json

# in CI:
keyway cloud analyze --path deploy/ --baseline .keyway/baseline.json --fail-on high
```

### Flags

| Flag | Default | Purpose |
|---|---|---|
| `--path` | `.` | Manifest files/dirs to scan (repeatable). Skips `node_modules`, `vendor`, `.git`. |
| `--server` | `$KEYWAY_CLOUD_URL` | Cloud API base URL. Empty ⇒ offline mode. |
| `--token` | `$KEYWAY_TOKEN` | Bearer token for `--server`. |
| `--project` | `$KEYWAY_PROJECT` | Project id on the server. |
| `--baseline` | — | Offline: contract file to diff against. |
| `--write-baseline` | — | Offline: write the derived contract to this file. |
| `--fail-on` | `high` | Exit non-zero at/above this severity: `none\|low\|medium\|high\|critical`. |
| `--json` | off | Emit the analysis as JSON. |

Exit code is `0` when drift is below the gate, `1` when it meets `--fail-on` — so
the step fails the job.

## GitHub Action

```yaml
# .github/workflows/keyway.yml
name: Keyway
on: [pull_request]
jobs:
  contract-drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: Keyway-AI/keyway@v0
        with:
          server: https://cloud.example.com     # omit for offline mode
          token: ${{ secrets.KEYWAY_TOKEN }}
          project: ${{ vars.KEYWAY_PROJECT }}
          path: deploy/
          fail-on: high
```

Offline variant (no server/token): pass `baseline: .keyway/baseline.json` instead
of `server`/`token`/`project`. The action installs the CLI, runs the analysis, and
writes the report to the job summary. A full example lives at
[`examples/github-actions/keyway.yml`](../examples/github-actions/keyway.yml).

## Agent / MCP token check

Gate an agent, MCP, or on-behalf-of **token** on the statically-checkable
agent-auth invariants — audience binding (MCP-01/02), the delegation `act` claim
and chain shape (DEL-01/02), scope minimization (SCOPE-01), and expiry (SCOPE-02).
It runs on a single token, verifies no signature, and sends nothing anywhere.

```yaml
# .github/workflows/agent-token.yml
name: Agent token
on: [pull_request]
jobs:
  agent-auth:
    runs-on: ubuntu-latest
    steps:
      - uses: Keyway-AI/keyway/actions/agent-inspect@v0
        with:
          token: ${{ secrets.AGENT_TOKEN }}        # pass via a secret; piped on stdin
          audience: https://mcp.example/api        # the resource the token must be bound to
          require-delegation: "true"               # on-behalf-of tokens must carry act
          max-lifetime: 1h                          # flag long-lived agent credentials
          allowed-scopes: files:read,tasks:write    # anything else is flagged
          max-delegation-depth: "2"                 # flag over-deep act chains
          fail-on: high                             # exit non-zero at/above this severity
          version: latest                           # pin to a release tag for reproducibility
```

Equivalent CLI (local or any CI):

```bash
echo "$AGENT_TOKEN" | keyway agent inspect \
  --audience https://mcp.example/api --require-delegation \
  --max-lifetime 1h --allowed-scopes files:read,tasks:write \
  --max-delegation-depth 2 --fail-on high
```

Each finding maps to a threat in `keyway threats coverage --domain agent`. `--json`
emits machine-readable findings.

## Tokens

CI tokens are long-lived (1 year) bearer credentials minted per user via
`POST /v1/tokens` and shown once in the UI. They authenticate the same endpoints a
signed-in browser session does, scoped to that user's projects. They are stateless
(signed, not stored), so revoking a single token isn't possible without rotating
`KEYWAY_CLOUD_SESSION_SECRET` (which invalidates all sessions and tokens) — treat
them like any CI secret. See [cloud.md](cloud.md) for the API.
