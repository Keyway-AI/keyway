# Verifying AI‑Agent Authorization: the Layer Nobody Tests

*Keyway Research Note 04 · 2026‑08‑07 · reproduces against `v0.1.0`*

> **TL;DR** — AI agents carry OAuth/MCP tokens to act on a user's behalf, and the
> authorization around them is new, high‑stakes, and largely unverified. We apply
> the auth‑contract model to agent tokens: audience binding (RFC 8707 / 9728),
> on‑behalf‑of delegation (RFC 8693), scope minimisation, and expiry — each finding
> cited to its spec. Coverage of a cited **15‑threat** agent taxonomy is **40%
> (6/15)**, with every gap named. The check set runs statically on a single token,
> in the browser or in CI, with nothing sent anywhere.

## 1. Problem

When an agent calls a tool or an MCP server on a user's behalf, a bearer token
travels with the request. The failure modes are OAuth failure modes with sharper
consequences, because the caller is autonomous:

- a token **not bound to the resource** it is presented to can be replayed or
  passed through to a different server (confused‑deputy);
- an **on‑behalf‑of** token with no verifiable delegation chain makes "who
  authorised this?" unanswerable;
- an **omnibus scope** (`admin:*`) hands an injected or misbehaving agent the keys
  to everything;
- a **non‑expiring** agent credential never stops being useful to an attacker.

These are not hypothetical: they map onto the MCP authorization spec, OAuth token
exchange, and the OWASP LLM & Agentic Top 10.

## 2. Method

`keyway agent inspect` (`internal/agentauth`, `POST /v1/agent/inspect`) decodes a
token and evaluates it against a policy, emitting cited findings:

| Threat | Check | Normative source |
|---|---|---|
| `MCP-02` | token carries **no audience** — unbound to any resource | RFC 8707 / RFC 9728 |
| `MCP-01` | audience **does not include** the presented resource | RFC 8707 |
| `DEL-01` | on‑behalf‑of token **missing `act`** — delegation unverifiable | RFC 8693 |
| `SCOPE-01` | **over‑broad scope** (`admin:*`, `*`) — least privilege violated | OWASP Agentic |
| `SCOPE-02` | **no `exp`** / lifetime exceeds policy — non‑expiring credential | OAuth BCP |

The analysis is **static and stateless** — one token in, findings out — so it runs
anywhere, including client‑side in the browser (the marketing site's live
inspector runs exactly this check set with nothing leaving the page) and in CI.

## 3. Coverage, measured honestly

Agent auth is a young frontier, and we state its immaturity rather than hide it.
Reproduced with `keyway threats coverage`:

| Domain | Covered | Total | % |
|---|---|---|---|
| **agent** | 6 | 15 | **40%** |

The 9 uncovered threats are named and cited — e.g. `AGT-01` prompt‑injection‑driven
privilege escalation, agent‑identity spoofing, consent phishing. Treat agent
findings as high‑value but **non‑exhaustive**: a clean result means the checked
invariants hold, not that the token is safe against every documented threat.

## 4. Threats to validity

- Static single‑token analysis cannot see server‑side enforcement — a token may
  be well‑formed yet still be over‑trusted by a permissive MCP server.
- The policy (required audience, max lifetime, allowed scopes) is supplied by the
  operator; defaults are conservative but not universal.
- 40% coverage of a fast‑moving spec surface will shift as the taxonomy grows;
  the percentage is a snapshot, reproducible at any commit.

## 5. Reproduce

```bash
keyway agent inspect --token "$JWT" --audience https://mcp.example/api
keyway threats coverage        # the agent-domain denominator
```

Prev: **[Note 03 — Adversarial Verification](03-adversarial-verification.md)** ·
Series index: **[../whitepaper.md](../whitepaper.md)**.
