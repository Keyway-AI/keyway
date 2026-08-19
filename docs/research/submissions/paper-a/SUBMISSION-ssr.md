# SSR 2026 submission — Paper A (Vision)

**Venue:** Security Standardisation Research (SSR) 2026, Baltimore, Dec 13–15.
**Deadline:** 2026-09-15 (AoE). **Track:** Vision paper.
**Format:** Springer LNCS (`llncs`), ≤23 pp incl. references, double-blind.
**Build:** `paper-a/ssr26/main.tex` → Overleaf (Springer LNCS template).
**CFP confirmed (2026-08-19, ssresearch26.umbc.edu/call-for-papers):**
- **Review:** double-blind. Exact rule: *"author names and affiliations must be
  omitted, and references to the authors' own prior work should be written in the
  third person."* Author block anonymized as "Anonymous Author(s)"; the one
  self-reference ("our companion systematization") was rephrased in the third person.
- **Track:** Vision (work-in-progress / preliminary results / position). **Title
  must start with `Vision:`** — done.
- **Length:** ≤23 pp Springer LNCS incl. references, appendices excluded
  ("reviewers are not required to read the appendices").
- **Submission server:** EasyChair — https://easychair.org/conferences/?conf=ssr2026
- **Originality:** *"original, unpublished, and not submitted to journals or other
  conferences/workshops that have proceedings."* SSR has LNCS proceedings, so
  publishing this Vision paper here makes the later full measurement paper (USENIX /
  IMC) a follow-on that must be substantially extended — plan accordingly.
- **Dates:** submit 2026-09-15 · notify 2026-10-05 · camera-ready 2026-10-15 ·
  conference Dec 13–15. One author must register. PC chair Keke Chen
  (ssr2026pc@gmail.com).

## Title
Vision: Measuring JWT Authorization Contracts in the Wild

(SSR requires Vision-track titles to start with `Vision:`. The full-research
wrappers for other venues keep the longer, un-prefixed title.)

## Abstract (portal-ready plaintext — paste into the submission form)

> Whether a service accepts a bearer token comes down to configuration, not code:
> which issuers it trusts, which audiences it binds to, which algorithms it allows,
> and which claims it requires. These settings are spread across service meshes,
> proxies, and identity providers, they are rarely written down in one place, and a
> single change (a rotated key, a widened audience, a dropped claim) silently turns
> valid users away or starts admitting tokens that should be refused. We call this
> configuration a service's implicit authorization contract, and we think it should
> be measured directly rather than assumed. We give a method that derives the
> contract from deployment configuration and run it over 428 public repositories. In
> a first sample of 102 JWT-validating configurations, none pinned a signing
> algorithm in configuration (100%, leaving it to the verifying library's default),
> 85.3% required no claims, and 42.2% left the token audience unbound, all with
> Wilson 95% confidence intervals. Because a measurement is only as good as its
> extractor, we validate it two ways and state each method's limits: a kind-aware
> oracle for recall (issuers 89.6%, audiences 99.0%), and a negative control for
> precision that plants values a correct tool must ignore. We release the tool, the
> corpus manifest, and the analysis. These numbers are preliminary and from a small
> sample; the paper sets out the agenda a full study needs: scale, drift over time,
> and the line between what configuration reveals and what only runtime can.

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
