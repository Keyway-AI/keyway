# Submission package — venues, deadlines, formats, and drafts

**Researched 2026-08-08. Re-confirm every date and rule on the official CFP before
submitting — they move, and 2027 pages were still partial when this was compiled.**

> ## Readiness — read this first
> **Neither paper is submittable as of today.** The drafts here carry our *real,
> current* results and are formatted per venue, but each has explicit
> **`[PENDING]` gates** where a final number must go. Do **not** submit until:
> 1. **Paper A** — the **hand-labelled validation** is done (replaces the
>    independent-parse proxy as the accuracy claim) and the corpus is scaled beyond
>    the current 428-repo preliminary crawl. The prevalence numbers are currently
>    marked *preliminary / not for citation*.
> 2. **Paper B** — a real **measurement** of MCP/agent deployments exists (today it
>    is a design + taxonomy only). It is drafted as a **position / work-in-progress**
>    paper, which some venues accept, but the empirical section is a gate.
>
> Submitting before these gates close would be fabrication — exactly what we don't do.

## Paper A — measurement study (`paper-a/`)

Venue-formatted wrappers all `\input` one shared body (`paper-a/common/body.tex`),
so the content stays identical and only the template/length differs.

| Venue | Next deadline | Page limit (body) | Template | Blind | Fit |
|---|---|---|---|---|---|
| **IMC 2027** (Internet Measurement Conf.) | Cycle 1 ~**Nov 2026**, Cycle 2 ~**Apr 2027** (IMC'26 was 11/20 & 4/29) | 13 pp text+figs, refs/appendix unlimited | ACM `acmart` `sigconf` | double | **Best** — measurement is IMC's core |
| **USENIX Security '27** | Cycle 2 **2027-01-26** (abs 01-19) | 13 pp excl. refs/appendix | USENIX `usenix-2020-09` | double | Strong; measurement-friendly, most runway |
| **NDSS 2027** | Fall cycle ~**Nov 2026** | 18 pp **total** (incl. refs) | NDSS `ndss` (Times 10pt, 2-col, US-letter) | double | Strong |
| **IEEE S&P 2027** | **2026-11-17** (abs 11-10) | ~13 pp + refs (confirm) | `IEEEtran` (conference) | double + Ethics §| Stretch |

Build: open the venue's `main.tex` in Overleaf with that venue's official template
(links in each wrapper's header comment); the shared body compiles under all four.
Anonymize for submission — the wrappers default to an anonymous author block with
the real authors (Archit Sharma, Garima Mann) in a camera-ready comment.

## Paper B — agent-auth SoK/position (`paper-b/`)

| Venue | Deadline | Type accepted | Template | Fit |
|---|---|---|---|---|
| **SaTML 2027** | **2026-09-29** | research / **SoK** / **position** | IEEE `IEEEtran` | Strong; explicitly takes SoK + position |
| **AIDC 2026** (Agentic-AI, co-ACSAC) | **2026-09-25** | workshop / early work | ACM `acmart` | On-topic, near-term |
| **AISec** (w/ CCS) | next cycle (2026 closed 07-24) | workshop | ACM `acmart` | AI-security workshop |

`paper-b/` is an **extended abstract / work-in-progress** (no measurement yet),
honestly framed as early work; the empirical section is a `[PENDING]` gate.

## Non-academic (present the tool now — not the credibility goal)

| Venue | Deadline | Notes |
|---|---|---|
| Black Hat Europe Arsenal 2026 | confirm at europe-arsenal-cfp.blackhat.com | open-source tool demo, Dec 2026 |
| OWASP AppSec Days Portugal 2026 | 2026-09-23 | small/regional |

## The one preprint step to do regardless
Post the consolidated Paper A to **arXiv `cs.CR`** once the validation gate closes.
First `cs.CR` submission needs an endorsement or an academic-affiliated email — line
that up.

Sources: [IMC'26 CFP](https://conferences.sigcomm.org/imc/2026/cfp/) ·
[USENIX Sec '27 CFP](https://www.usenix.org/conference/usenixsecurity27/call-for-papers) ·
[NDSS'26 templates](https://www.ndss-symposium.org/ndss2026/submissions/templates/) ·
[S&P 2027 CFP](https://sp2027.ieee-security.org/cfpapers.html) ·
[SaTML 2027 CFP](https://satml.org/call-for-papers/) ·
[sec-deadlines](https://sec-deadlines.github.io/).
