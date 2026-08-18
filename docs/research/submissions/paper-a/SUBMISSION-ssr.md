# SSR 2026 submission — Paper A (Vision)

**Venue:** Security Standardisation Research (SSR) 2026, Baltimore, Dec 13–15.
**Deadline:** 2026-09-15 (AoE). **Track:** Vision paper.
**Format:** Springer LNCS (`llncs`), ≤23 pp incl. references, double-blind.
**Build:** `paper-a/ssr26/main.tex` → Overleaf (Springer LNCS template).
**Confirm on the CFP before submitting:** the **Vision** track exists and its rules
(some venues cap Vision at fewer pages); double-blind policy; page limit.

## Title
Authorization Contracts in the Wild: Measuring JWT and Agent-Auth Verification
Configuration at Scale

## Abstract (portal-ready plaintext — paste into the submission form)

> Authorization is the security property everyone assumes and no one measures.
> Whether a bearer token is accepted is decided not by code but by configuration —
> the issuers a verifier trusts, the audiences it binds to, the algorithms it
> allows, the claims it requires — an implicit authorization contract that is
> rarely written down and, when it drifts, causes outages or silent
> over-acceptance. We argue this contract should be a first-class, measurable
> object, and take a concrete first step: a method to derive it automatically from
> deployment configuration, and a preliminary measurement of its hygiene across 428
> public repositories. We quantify RFC-defined weaknesses — unbound audiences,
> absent algorithm pinning, missing required claims — with Wilson confidence
> intervals; and because a measurement is only as trustworthy as its instrument, we
> validate extraction with a kind-aware oracle for recall (issuers 89.6%, audiences
> 99.0%) and a deliberately non-circular negative control for precision, reporting
> what each can and cannot establish. We release the instrument, the corpus
> manifest, and the analysis, and set out the research agenda — scale, drift, and
> the static/runtime frontier — that a full study will complete.

## Readiness: ~80%. Checklist before submit

- [ ] Open `paper-a/ssr26/main.tex` in Overleaf (Springer LNCS + pgfplots); compile
      clean; confirm Fig. 1 (prevalence + Wilson CIs) renders.
- [ ] Proofread end-to-end (human pass); confirm every `\cite` resolves and the
      preliminary numbers match the repo (`bench/measurement/FINDINGS.md`).
- [ ] Keep the **preliminary/not-final framing** explicit (it already is): the
      numbers motivate; the agenda is the contribution. This is the whole point of
      the Vision track — do not overclaim.
- [ ] Anonymize: no author names; replace self-identifying tool/repo references
      with neutral phrasing ("an open-source instrument"). Camera-ready authors:
      Archit Sharma, Garima Mann.
- [ ] Confirm length ≤ the Vision page limit; trim §§ Method/Corpus if needed.
- [ ] Register + upload by the deadline.

## Honest note
This is the earliest track at which Paper A can be submitted **without** the two
research gates (G1 hand-labelling, G2 corpus scaling). If those close before
Sep 15, consider upgrading to SSR's regular research track instead of Vision;
otherwise Vision is the correct, honest home.
