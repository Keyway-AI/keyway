# AIDC 2026, submission notes (Paper B family)

**Venue:** Workshop on Agentic AI in Offensive and Defensive Cyber Operations (AIDC),
co-located with ACSAC, Dec 7 2026, Los Angeles. CFP: <https://aidcworkshop.github.io/>

Two separate things live here:

## 1) Paper track, `aidc26/main.tex` (compiles to `main.pdf`)
- **Deadline:** 2026-10-02 (AoE), *extended from Sep 25*.
- **Format (confirmed Oct 2026):** `\documentclass[conference,compsoc]{IEEEtran}`,
  US Letter, **double-blind**. Short position/WIP paper limit: **6 pp excl. refs**.
  Our draft is **3 pp total** (well under). Submit at <https://aidc.submit.acsac.org/>.
- **What it is:** a *distinct* short paper, "Attack Your Own Agent's Authorization:
  Testing MCP and OAuth Agent Tokens Before Deployment." Thesis: sort the 15
  agent-token threats by whether a defender can reproduce the attack offline (6 yes /
  9 no), and release a paired generator + analyzer for the 6. Offense/defense framing
  for AIDC's theme (areas 1, 2, 5).
- **Decision (author):** submit the AIDC paper track **now**, and **withdraw it if the
  overlapping work is accepted at SSR** (notification Oct 5). The AIDC paper is written
  as a different contribution (offense/defense + tool, not a systematization) to reduce
  overlap, but it shares the taxonomy, so the withdraw-on-SSR-accept plan is what keeps
  it clean. Note AIDC's paper deadline (Oct 2) is *before* SSR's notification (Oct 5),
  so the two overlap in review for ~3 days by design.
- **Anonymized:** no author names or repo links in the PDF (the one name is a LaTeX
  comment, which does not render). Repository link withheld for review.

## 2) Lightning talk, `aidc26/LIGHTNING-TALK.md`
- **Deadline:** 2026-10-08 (AoE). Non-archival 5-min talk, title + 200-word abstract.
- **No conflict with SSR**, it is a talk, not a paper. The clean, low-risk way to put
  the agent-auth work in front of the AIDC audience regardless of the SSR outcome.
- Abstract is written (188 words) and ready to paste into the form.

## Status, BOTH SUBMITTED (2026-10-01/02)
1. ☑ **Paper** submitted to AIDC paper track as **#43**, marked *ready for review*
   (`aidc.submit.acsac.org`, HotCRP). Editable/withdrawable until Oct 2 11:59:59 UTC.
   Short position paper; authors Archit Sharma + Garima Mann (Nometria Inc). Topics:
   red-teaming frameworks, formal verification/specifications, policy frameworks.
   Needed an ORCID iD on the submitting author's profile to mark ready (added).
2. ☑ **Lightning talk** submitted via the Google Form ("response recorded"). Non-archival.
   Answered "not a published work."

## Remaining action
- **Oct 5, SSR notification.** If the overlapping SoK is accepted at SSR, **withdraw #43**
  (Withdraw button on the paper's edit page). The lightning talk stays regardless.

## Build
```
cd docs/research/submissions/paper-b/aidc26 && tectonic main.tex   # -> main.pdf
```
