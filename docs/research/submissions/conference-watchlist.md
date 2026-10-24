# Conference CFP watch list — talks for Keyway (Paper A + Paper B)

Non-archival industry talks, so they don't conflict with the academic paper
submissions (SACMAT / WAITI / PAM). Good O-1 "speaking at distinguished venues"
evidence. Speakers named; Keyway may be named.

**Talk inventory** (full drafts in [PROPOSALS.md](kubecon-eu27-colocated/PROPOSALS.md)):
- **Talk B** — *The Agent Is a Bearer Token* (Paper B agent-auth SoK; MCP/agents)
- **Talk A** — *Your Tokens Are Fine. Your Validators Aren't.* (Paper A JWT measurement; Keycloak/IdP)
- **Talk X** — *Measuring the Authorization Layer* (bridge; cloud-native security)

---

## Submitted / in flight

All three submitted via Sessionize on 2026-10-14 (notifications 2026-12-14). Speaker:
Archit Sharma, Founder at Nometria (solo). Can be withdrawn/edited until Oct 18.

| Event | Deadline | Target | Talk | Title | Topic tag | Status |
|---|---|---|---|---|---|---|
| CNCF Co-located EU 2027 | Oct 18 2026 | Agentics Day: MCP + Agents | Talk B | The Agent Is a Bearer Token: Authorization Failures Hiding in MCP | Security + Trust + Reliability | ✅ submitted |
| CNCF Co-located EU 2027 | Oct 18 2026 | KeycloakCon | Talk A | Your Tokens Are Fine. Your Validators Aren't. | Keycloak Security Best Practices | ✅ submitted |
| CNCF Co-located EU 2027 | Oct 18 2026 | Open Source SecurityCon | Talk X | Measuring the Authorization Layer: From JWT Config to Agent Tokens | Supply Chain + Application Security | ✅ submitted |

**Profile note:** submitting updated the Sessionize speaker profile globally — tagline
is now "Founder at Nometria" and the bio is the Keyway/authorization one. This also
applies to the 3 pre-existing talks on this CFP (agent tool permissions drift, LLM
guardrails fail-open, detector selection). The **Company Website** on the profile
still points to credwise.co.in (left as-is; no Nometria URL on hand to replace it).

---

## Pending results & deadlines (verified 2026-10-20). All AoE = UTC-12.

**Trek:** Archit off-grid ~Oct 16–25 (limited phone access).

| Date | Item | Type |
|---|---|---|
| Oct 19 (AoE ≈ Oct 20 17:30 IST) | WAITI #20 (Paper B SoK) acceptance notification | result |
| Oct 23 | AIDC paper #43 notification | result |
| Oct 25 | AIDC lightning talks (A + B) notification | result |
| **Oct 27 (AoE)** | **WAITI #20 camera-ready** (if accepted) | deadline |
| Oct 30 | AIDC #43 final paper (if accepted) | deadline |
| Oct 31, 07:59 EDT | SACMAT SoK cycle-1 submission (fallback) | deadline |
| Oct 31 | SACMAT #13 (Paper A) edit window closes | deadline |
| Nov 2 | AIDC final lightning abstract (if accepted) | deadline |

- WAITI dates per https://waiti-workshop.github.io/dates/ — site states all deadlines
  are 11:59 PM AoE (UTC-12). Submission was Oct 12 (done); camera-ready Oct 27.
- AIDC dates per https://aidcworkshop.github.io/ — notifications pushed from Oct 16 to
  Oct 23 (paper) / Oct 25 (lightning); finals Oct 30 / Nov 2. Workshop Dec 7, LA (ACSAC).
- SACMAT SoK fallback (paper-b/sacmat27) is submission-ready; submit by Oct 31 if WAITI
  + AIDC reject. See [[ssr-reject-resubmit]].

---

## Watch list — CFP not open yet (re-check on the "check" date)

| Event | Dates / Location | Expected CFP window | Check on | Best talk | Angle to tailor |
|---|---|---|---|---|---|
| **AGNTCon + MCPCon NA 2027** | Apr 28–29, San Francisco | Nov 2026–Jan 2027 | **2026-12-01** | Talk B | Sharpen to MCP-specific: passthrough, `act`-chain, scope minimization, confused-deputy consent. Lead with the static/runtime line. |
| **Open Source Summit NA 2027** | May 17–19, Vancouver | Nov 2026– | **2026-12-15** | Talk X | Broaden to OSS supply-chain + measurement framing; tool + corpus release. |
| **AGNTCon + MCPCon EU 2027** | Sep 15–16, London | mid-2027 | **2027-04-01** | Talk B | Same as AGNTCon NA; EU travel pairs with other EU events. |
| **KubeCon + CloudNativeCon NA 2027** | Nov 8–11, New Orleans | ~spring 2027 | **2027-05-01** | Talk A or X | Biggest venue. Mesh/Envoy + SPIFFE/SPIRE + Keycloak angle; main CFP, not co-located. |
| **Open Source Summit EU 2027** | Sep 20–22, Glasgow | mid-2027 | **2027-04-15** | Talk X | Broad OSS + security. |
| Confidential Computing Summit 2027 | Apr 19–20, SF | TBA | 2027-01-15 | Talk X (identity angle) | Workload-identity/attestation adjacency; weaker fit. |

**Tailoring note:** all watch-list talks reuse the three existing abstracts; only the
title emphasis and the 1–2 framing sentences change per venue. No new content needed —
retarget at submission time.

## Notes
- AGNTCon + MCPCon is a dedicated MCP/agents conference — the tightest home for Talk B.
  Its live `/cfp` page still shows the closed 2026 edition; watch for the 2027 refresh.
- KubeCon NA 2027 (New Orleans) is the best all-round fit and the biggest stage, but its
  CFP won't open until ~spring 2027.
- Maintainer Summit EU 2027 (Dec 1) is eligibility-gated (CNCF maintainer/TAG role) —
  skip unless that applies.
