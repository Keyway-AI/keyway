# Agent auth: the verification gap

> A landscape + strategy brief on authentication and authorization for AI agents
> (MCP, agent-to-agent, on-behalf-of delegation, non-human identity), and where
> Keyway extends. This is the narrative behind the **agent** domain of the threat
> taxonomy ([`internal/threats/agent.go`](../internal/threats/agent.go)); coverage
> is tracked in [threat-coverage.md](./threat-coverage.md).
>
> _Compiled July 2026 from a multi-source research sweep. Draft-status standards
> (ID-JAG, on-behalf-of, WIMSE) are moving fast — revisit before committing
> roadmap._

## Thesis

The industry has largely solved **issuing** and **enforcing** auth for AI agents,
and is racing to **govern** non-human identities. Proving any of it is *actually
correct* — verifying and adversarially testing agent-auth contracts — is
genuinely unowned. That is exactly the model Keyway already runs on JWTs.

## 1. How agent auth works now

Every layer rests on the primitives Keyway already speaks: access tokens are
**JWTs** verified against issuer **JWKS**; **OIDC** carries the human into
delegation; audience binding decides who a token is *for*. The new work is a
delegation grammar on top, only partly standardized:

- **MCP** is the concrete standard shipping today, built on **OAuth 2.1**. The
  pivotal [March → June 2025 revision](https://forgecode.dev/blog/mcp-spec-updates/)
  re-cast the MCP server as strictly an OAuth **resource server** (not also the
  authorization server), structurally closing the confused-deputy / passthrough
  class. It mandates [RFC 9728 protected-resource metadata](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)
  for discovery and **RFC 8707 resource indicators** so a token's `aud` is bound
  to one server. Dynamic Client Registration is now deprecated in favor of
  Client-ID Metadata Documents.
- **Delegation** runs on [RFC 8693 Token Exchange](https://www.rfc-editor.org/info/rfc8693/)
  (a *published* standard): the `act` claim records a nestable, auditable
  delegation chain; `may_act` pre-authorizes delegates. `actor_token` +
  `subject_token` yields **delegation** (agent keeps its identity) rather than
  **impersonation**.
- **Emerging (draft):** the WG-adopted [Identity Assertion Authorization Grant](https://oauth.net/cross-app-access/)
  (ID-JAG / "Cross-App Access") makes an enterprise IdP the control plane for
  agent-to-app access; the individual [On-Behalf-Of for AI Agents](https://www.ietf.org/archive/id/draft-oauth-ai-agents-on-behalf-of-user-00.html)
  draft; [WIMSE](https://datatracker.ietf.org/doc/draft-ietf-wimse-workload-identity-practices/)
  workload identity; and Google **A2A** (Agent Card `securitySchemes`). The
  OpenID Foundation's [Identity Management for Agentic AI](https://openid.net/wp-content/uploads/2025/10/Identity-Management-for-Agentic-AI.pdf)
  whitepaper finds current standards only *partially* cover multi-hop delegation.

**Net:** primitives are mature; a purpose-built, interoperable *user → agent →
agent* delegation standard is still coalescing.

## 2. Threat surface

All the high-severity threats reduce to the same root Keyway already targets — an
**under-validated, over-scoped, or wrong-audience token**. Ranked:

| # | Threat | Severity | Source |
|---|---|---|---|
| 1 | Prompt-injection privilege escalation + excessive agency | Critical | [OWASP LLM01/06](https://owasp.org/www-project-top-10-for-large-language-model-applications/), [Agentic Top 10](https://genai.owasp.org/2025/12/09/owasp-genai-security-project-releases-top-10-risks-and-mitigations-for-agentic-ai-security/) |
| 2 | Confused deputy via MCP static client IDs | Critical | [MCP spec](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices) |
| 3 | OAuth token passthrough / theft / replay | Critical | MCP spec; [Salesloft-Drift breach](https://aembit.io/blog/real-life-examples-of-workload-identity-breaches-and-leaked-secrets-and-what-to-do-about-them-updated-regularly/) |
| 4 | Over-scoped / non-expiring agent credentials | High | [2026 NHI report](https://securityboulevard.com/2026/07/the-agent-identity-problem-non-human-identities-outnumber-humans-45-to-1-and-ai-agents-are-making-it-worse/) (45:1, ~69% static keys) |
| 5 | Tool poisoning / rug-pull + MCP CVEs | High | [CVE-2025-49596 (CVSS 9.4)](https://www.oligo.security/blog/critical-rce-vulnerability-in-anthropic-mcp-inspector-cve-2025-49596) |
| 6 | Session hijacking | Med–High | MCP spec |
| 7 | Delegation-chain / audience-scope confusion | Med–High | [RFC 8725](https://datatracker.ietf.org/doc/html/rfc8725), [RFC 9700](https://datatracker.ietf.org/doc/html/rfc9700) |

The bottom four are Keyway's existing domain — token validation, audience
binding, scope minimization, key rotation — re-pointed at agent tokens.

## 3. Vendors and the whitespace

The market splits into three layers, and is already consolidating (Cisco →
Astrix, CrowdStrike → SGNL):

- **Issue & delegate (CIAM):** [Auth0/Okta "Auth for GenAI"](https://auth0.com/blog/auth0-for-ai-agents-generally-available/)
  (Token Vault, async human-in-the-loop, FGA for RAG), [WorkOS AuthKit](https://workos.com/docs/authkit/mcp),
  [Descope Agentic Identity Hub 2.0](https://www.descope.com/press-release/agentic-identity-hub-2.0),
  Stytch Connected Apps, [Clerk M2M](https://clerk.com/changelog/2025-10-14-m2m-ga).
- **Enforce policy (FGA):** [OpenFGA](https://openfga.dev/docs/use-cases/ai-agent-authorization),
  Oso, Cerbos, Aserto, AWS Cedar / Verified Permissions, [SGNL](https://www.crowdstrike.com/en-us/press-releases/crowdstrike-to-acquire-sgnl-for-ai-era/).
- **Discover & govern (NHI security):** [Astrix](https://www.securityweek.com/cisco-moves-to-acquire-astrix-security-to-tackle-non-human-identity-risks/),
  Token Security, Aembit, Clutch, Oasis, Britive; clouds ([AWS AgentCore Identity](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/identity.html),
  Entra Agent ID); gateways (Solo.io, Lasso, Pomerium).

**The whitespace:** every vendor above issues credentials, enforces policy, or
discovers/governs identities. Almost none prove agent auth is *actually correct*:

- **No one tests the intersection** — agent policy ∩ token scope ∩ the human's
  real entitlement (the confused-deputy gap). ([SANS](https://www.sans.org/blog/your-ai-agent-easily-confused-deputy-why-cloud-security-needs-a-credential-broker/),
  [CSA](https://labs.cloudsecurityalliance.org/research/csa-research-note-ai-agent-confused-deputy-prompt-injection/))
- **JWT/JWKS correctness for agents is assumed, never probed** — alg confusion,
  audience/scope confusion, expiry, key rotation on issued/exchanged agent tokens.
- **No adversarial/generative harness** tests whether an agent's OAuth exchange,
  scope downgrade, or on-behalf-of flow behaves correctly *before* deployment.
- **MCP-specific end-to-end verification is absent** — does the issued token
  actually restrict to the declared tools?

## 4. Keyway's opening — extension roadmap

Each existing Keyway capability maps cleanly onto the unowned verification layer:

| Priority | Move | Maps to |
|---|---|---|
| **P1** ✅ | Agent-auth threat taxonomy (this domain) — MCP passthrough, confused-deputy, DCR abuse, missing/forged `act`, delegation-chain abuse, over-scope, non-expiring creds, session-as-auth, injection escalation, tool poisoning — all cited, all currently gaps. | threat taxonomy + coverage report |
| **P2** ✅ | Contract-verify delegated & MCP resource-server tokens — `internal/agentauth` statically checks `aud` binding (RFC 8707/9728), the delegation `act` claim (RFC 8693), scope minimization, and expiry. `keyway agent inspect`; covers MCP-01/02, DEL-01, SCOPE-01/02 (agent domain 0 → 33%). | contract discovery + diff |
| **P3** ✅ | Generative attack corpus for agent flows — live audience/resource-binding attacks (resource passthrough MCP-01, unbound audience MCP-02, sibling-hop DEL-02) fired at endpoints via `keyway probe --harness`; a correct resource server must reject each, accepting one is a live passthrough vuln (agent 33 → 40%). Flow-level attacks (confused-deputy `redirect_uri`, DCR) need a future OAuth-flow tester. | generative / invariant attack harness |
| **P4** | Delegation-chain blast radius: if an agent's key, scope, or issuer changes — or a delegate is revoked — which downstream hops break or over-trust a token they shouldn't accept? | blast-radius resolver |

**Positioning:** the open-source **correctness layer** for agent auth —
independent, standards-anchored, complementary to the issuers, enforcers, and
governors. As those layers consolidate into security platforms, an independent
verifier that proves the contract holds gets more valuable, not less.

_See [threat-coverage.md](./threat-coverage.md) for live coverage
(`keyway threats coverage --domain agent`)._
