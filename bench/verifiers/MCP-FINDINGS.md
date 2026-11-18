# MCP-server-class empirical eval (A7, Paper B)

> The agent-specific half of the verifier eval: fire the adversarial **agent-token**
> corpus at real **MCP-server** auth implementations and record which accept a token a
> correct MCP resource server must reject. The headline agent threat is **MCP-01/02**:
> a token minted for a *different* resource (wrong or absent `aud`) must be rejected,
> never passed through. This complements the 8-library + Envoy eval in
> [RESULTS.md](out/RESULTS.md) (proxy/middleware classes) with the MCP class.

Reproduce: `bench/verifiers/run-mcp-eval.sh` (needs `uv` + Python ≥3.10). Each target
is a real MCP auth component behind a thin 200/401 handler — the same isolation the
library targets use, so the auth decision is measured without the MCP JSON-RPC layer.
All controls were accepted (both targets trust the test issuer), so rejections are
meaningful.

## Targets (2 MCP-server-class verifiers)

| # | Target | What it is |
|---|---|---|
| 1 | `mcp-fastmcp` | FastMCP's `JWTVerifier` — the one MCP framework that *bundles* an OAuth resource-server verifier (JWKS signature + issuer + `aud`==resource). |
| 2 | `mcp-python-sdk` | The official MCP Python SDK's `BearerAuthBackend` in its **default** config (`resource_server_url` unset), with a competent bring-your-own verifier (RS256 sig + issuer + expiry via PyJWT; resource binding delegated to the framework as the MCP architecture intends). |

## Findings — both accept a token they should reject; the SDK default accepts passthrough

| Target | Accepted attack(s) | Meaning |
|---|---|---|
| `mcp-fastmcp` | CLM-08 missing `exp`; CLM-02 not-yet-valid (`nbf` ignored) | binds audience correctly, but accepts a **non-expiring** agent token (SCOPE-02) and ignores `nbf` |
| `mcp-python-sdk` | **MCP-01 passthrough (critical)**, MCP-02 unbound aud, CLM-04 wrong aud, DEL-02 sibling resource, CLM-08 missing `exp` | with resource binding off by default, a token minted **for another resource is accepted** — token passthrough, the SoK's #1 agent threat |

**FastMCP gets the headline right** — it rejects MCP-01 passthrough, MCP-02 unbound
audience, wrong audience, and the sibling-resource delegation token. Its gaps are
expiry (no `exp`) and `nbf`.

**The reference SDK's default does not bind audience at all.** Its `BearerAuthBackend`
enforces the RFC 8707 resource binding only when `resource_server_url` is set, which
**defaults to `None`**; with it unset, a wrong-/absent-`aud` token authenticates. It
also accepts a no-`exp` token (the middleware checks expiry only if the verifier
populates `expires_at`).

## Per-threat prevalence (of 2 MCP verifiers)

| Threat | Severity | Accepting | Note |
|---|---|---|---|
| CLM-08 missing `exp` (SCOPE-02 non-expiring) | medium | **2/2** | both accept a token with no expiry |
| MCP-01 token passthrough | critical | **1/2** | MCP Python SDK default |
| MCP-02 unbound audience | high | **1/2** | MCP Python SDK default |
| CLM-04 wrong audience | high | **1/2** | MCP Python SDK default |
| DEL-02 sibling-resource delegation | high | **1/2** | MCP Python SDK default |
| CLM-02 not-yet-valid (`nbf`) | medium | **1/2** | FastMCP |

## The architectural finding (source analysis, both SDKs)

The base MCP SDKs — Python (`mcp`) and TypeScript (`@modelcontextprotocol/sdk`) — ship
**no JWT verifier**: token verification is bring-your-own. So a deployed MCP server on
the base SDK inherits its security from two places, *both* of which default toward
accepting bad tokens:

1. **Its chosen JWT library's defaults** — exactly what the 8-library + Envoy eval
   measured: missing-`exp` accepted by 4/9, unbound-audience by 1/9 (python-jose).
2. **The SDK's resource binding, which is opt-in** (`resource_server_url` default
   `None`). The MCP spec's #1 requirement — reject a token not bound to this resource —
   is therefore off unless the operator explicitly turns it on.

**FastMCP is the exception**: it bundles the binding and enforces MCP-01/02 by default
(though it still accepts non-expiring tokens). So the agent-specific audience check
exists in the ecosystem's opinionated framework but not in the reference SDK's default
path — precisely the kind of default-footgun the SoK argues should be a tested,
enforced property rather than a specification obligation.

## Honest notes (for the paper and reviewers)

- These are **reference implementations stood up locally**, not scans of third-party
  servers. The MCP Python SDK target uses a standard, competent BYO verifier (signature
  + issuer + expiry) in the SDK's **default** config; the finding is about that default,
  not about a deliberately weakened verifier.
- `mcp-fastmcp` tests FastMCP's `JWTVerifier.verify_token`; `mcp-python-sdk` tests the
  SDK's real `BearerAuthBackend` via Starlette.
- Defaults/footguns, not exotic bugs — the same thesis as the library eval.
- Synthetic tokens only, localhost only (the probe engine is default-deny on hosts).
- Raw artifacts: `out/mcp-conformance.{txt,json}`. Reproduce: `run-mcp-eval.sh`.

## How this folds into SACMAT #35 (Paper B)

Together with the 8-library + Envoy result, this completes the three-class empirical
picture the reviewers asked for (proxy / middleware / **MCP server**). The measured
headline for the agent-auth SoK: *across the MCP-server class, the reference SDK's
default configuration accepts token passthrough (MCP-01), and every MCP verifier
measured accepts a non-expiring agent token* — the audience-binding and non-expiry
weaknesses are the default, not the corner case.
