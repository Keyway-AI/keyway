# SSR 2026 submission — Paper B (SoK)

**Venue:** Security Standardisation Research (SSR) 2026, Baltimore, Dec 13–15.
**Deadline:** 2026-09-15 (AoE). **Track:** Systematization of Knowledge (SoK).
**Format:** Springer LNCS (`llncs`), ≤23 pp incl. references, double-blind.
**Build:** `paper-b/ssr26/main.tex` → Overleaf (Springer LNCS template).
**CFP confirmed (2026-08-19, ssresearch26.umbc.edu/call-for-papers):**
- **Review:** double-blind. Exact rule: *"author names and affiliations must be
  omitted, and references to the authors' own prior work should be written in the
  third person."* Author block anonymized as "Anonymous Author(s)". This paper cites
  no prior work of ours, so no self-references to rewrite.
- **Track:** SoK. **Title must start with `SoK:`** — already does
  ("SoK: Authorization for Autonomous Agents").
- **Length:** ≤23 pp Springer LNCS incl. references, appendices excluded.
- **Submission server:** EasyChair — https://easychair.org/conferences/?conf=ssr2026
- **Originality:** *"original, unpublished, and not submitted to journals or other
  conferences/workshops that have proceedings."* The measurement section is stated
  as future work, so no unpublished-elsewhere conflict.
- **Dates:** submit 2026-09-15 · notify 2026-10-05 · camera-ready 2026-10-15 ·
  conference Dec 13–15. One author must register. PC chair Keke Chen
  (ssr2026pc@gmail.com).

## Title
SoK: Authorization for Autonomous Agents

## Abstract (portal-ready plaintext — paste into the submission form)

> When an AI agent calls a tool or an MCP server on your behalf, it carries a bearer
> token (OAuth or MCP) on the request, and the ways this goes wrong are OAuth
> failures with sharper consequences: a token not bound to the server it is sent to
> can be replayed elsewhere, an over-broad scope hands an injected agent far more
> than it needs, and a token with no verifiable delegation chain makes "who
> authorised this?" unanswerable. The practitioner lists that catalogue these risks
> do not agree on what must hold or on what a tool can actually check. We organize
> the agent-authorization threat model for the Model Context Protocol (MCP) and
> agent deployments into six categories (audience binding, delegation, scope, agent
> identity, confused-deputy, and agency), and for each of 15 threats we give its
> normative source (OAuth RFCs 8693/8707/9728/9700, the MCP authorization
> specification, the OWASP LLM and Agentic Top 10s, and the relevant CWEs) and the
> invariant it breaks. Our main result is where the line falls between what a tool
> can decide on its own and what it cannot: 6 of the 15 threats can be checked from a
> single token or server manifest, and we release an open-source analyzer for them,
> while the other 9 need runtime consent, a human in the loop, or a workload-identity
> layer the ecosystem is still building. Knowing which side of that line a threat
> sits on tells you what to automate and what remains an open problem.

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
