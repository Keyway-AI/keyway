# Deriving Token‑Auth Contracts from Deployment Configuration

*Keyway Research Note 01 · 2026‑08‑07 · reproduces against `v0.1.0`*

> **TL;DR** — What a service expects from a JWT is an implicit contract scattered
> across Istio, Envoy, Kubernetes, and IdP config, plus the *defaults* of whatever
> library does the verifying. We show that this contract can be derived
> automatically, with per‑field provenance, and reduced to a content hash so that
> "did the contract change?" becomes a deterministic equality check. On 26 real
> configuration files the discovery layer recovers the intended consumer inventory
> with no false extractions (`make bench`, L1: TP=26, FP=0).

## 1. Problem

A JWT is accepted or rejected by a *verifier*, whose behaviour is defined by:
the issuers it trusts, the audiences it binds to, the algorithms it allows, the
claims it requires, and how it fetches and caches signing keys. This is a
**contract** between token issuers and token consumers — and it is almost never
written down in one place. It lives in `RequestAuthentication` CRDs, Envoy filter
chains, OIDC discovery documents, Keycloak client registries, and, critically, in
the *defaults* of the verifying library (a clock skew of ±60s here, a 5‑minute
JWKS cache there) that no config file states.

Asking engineers to declare this inventory by hand does not scale and goes stale.
We derive it instead.

## 2. Method

A **consumer** is any entity that validates tokens. Keyway's `discovery` package
composes independent *discoverers*, each mapping one configuration source to a set
of consumers:

- **Istio** — `RequestAuthentication` (issuer, audiences, `jwksUri`) and
  `AuthorizationPolicy` (required claims / `when` conditions);
- **Envoy** — JWT‑authentication HTTP filters;
- **Kubernetes** — workload metadata, yielding a **StableID** so the same service
  is tracked across snapshots even as pods churn;
- **OIDC / Keycloak** — discovery documents and client registries;
- **in‑cluster** — live CRDs via client‑go, for what static files miss.

Two design choices make the output trustworthy rather than a best guess:

1. **Provenance on every field.** Each extracted value records `{source, locator,
   observed_at, confidence}`. A downstream drift finding can therefore point at
   the exact file and line that caused it — no unattributable alarms.
2. **A library‑defaults database.** The contract must capture what the verifier
   does *by default*, not only what config overrides. Keyway merges a semver‑keyed
   defaults DB (clock skew, JWKS cache TTL, algorithm allow‑lists) so the derived
   contract reflects real runtime behaviour.

Consumers are then normalised into a `ContractVersion` — a graph of issuers,
consumers, and edges — reduced to a **content hash**. Identical hash ⇒ identical
contract ⇒ no‑op.

## 3. Results

Reproduced with `make bench` (layer L1, `bench/out/scorecard.json`):

| Metric | Value |
|---|---|
| Real config files | 26 |
| Consumers correctly recovered (TP) | 26 |
| False extractions (FP) | 0 |

Discovery also feeds the **L3‑realistic** layer, where 400 pairs are rendered to
YAML and run through the *real* discovery engine before diffing; that layer scores
perfectly too (TP=240, FP=0, TN=160), confirming discovery is faithful enough to
support drift detection end‑to‑end, not just in isolation.

## 4. Threats to validity

- Discovery reads **configuration**, so a verifier whose true behaviour diverges
  from both its config and its library defaults is invisible to this layer — that
  is what the adversarial probe layer (Research Note 03) exists to catch.
- The library‑defaults DB is only as current as its entries; an unknown library
  falls back to conservative assumptions and lower confidence.
- The 26‑file figure demonstrates faithful extraction on curated inputs; breadth
  across the long tail of real‑world config is an ongoing corpus effort
  (`docs/independent-benchmark.md`).

## 5. Reproduce

```bash
make bench                       # L1 discovery + L3 layers
keyway discover --path ./config  # derive consumers from your own manifests
```

Next: **[Research Note 02 — Zero‑False‑Alarm Drift Classification](02-drift-classification.md)**.
