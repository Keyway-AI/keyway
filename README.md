<div align="center">

# 🔑 Keyway

**Know which consumers will break _before_ you rotate a signing key, change an issuer, or drop a claim.**

[![CI](https://github.com/nometria/keyway/actions/workflows/ci.yml/badge.svg)](https://github.com/nometria/keyway/actions/workflows/ci.yml)
[![Detection: 100% TPR · 0% FPR](https://img.shields.io/badge/detection-100%25_TPR_%C2%B7_0%25_FPR-2f7fe0)](BENCHMARK.md)
[![Real-world: 8/8 documented risks](https://img.shields.io/badge/real--world-8%2F8_CVEs_%26_incidents-16a34a)](docs/realworld-validation.md)
[![Go Reference](https://pkg.go.dev/badge/github.com/nometria/keyway.svg)](https://pkg.go.dev/github.com/nometria/keyway)
[![Go Report Card](https://goreportcard.com/badge/github.com/nometria/keyway)](https://goreportcard.com/report/github.com/nometria/keyway)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

</div>

---

Keyway derives your JWT **consumer inventory automatically** — from Istio, Envoy, Kubernetes,
OIDC discovery and a shipped library-behaviour database — and **verifies it with real tokens
against real staging endpoints** using 13 purpose-built probes. It versions the derived
contract, diffs it on every change, classifies each change as _widened_ or _narrowed_, and
answers the one question that actually matters before a key rotation:

> **"If I rotate this key, who breaks, and how long is the safe grace period?"**

Keyway **never mutates your configuration, never blocks a deploy, and never asks you to
author a model file.** If a feature needs you to describe your own system, it is out of scope
by definition.

## Why

Key rotations, issuer migrations and claim removals cause outages because nobody has an
accurate, current map of _who validates what_. That map is normally tribal knowledge that
rots. Keyway rebuilds it on every run and, crucially, **proves it** by minting synthetic
tokens (expired, wrong-issuer, `alg=none`, tampered, canary, …) and watching how each
consumer responds.

## What it does

| Capability | How |
|---|---|
| **Auto-discovers consumers** | Istio `RequestAuthentication`, Envoy `jwt_authn`, K8s projected SA tokens, OIDC/Keycloak client registry |
| **Verifies with real tokens** | 13 probes (valid, expired, wrong-issuer/audience, `alg=none`, alg-confusion, tampered, missing-claim, retired-key, canary, header-bypass, …), staging-only, with a hard production guard |
| **Versions & diffs the contract** | Canonical SHA-256 hash; identical systems produce identical hashes; a first run establishes a **baseline with zero alerts** |
| **Classifies changes** | widened / narrowed / neutral / unknown, with severity |
| **Answers blast radius** | `keyway blast-radius rotate-key --issuer … --kid …` → who breaks, who's ready, recommended grace period, bounding consumer |
| **Runs a canary key** | Announces a key in JWKS _without signing_, then measures which consumers pick it up |
| **Web dashboard** | React + TypeScript UI over the HTTP API |

## Quickstart

```bash
# 1. Bring up Postgres (+ a reference Keycloak) for local dev
make dev-up

# 2. Build the binaries
make build

# 3. Point Keyway at your cluster / issuers
./bin/keyway init
./bin/keyway issuer add --type keycloak --url https://kc.example.com/realms/main \
    --admin-credential-env KC_ADMIN

# 4. Discover consumers and snapshot the contract (first run = baseline, zero alerts)
./bin/keyway discover --namespace default
./bin/keyway snapshot

# 5. The demo
./bin/keyway blast-radius rotate-key --issuer keycloak-main --kid rsa-2026-01
```

```
Rotating rsa-2026-01 on keycloak-main affects 47 consumers.

WILL BREAK (3)
  payments-api          48h JWKS cache, RefreshUnknownKID=false   [probe:canary_key #8812]
                        owner: team-payments
  legacy-reporting      no JWKS refresh configured                [lib:keyfunc v1.9.0]
                        owner: team-data
  mobile-gateway        cached key pinned in config               [istio:RequestAuthentication/mobile-gw]
                        owner: team-mobile

READY (41)   run with --verbose to list
UNKNOWN (3)  insufficient evidence — not probeable

RECOMMENDED GRACE PERIOD: 9d 6h
  bound by payments-api (48h cache, measured 6d4h to pick up canary, x1.5 margin)
  NOTE: 3 consumers unknown — treat as a lower bound.
```

## Web UI

```bash
make serve        # starts the API + scheduler on :8080
make web-dev      # starts the Vite dev server on :5173 (proxies /v1 → :8080)
```

## Architecture

```
 discovery ──┐
             ├─▶ contract build ─▶ hash/version ─▶ diff ─▶ classify ─▶ notify
 issuers  ───┤          │                                     ▲
             │          ▼                                     │
 libdefaults │       probe engine (13 probes) ────────────────┘
             │          │
             └──────────┴─▶ blast radius + grace period ─▶ CLI / HTTP API / Web UI
```

See [`docs/architecture.md`](docs/architecture.md) for the full design and
[`rollover-prd-v0.2-implementation.md`](rollover-prd-v0.2-implementation.md) for the
complete buildable specification.

## How accurate is it?

On a corpus of **805 realistic before/after changes** (half real contract
changes, half ordinary redeploy noise), Keyway catches **100% of real changes**
with **0% false alarms** — including a "noisy redeploy" that churns six unrelated
things at once. See [**BENCHMARK.md**](BENCHMARK.md) for the plain-English study
and market comparison, [docs/benchmark.md](docs/benchmark.md) for methodology, and
reproduce it yourself:

```bash
make bench            # scorecard
make bench-report     # + an interactive report.html
```

And it's validated against **real, documented incidents** — `alg=none`
([CVE-2022-23540](https://nvd.nist.gov/vuln/detail/CVE-2022-23540)), RS256→HS256
confusion ([CVE-2022-23541](https://nvd.nist.gov/vuln/detail/CVE-2022-23541)), and
the JWKS key-rotation outage from
[openfga/openfga#3099](https://github.com/openfga/openfga/issues/3099). Keyway
detects **8 of 8** — see [docs/realworld-validation.md](docs/realworld-validation.md):

```bash
make validate         # reproduce each cited incident and check Keyway flags it
```

CI fails the build if accuracy drops below the PRD §13.4 thresholds **or** if
Keyway stops detecting any documented real-world risk.

## Project status

Keyway is under active construction. Track exactly what is built vs. pending in
[**PROGRESS.md**](PROGRESS.md). Milestones follow §15 of the PRD (M1–M9).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Keyway ships a 400-scenario benchmark harness
(`bench/`) that gates accuracy in CI — new discovery/probe/diff logic is expected to keep the
scorecard above the thresholds in the PRD §13.4.

## Security

Keyway mints synthetic tokens and talks to auth infrastructure. Please read
[SECURITY.md](SECURITY.md) before running it, and **never** point it at production without the
explicit `--i-know-this-is-production` flag. Report vulnerabilities per SECURITY.md.

## License

[Apache License 2.0](LICENSE).
