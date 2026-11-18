# KubeCon + CloudNativeCon Europe 2027 — co-located talk proposals (DRAFT)

**Event:** co-located day Mon 15 March 2027, Barcelona (KubeCon EU 2027 is 15–18 Mar).
**CFP portal:** https://sessionize.com/cncf-hosted-co-located-events-europe-2027/
**CFP closes:** Sun 18 Oct 2026, 23:59 CET · **Notifications:** 14 Dec 2026.
**⚠️ Submit by Oct 15** — Archit is on the trek (no laptop) Oct 16–25.

These are non-archival **talks**, so they do not conflict with the SACMAT / WAITI / PAM
paper submissions — the same work can be presented. Speakers are named (unlike the
anonymized paper track); the open-source tool (Keyway) may be named.

Sessionize submission needs, per proposal: **Title**, **Session abstract** (public,
keep ~400–600 chars), **optional notes to reviewers**, **session format/length**, and
a **speaker bio**. You can usually tag a submission to one or more of the co-located
events — tag each to the event named below.

---

## Shared speaker bio *(personalize / verify before submitting)*

> Archit Sharma builds Keyway at Nometria, an open-source tool that derives and checks
> the authorization contract a service enforces on JSON Web Tokens — which issuers it
> trusts, which audience it binds, which algorithms and claims it requires — across
> service meshes, proxies, and identity providers. His recent work measures how that
> contract is configured across thousands of public deployments and systematizes the
> authorization failures that follow autonomous agents into MCP. Co-presenter: Garima
> Mann (Nometria).

*(Add links/credentials you want public. If submitting solo to a given event, drop the
co-presenter line; Sessionize lets you add a second speaker where co-presenters are
allowed.)*

---

## 1 — Agentics Day: MCP + Agents  *(Paper B — agent-authorization SoK)*

**Title:** The Agent Is a Bearer Token: Authorization Failures Hiding in MCP

**Session abstract (public):**
> An autonomous agent is, to the services it calls, a bearer token with initiative.
> When an attacker steers that agent through untrusted input, the only thing standing
> between them and your APIs is how the token is scoped and validated — and today that
> layer is wide open. We walk the authorization failures that follow agents into MCP:
> token passthrough across resource boundaries, over-broad scopes a prompt injection
> can cash in, delegation chains with no on-behalf-of record, and the confused-deputy
> consent flows that grant them. Then we draw the line that matters in practice —
> which of these you can catch before the agent ever runs, from the token and the
> server's declared contract, and which need a runtime control. You leave knowing what
> to gate in CI and what still needs defense in depth.

**Notes to reviewers:**
> Based on a systematization of 15 agent-authorization threats across six categories
> (audience binding, delegation, scope, agent identity, confused deputy, agency),
> mapped to the RFCs and MCP clauses they violate. Six of the fifteen are
> contract-verifiable before deployment; nine are irreducibly runtime. We ship an
> open-source analyzer for the static subset. Practitioner-focused, tool-agnostic, no
> product pitch; grounded in a research SoK under submission (non-archival talk).

**Format:** 25–30 min session (or lightning if that's the only slot).

---

## 2 — KeycloakCon  *(Paper A — JWT validation measurement)*

**Title:** Your Tokens Are Fine. Your Validators Aren't.

**Session abstract (public):**
> Keycloak mints well-formed tokens — but whether a token is *accepted* is decided
> downstream, by each service's configuration: which issuers it trusts, which audience
> it binds, which algorithms it allows, which claims it requires. Those checks are
> scattered across meshes, proxies, and gateways and rarely written in one place. We
> derived this implicit authorization contract from 2,718 public repositories and
> measured it across 530 JWT-validating services. The results, with confidence
> intervals: none pin a signing algorithm in configuration, four in five require no
> claim beyond issuer and audience, and a majority bind no audience at all — so a token
> minted for one service is, as configured, accepted at another. We show what a
> Keycloak deployer can check today to close the gap between a good token and a sloppy
> validator.

**Notes to reviewers:**
> Measurement study of JWT validation configuration at population scale; recall
> validated against an independent parse, numbers conservative (examples removed,
> deduplicated). Directly actionable for Keycloak/OIDC deployers: audience binding,
> algorithm pinning, required claims. Tool and corpus manifest released open source.
> Non-archival talk; paper under submission.

**Format:** 25–30 min session.

---

## 3 — Open Source SecurityCon  *(bridges both papers)*

**Title:** Measuring the Authorization Layer: From JWT Config to Agent Tokens

**Session abstract (public):**
> Authorization in cloud-native systems lives in configuration, not code — and almost
> nobody measures it. We did. Across 2,718 public repositories and 530 JWT-validating
> services we found the authorization contract is routinely wide open: no algorithm
> pinned in config, most services requiring no claim beyond issuer and audience, and a
> majority binding no audience, which is token passthrough at population scale. The
> same layer now governs autonomous agents and MCP, on a faster-moving surface where
> an injected agent reaches whatever its token allows. We connect the measurement to a
> systematization of agent-authorization threats, and draw the line between what a
> pre-deployment check can guarantee and what only a live runtime can — with an
> open-source tool for the part you can check today.

**Notes to reviewers:**
> Combines a large-scale measurement of JWT authorization configuration with a
> systematization of agent/MCP authorization threats (static vs runtime enforcement).
> Open-source analyzer + released corpus manifest. Methodology validated (independent
> parse recall, conservative cleaning). Vendor-neutral, no product pitch. Non-archival
> talk; underlying papers under submission.

**Format:** 25–30 min session.

---

## Submission checklist (Sessionize)
- [ ] Create/sign in to Sessionize, open the CNCF co-located EU 2027 CFP.
- [ ] Submit proposal 1 → tag **Agentics Day: MCP + Agents**.
- [ ] Submit proposal 2 → tag **KeycloakCon**.
- [ ] Submit proposal 3 → tag **Open Source SecurityCon**.
- [ ] Add speaker bio + headshot + any links; add Garima as co-speaker where wanted.
- [ ] Confirm each session format matches what the event offers.
- [ ] Done before Oct 16 departure.
