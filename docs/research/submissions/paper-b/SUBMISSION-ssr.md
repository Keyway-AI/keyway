# SSR 2026 submission — Paper B (SoK)

**Venue:** Security Standardisation Research (SSR) 2026, Baltimore, Dec 13–15.
**Deadline:** 2026-09-15 (AoE). **Track:** Systematization of Knowledge (SoK).
**Format:** Springer LNCS (`llncs`), ≤23 pp incl. references, double-blind.
**Build:** `paper-b/ssr26/main.tex` → Overleaf (Springer LNCS template).
**Confirm on the CFP before submitting:** SoK track exists and its exact rules;
double-blind policy; page limit; whether a work-in-progress/empirical-pending SoK
is in scope (state it in the paper as we do).

## Title
SoK: Authorization for Autonomous Agents

## Abstract (portal-ready plaintext — paste into the submission form)

> AI agents increasingly act for a user while carrying OAuth or MCP bearer tokens,
> but the authorization around them is new and not well understood. The practitioner
> lists that exist name risks without agreeing on what has to hold or on what a tool
> can actually check. We organize the agent-authorization threat model for Model
> Context Protocol (MCP) and agent deployments into six categories: audience
> binding, delegation, scope, agent identity, confused-deputy, and agency. For each
> of 15 threats we give its normative source (OAuth RFCs 8693/8707/9728/9700, the
> MCP authorization specification, the OWASP LLM and Agentic Top 10s, and the
> relevant CWEs) and the invariant it breaks. Our main point is where the line falls
> between what a tool can decide statically and what it cannot. Six of the 15 threats
> can be checked from a single token or server manifest, and we give an open-source
> analyzer for those; the other nine need runtime consent, a human in the loop, or a
> workload-identity layer the ecosystem is still building. Knowing which side of
> that line a threat sits on tells you what to automate and what is still an open
> problem.

## Readiness: ~85%. Checklist before submit

- [ ] Open `paper-b/ssr26/main.tex` in Overleaf (Springer LNCS template); compile clean.
- [ ] Proofread the body end-to-end (human pass); confirm Table 1 (the 15-threat
      taxonomy) renders and every `\cite` resolves.
- [ ] Confirm total length ≤ 23 pp (we are well under; room to expand per-category
      prose if desired).
- [ ] Keep it **anonymous** (no author names/affiliations, no self-identifying
      "our tool Keyway" — use "an open-source analyzer"; scrub repo URLs that
      deanonymize). Camera-ready authors: Archit Sharma, Garima Mann.
- [ ] Verify the `refs.bib` URLs/years resolve.
- [ ] Optional strengthening (not required for SoK): expand the gap analysis with
      1–2 worked examples; add a short "what a standard could mandate" paragraph
      (SSR reviewers value standardisation implications).
- [ ] Register the paper / abstract by the deadline; upload the PDF.

## Why this fits SSR
The venue is about how security standards are researched and implemented; this SoK
maps agent-auth threats directly onto OAuth/MCP standards and their gaps — squarely
on theme. The empirical measurement is explicitly future work (stated in the
paper); SoK does not require it.
