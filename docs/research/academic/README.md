# Keyway → academic publication: the plan

**Goal (stated):** academic credibility and reputation first; developer adoption
is expected to follow. This directory is the program to get there — honestly.

> These are **plans, not papers.** They contain no results. Keyway's current
> write‑ups (`../whitepaper.md`, `../01`–`04`) are strong *engineering*
> documentation; they are **not** yet publishable research, for one reason below.

## The one thing that must change

Every accuracy number Keyway ships today comes from a **corpus we authored
ourselves.** In academic review that is near‑disqualifying regardless of how
honest the caveats are: 100% on your own benchmark proves consistency, not a fact
about the world. The entire plan is built around fixing that — turning the tool
from *the claim* into *the instrument* that measures something real.

## What "credibility" actually takes

1. **A real contribution** — new technique, empirical finding, or systematization.
   A tool + a self‑made benchmark is infrastructure, not a contribution.
2. **Peer review at a recognized venue** — that is the credential. arXiv alone
   confers almost none.
3. **Author & reproducibility credibility** — who is on the paper, and can others
   rerun it.

## The two papers

| | **[Paper A — Measurement study](paper-a-measurement-study.md)** | **[Paper B — Agent‑auth SoK + measurement](paper-b-agent-auth-sok.md)** |
|---|---|---|
| Core | What auth weaknesses exist in real public config, at scale, and how they drift | Systematize + measure authorization threats for autonomous/MCP agents |
| Risk / reward | Lower risk, mechanical, defensible | Higher risk, higher upside — the field is nearly empty |
| Novelty source | Population‑scale *contract + drift* view (attacks themselves are known) | Early, rigorous framing of an under‑studied space |
| Best near‑term venues | USENIX Security (measurement‑friendly), NDSS, S&P | AIDC, SaTML, AISec workshops → main‑track SoK |

They **compose**: A's corpus/pipeline is B's measurement substrate; B's taxonomy
sharpens A's agent RQs.

**Recommendation.** If you want the safest first *real* paper, do **A** and build a
track record. If you want maximum reputational leverage and can invest in doing an
SoK properly, do **B** — being "the paper that framed agent‑auth" is a durable
citation. Many teams do A→B.

## Credibility multipliers (use them)

- **Bring in an academic co‑author.** A systems‑security professor or senior PhD
  student changes both the odds and the perceived credibility, and shores up the
  parts this work is weakest on (related‑work positioning, evaluation rigor). This
  is the **single highest‑leverage move** for the stated goal.
- **Chase Artifact Evaluation badges.** USENIX/CCS/S&P/NDSS award
  *Available / Functional / Reproduced* badges. Keyway's `make bench` /
  `make mutation` reproducibility is tailor‑made for this — honest, bankable
  credibility exactly where most tool papers are weak.
- **Lead with the real‑world finding, never the synthetic 100%.** Keep the
  self‑authored benchmark as a validation appendix.

## Where & when to submit

See **[venues‑and‑cfps.md](venues-and-cfps.md)** — researched 2026‑08‑08, with the
open vs closed status of each venue and a recommended submission sequence. Short
version, given it's mid‑2026: arXiv preprint + a tool demo (Black Hat Europe
Arsenal / OWASP) now → a near‑term AI‑security workshop for Paper B (AIDC Sep 25 /
SaTML Sep 29) → the measurement study aimed at NDSS/S&P (late 2026) or USENIX
Security '27 Cycle 2 (Jan 2027). **Confirm every deadline on the official page —
they move.**

## Honest bottom line

The tool is real and the honesty is a genuine differentiator, but academic
credibility is a high bar that the current artifacts do **not** yet clear. The gap
is a real evaluation and a sharp contribution — both are a few months of focused
work (less with a co‑author). This directory is the map; the next step is choosing
A or B and building the study.
