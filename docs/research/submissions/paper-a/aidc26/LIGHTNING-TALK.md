# AIDC 2026, Lightning Talk submission (Paper A, measurement)

**Status: ☑ SUBMITTED 2026-10-04** via the Google Form ("Your response has been
recorded"). Presenter Archit Sharma, Nometria Inc; answered "not a published work."
Non-archival; notification 2026-10-16, final abstract due 2026-10-30.

**Track:** Lightning talk (5-minute in-person). Non-archival presentation slot.
**Deadline:** 2026-10-08 (AoE). **Notification:** 2026-10-16. **Final abstract:** 2026-10-30.
**Submit via:** the Lightning Talk form linked from <https://aidcworkshop.github.io/>
(Google Form: title + abstract, max 200 words). No bio or paper required.

Non-archival, so this does **not** conflict with Paper A (Vision) under review at SSR,
and it is a distinct talk from the Paper B (agent-auth) lightning talk already submitted.

**Fit note (honest, CFP checked 2026-10-04):** AIDC's theme is agentic cyber
operations; this is a JWT-config measurement study. It belongs only on its
agent-token framing: an agent carries an OAuth/MCP bearer token, and the
configuration we measure is exactly the layer that decides how far that token
reaches. The talk leads with that, not with web auth in general.
- The CFP states **no per-author limit** on lightning talks, so a second one
  alongside the Paper B talk is procedurally fine.
- Lightning talks are **non-archival** ("not published by IEEE") and the CFP
  explicitly welcomes "work-in-progress ideas and preliminary results," which is
  what this is.
- One caveat: the CFP prohibits "simultaneous submission of the same work to
  multiple venues." That targets archival paper dual-submission; a non-archival
  talk is generally outside it (the same basis the Paper B talk relied on), but the
  content overlaps with Paper A (Vision) under review at SSR, so treat it as a
  conscious, low-risk call rather than automatic.
- Thematic fit is weaker than Paper B; if the organizers want only one talk from
  us, keep Paper B.

---

## Title  *(revised 2026-10-06 for the expanded paper; small change from the first submission)*

**Where Agent Tokens Land: Measuring JWT Authorization Contracts at Scale**

## Abstract  *(paste into the form, 196 words)*

An autonomous agent calls tools and MCP servers with a bearer token, and whether that
token is accepted is decided by configuration, not code: which issuers a service
trusts, which audience it binds, which algorithm it allows, and which claims it
requires. These checks are spread across service meshes and proxies and rarely
written in one place, so when an attacker steers an agent through untrusted input,
they decide how far the agent's token reaches.

We derived this authorization contract from 2,718 public repositories and measured it
across 530 JWT-validating services. No service pins a signing algorithm in
configuration. Four in five require no claim beyond issuer and audience. A majority
bind no audience, so a token minted for one service is, as declared, accepted at
another, which is token passthrough at population scale. These figures are
conservative: removing examples and duplicates lowers them, and the finding holds
across every cleaning choice.

This talk walks the method, the numbers with confidence intervals, and the line
between what configuration reveals and what only a live runtime can. Takeaway: the
layer that governs an agent's authority is measurable today, and most of it is wide
open.

---

## 5-minute outline (speaker notes, not submitted)

- **0:00, Hook (45s).** An agent is a bearer token with autonomy. What decides how far
  that token reaches is not the model and not the code; it is configuration.
- **0:45, The contract (1m).** Issuers, audiences, algorithms, claims, JWKS behaviour,
  spread across meshes, proxies, and IdPs. Rarely centralised. We call it the implicit
  authorization contract and derive it from deployment config with per-field provenance.
- **1:45, Measuring it (1m).** 2,718 repos, 530 JWT-validating services, per repository,
  examples excluded and deduplicated. Recall validated against an independent parse.
- **2:45, The numbers (1m).** 100% no algorithm pinned in config; 80.9% no required
  claims; 53.4% unbound audience. Wilson CIs. The audience gap is token passthrough.
- **3:45, Agent stakes (45s).** The same contract now governs agent/MCP tokens, on a
  faster-moving surface where an injected agent reaches whatever the token allows.
- **4:30, Takeaway (30s).** This layer is measurable now; most of it is open. Tool and
  corpus manifest are released.

## Notes
- Lightning talks are not anonymized (unlike the paper track), so the slides may name
  the authors and the open-source tool.
- Keep it concept- and method-first; the CFP prohibits product pitches.
- The 53.4% / 80.9% figures are the scaled, validated numbers (see Paper A revision);
  the human gold-standard labelling is still pending, so present them as a measured
  baseline, not a final audit.
