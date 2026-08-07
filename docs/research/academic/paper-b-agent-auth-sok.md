# Paper B — Authorization for Autonomous Agents: Systematization + Measurement

*Design document. Status: proposed. A plan, not results. The threat taxonomy it
builds on is real (`internal/threats`, 15 agent‑domain threats, all cited); every
empirical claim below is to be measured, not assumed.*

## 1. Why this paper (the upside case)

AI agents now carry OAuth/MCP bearer tokens and act autonomously on a user's
behalf. The authorization layer around them is **new, high‑stakes, and barely
studied** — which is exactly why a rigorous, early treatment carries outsized
reputational leverage. Keyway already ships a cited 15‑threat agent taxonomy and a
static analyzer; that is the seed of a **Systematization of Knowledge (SoK)** paper
strengthened by a **measurement** of real MCP servers and agent tokens.

This is higher‑risk than Paper A (SoK is hard to do well; the field is moving fast)
but higher‑reward (the space is open, and "the paper that framed agent‑auth
threats" is a durable citation).

## 2. Contribution statement

> We systematize the authorization threat model for autonomous‑agent and MCP
> deployments, mapping each threat to its normative source (OAuth RFCs 8693/8707,
> RFC 9728, the MCP authorization spec, OWASP LLM & Agentic Top 10) and to a
> concrete, checkable invariant. We then measure the prevalence of these weaknesses
> across public MCP servers and issued agent tokens, and release a static analyzer
> and the taxonomy as artifacts.

Contributions: (i) a **structured, cited threat model** for agent authorization
with checkable invariants (not a listicle); (ii) a **measurement** grounding the
taxonomy in reality; (iii) **released artifacts** (analyzer + taxonomy).

## 3. Structure (SoK + measurement hybrid)
1. **Threat model & systematization.** Organize the agent‑auth threat space by
   category (audience binding, delegation, scope, expiry, identity, agency). For
   each: definition, normative source, the invariant that must hold, and the
   failure it prevents. This is a rigorous version of `docs/threat-coverage.md`.
2. **Gap analysis.** What existing OAuth/OIDC machinery does and does not cover for
   autonomous callers (e.g., confused‑deputy via token passthrough; delegation
   chains that OAuth token exchange models but MCP deployments rarely verify).
3. **Measurement.** Prevalence of unbound audiences, missing `act`, omnibus scopes,
   non‑expiring credentials across a corpus of public MCP servers / sample agent
   tokens / reference deployments.
4. **What's detectable statically vs needs runtime.** Honest capability line.

## 4. Research questions
- **RQ1.** What is a complete, source‑grounded threat model for agent/MCP
  authorization, and where does it exceed the classic OAuth threat model?
- **RQ2.** Which threats are enforced by the MCP spec / common SDKs by default, and
  which are left to the deployer (and thus likely missed)?
- **RQ3.** Empirically, how prevalent are the checkable weaknesses across public
  MCP servers and agent‑token samples?
- **RQ4.** Which are decidable from a single token / server manifest (static) vs
  require interaction?

## 5. Data & methodology
- **Systematization corpus:** the specs themselves + published incidents/advisories
  + prior agent‑security literature; coding of threats by two authors with
  inter‑rater agreement reported.
- **Measurement corpus:** public MCP server registries and repos; sample tokens
  from reference/self‑hosted deployments (**never** exfiltrated real user tokens —
  ethics §7). Analyzer = `internal/agentauth`, run statically.
- **Validation:** manual review of a labelled sample for analyzer precision/recall
  on real manifests.

## 6. Related work (must‑cite)
- OAuth 2.0 threat model & security BCP; token exchange (8693), resource
  indicators (8707), protected‑resource metadata (9728).
- MCP authorization specification and its evolution.
- OWASP LLM & Agentic Top 10; published prompt‑injection → tool‑abuse chains.
- Confused‑deputy and delegation literature (classic + recent).
- Any prior agent‑security SoKs (likely few — establish the gap explicitly).

## 7. Threats to validity / ethics
- **Fast‑moving target:** the MCP spec changes; date‑stamp the systematization and
  frame coverage as a snapshot (as the tool already does).
- **Measurement bias:** public MCP servers skew toward hobbyist/early‑adopter;
  report it.
- **Ethics:** no probing of third‑party live agents; responsible disclosure for
  identifiable high‑severity findings; no real user tokens.
- **Construct:** static single‑token analysis can't see server‑side enforcement —
  stated as a limitation, measured in RQ4.

## 8. Target venues (see `venues-and-cfps.md`)
- **Near‑term, on‑topic, open:** **AIDC** (agentic‑AI focus) and **SaTML** (secure
  & trustworthy ML) — strong fits for an early workshop/short version.
- **AI‑security workshops:** **AISec** (with CCS) for the next cycle.
- **Main‑track SoK:** USENIX Security / IEEE S&P run SoK tracks — realistic only
  once the systematization is thorough and the measurement is real.

## 9. Effort & timeline (honest)
Systematization (rigorous, two‑coder) ~3–4 weeks; measurement pipeline + labelling
~3–5 weeks; writing ~3 weeks. A workshop‑length version is achievable faster than
Paper A; a main‑track SoK is comparable effort. The measurement is what separates
"a nice taxonomy" from "a paper," so do not skip it.

## 10. How A and B compose
Paper A's corpus/pipeline supplies Paper B's measurement substrate; Paper B's
taxonomy sharpens Paper A's RQ3. Sequencing options are in the academic README —
short version: A first to build a track record, or B first to plant a flag in an
empty field.
