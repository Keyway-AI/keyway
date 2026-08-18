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

> Whether a service accepts a bearer token is decided by configuration, not by
> code: the issuers it trusts, the audiences it binds to, the algorithms it allows,
> and the claims it requires. That configuration is an implicit authorization
> contract. It is rarely written down, and when it drifts without anyone noticing,
> services either turn valid users away or start accepting tokens they should
> refuse. We think this contract is worth measuring directly, and this paper is a
> first step toward that. We give a method that derives the contract from deployment
> configuration, and we run it over 428 public repositories to see how healthy these
> contracts are in practice. We report how common a set of concrete, RFC-defined
> problems are (audiences bound to nothing, algorithms left unpinned, required
> claims missing), with Wilson confidence intervals. A measurement is only as good
> as the tool behind it, so we check the extractor two ways: a kind-aware oracle for
> recall (issuers 89.6%, audiences 99.0%), and a negative control for precision that
> plants values a correct tool must ignore. We are explicit about what each check
> does and does not show. We release the tool, the corpus manifest, and the
> analysis, and set out the agenda a full study needs: scale, drift over time, and
> where configuration stops telling you what runtime does.

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
