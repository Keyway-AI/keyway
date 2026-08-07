# Adversarially Verifying JWT Verifiers with Minted Attack Tokens

*Keyway Research Note 03 · 2026‑08‑07 · reproduces against `v0.1.0`*

> **TL;DR** — Static analysis says what a verifier *should* reject; it cannot say
> what it *does*. Keyway mints real attack tokens — `alg=none`, HS/RS key
> confusion, `jku`/`kid` injection, expiry, tampering, forged claims — and fires
> them at a **staging** verifier. An accepted attack token is a confirmed live
> vulnerability, not a pattern match. The engine is deny‑by‑default with a hard
> production guard. 13 purpose‑built probes span the classical JWT attack surface.

## 1. Problem

The dangerous failures of JWT verification are failures of *enforcement*: a
verifier that trusts `alg: none`, that confuses an HMAC secret with an RSA public
key (RS→HS confusion), that fetches keys from an attacker‑controlled `jku`, or
that never actually checks `exp`. None of these are visible in configuration —
the config may look correct while the running verifier is broken. The only sound
test is to **present the attack and observe the verdict**.

## 2. Method

Keyway's probe engine (`internal/probe`, `keyway probe`) is a taxonomy‑driven
harness. For each threat it **mints a real, well‑formed‑looking token** carrying
the attack and sends it to a discovered consumer's endpoint. The 13 probes cover:

| Probe | Attack |
|---|---|
| valid | baseline — a correct token must be *accepted* |
| `alg=none` | unsigned token accepted? |
| alg‑confusion | RS/HS key‑type confusion |
| wrong‑issuer / wrong‑audience | trust‑boundary enforcement |
| expired | `exp` actually enforced? |
| tampered | signature integrity |
| missing‑claim | required‑claim enforcement |
| retired‑key | key rotation honoured? |
| `jku` / `kid` injection | key‑sourcing from attacker input |
| canary | a key announced but never used for signing |
| header‑bypass | pre‑auth path confusion |

The correctness oracle is **invariant‑based**, not a fixed script: the valid token
must be accepted and *every* attack token must be rejected. A single accepted
attack is a confirmed finding with the exact request that triggered it.

**Safety is a first‑class property.** Minting attack tokens against the wrong
target is itself dangerous, so the engine is **deny‑by‑default**: it runs only
against an explicit staging allow‑list, a hard guard blocks production hosts, and
stored probe responses are scrubbed of JWT‑like strings so no token material is
ever persisted (defence‑in‑depth). This is why the live layer stays in the
customer's environment and is never run from the hosted Cloud.

An end‑to‑end scoring harness (`bench/l2`, `make bench-l2`) stands up real
containerised issuers + verifiers and scores the probe layer against them, so the
enforcement claims are exercised against actual validating services, not mocks.

## 3. Why this is stronger than pattern‑matching

A static scanner that greps for `alg: none` reports a *possible* problem wherever
the string appears — including in test fixtures, comments, and safe defaults —
and stays silent when the flaw arises from library behaviour rather than a visible
token. Keyway's probe reports a problem only when a live verifier **actually
accepts** the attack. The result is a confirmed vulnerability with zero
interpretation required, at the cost of needing a reachable staging endpoint.

## 4. Threats to validity

- Probing requires a staging endpoint that faithfully mirrors production
  verification; a staging verifier configured differently from production limits
  the conclusion to staging.
- The harness tests the enumerated attack classes; novel attacks outside the
  taxonomy are, by definition, not yet probed — coverage is tracked honestly
  (Research Note 04 / `docs/threat-coverage.md`).
- Deny‑by‑default means a misconfigured allow‑list yields *no* results rather than
  unsafe ones — a deliberate bias toward safety over coverage.

## 5. Reproduce

```bash
make bench-l2     # containerised issuers + verifiers, scored (needs docker)
keyway probe --harness --path ./config   # generative harness against staging
```

Prev: **[Note 02 — Drift Classification](02-drift-classification.md)** ·
Next: **[Note 04 — Verifying AI‑Agent Authorization](04-agent-auth.md)**.
