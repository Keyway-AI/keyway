# Keyway Architecture

Keyway answers one question: **who breaks if I change this auth thing, and how long
is the safe grace period?** It does so without asking the operator to describe their
system — everything is derived and then verified.

## The four layers

Keyway is organised as four measurable layers (the benchmark harness scores each):

| Layer | What | Packages |
|---|---|---|
| **L1 — Derivation** | Build the consumer inventory automatically | `discovery/*`, `libdefaults`, `contract/build` |
| **L2 — Verification** | Prove the derived contract with real tokens | `issuer/*`, `probe/*` |
| **L3 — Diff** | Version, diff, classify changes with zero baseline noise | `contract/hash`, `contract/version`, `diff/*` |
| **L4 — Attribution** | Bind each change to its cause | `attribution/*` |

## Data flow

```
                    ┌─────────────┐
  Istio / Envoy ───▶│             │
  K8s / OIDC    ───▶│  discovery  │──┐
                    └─────────────┘  │
                    ┌─────────────┐  │   ┌──────────────┐   ┌───────────┐
  go.mod, SBOM  ───▶│ libdefaults │──┼──▶│ contract     │──▶│ hash /    │
                    └─────────────┘  │   │ build        │   │ version   │
                    ┌─────────────┐  │   └──────────────┘   └─────┬─────┘
  Keycloak /    ───▶│  issuer     │──┘          │                 │
  K8s SA            │  (mint)     │             ▼                 ▼
                    └──────┬──────┘      ┌──────────────┐   ┌───────────┐
                           │             │ probe engine │   │ diff +    │
                           └────────────▶│ (13 probes)  │   │ classify  │
                                         └──────┬───────┘   └─────┬─────┘
                                                │                 │
                                                ▼                 ▼
                                     ┌───────────────────────────────────┐
                                     │ blast radius + grace period        │
                                     └───────────────┬────────────────────┘
                                                     ▼
                                   CLI  ·  HTTP API (§12)  ·  Web UI
```

## Key design rules

- **`internal/model` has no internal dependencies.** Every other package depends on
  it, never the reverse.
- **The contract hash is canonical and volatile-field-free.** Two runs against an
  unchanged system must produce the same hash (AC-1). See
  [`internal/contract/hash.go`](../internal/contract/hash.go).
- **The baseline flow emits zero events on first run.** A wall of findings on day one
  is the documented pilot-killer (PRD §8.2).
- **Minted tokens never touch disk.** Only `jti` + probe ID identify a run.
- **Probing is deny-by-default against production.** The engine refuses non-allowlisted
  hosts unless explicitly overridden.

## Storage

PostgreSQL. Contract versions are stored as canonical JSON blobs (JSONB) with hot query
columns (`hash`, `is_baseline`, `created_at`) promoted alongside. Change events and probe
results are first-class tables. Schema: [`internal/store/postgres/migrations/`](../internal/store/postgres/migrations)
(embedded into the binary so no external migration files ship).

## Deployment shape

A single static binary. `keyway` is the CLI; `keyway-runner` is the same binary in daemon
mode (`serve`). No dependencies beyond Postgres. Ships as a distroless image into the
customer VPC.
