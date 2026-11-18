# AIDC 2026, Lightning Talk submission

**Track:** Lightning talk (5-minute in-person). Non-archival presentation slot.
**Deadline:** 2026-10-08 (AoE). **Notification:** 2026-10-16. **Final abstract:** 2026-10-30.
**Submit via:** the Lightning Talk form linked from <https://aidcworkshop.github.io/>
(Google Form: title + abstract, max 200 words). No bio or paper required.

Lightning talks are presentation slots, not papers, so this does **not** conflict with
the Paper B SoK under review at SSR. It can safely reuse the same material.

---

## Title

**Attack Your Own Agent's Authorization Before an Attacker Does**

## Abstract  *(paste into the form, 188 words)*

An autonomous agent carries a bearer token on every tool or MCP call. If an attacker
can steer the agent through untrusted input, they inherit whatever that token allows.
The authorization layer is part of the attack surface, and most teams never test it.

This talk gives a practical way to reason about that surface. Of fifteen documented
agent-token threats, six can be reproduced offline from a single token or server
manifest: a wrong or missing audience, a broken on-behalf-of delegation chain, an
over-broad scope, and a non-expiring credential. The other nine appear only at
runtime, including confused-deputy consent flows, prompt-injection escalation, and
missing workload identity.

The six have a useful consequence. A defender can mint the exact tokens an attacker
would forge and confirm its own verifier rejects them, as a check that runs on every
build. In five minutes I will walk the loop end to end, from an adversarial token to
cited findings, and explain why the remaining nine mark where a runtime defense is
needed rather than a better scanner.

Takeaway: you can red-team the decidable part of agent authorization today, and know
exactly where the hard problems begin.

---

## 5-minute outline (speaker notes, not submitted)

- **0:00, Hook (45s).** One agent, one token. A poisoned document steers the agent;
  the attacker now has the token's authority. No memory bug required.
- **0:45, The surface (1m).** 15 threats, six categories. One slide: which six are
  decidable offline, which nine need a live runtime.
- **1:45, The six invariants (45s).** Audience binding, `act` delegation chain,
  least-privilege scope, expiry. Each lives in the token's own claims.
- **2:30, The loop, shown (1m15s).** An adversarial token (no `aud`, `admin:*`, no
  `exp`) and the analyzer's cited findings (MCP-02, SCOPE-01, SCOPE-02, DEL-01). Fail
  the build on any acceptance.
- **3:45, The nine (45s).** Confused deputy, prompt-injection escalation, workload
  identity. Why these are runtime-defense problems, not scanner gaps.
- **4:30, Takeaway (30s).** Test the decidable part on every build; know where the
  runtime agenda starts. Open-source generator + analyzer available.

## Notes
- Lightning talks are not anonymized (unlike the paper track), so the slides may name
  the authors and the open-source tool.
- Keep it concept- and method-first; the CFP prohibits product pitches.
