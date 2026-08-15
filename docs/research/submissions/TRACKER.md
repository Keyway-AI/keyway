# Submission tracker

Living checklist for every target venue: format/style, deadline, and the pending
tasks that stand between the current drafts and a real submission. Update the
**Status** and check boxes as work lands. Dates researched 2026-08-08 — re-confirm
on each CFP.

**As of ~2026-08-13, NO deadline below has passed.** Nearest upcoming, in order:
NDSS'27 summer **Aug 19**, USENIX Sec'27 Cy1 **Aug 25**, SECITC **Sep 7**,
SSR **Sep 15**, OWASP Portugal **Sep 23**, AIDC **Sep 25**, SaTML **Sep 29**,
S&P'27 **Nov 17**, NDSS'27 fall / IMC'27 Cy1 **~Nov**, USENIX Sec'27 Cy2 **Jan 26 2027**.

Legend: ☐ todo · ◑ in progress · ☑ done · ⧗ blocked on a research gate (human
labelling / corpus scaling).

---

## At a glance

| # | Venue | Paper | Type | Deadline | Template | Limit | Blind | Status |
|---|---|---|---|---|---|---|---|---|
| 1 | **SSR 2026** | B (SoK) | SoK | **2026-09-15** | Springer `llncs` | 23 pp incl refs | double | ◑ drafting |
| 2 | **SSR 2026** | A (Vision) | Vision | **2026-09-15** | Springer `llncs` | 23 pp incl refs | double | ◑ drafting |
| 3 | **AIDC 2026** | B | workshop | **2026-09-25** | ACM `acmart` sigconf | (confirm) | double | ☐ |
| 4 | **SaTML 2027** | B | SoK/position | **2026-09-29** | `IEEEtran` conf | (confirm) | double | ☐ |
| 5 | **SECITC 2026** | A | full research | **2026-09-07** | Springer `llncs` | 12 pp excl refs | double | ⧗ needs gates |
| 6 | **IMC 2027** | A | full research | ~Nov 2026 / ~Apr 2027 | ACM `acmart` sigconf | 13 pp + unlimited refs | double | ⧗ needs gates |
| 7 | **USENIX Sec '27** | A | full research | **2027-01-26** (Cy2) | USENIX `usenix-2020-09` | 13 pp excl refs/appx | double | ⧗ needs gates |
| 8 | **NDSS 2027** | A | full research | fall ~Nov 2026 | NDSS class | 18 pp total | double | ⧗ needs gates |
| 9 | **IEEE S&P 2027** | A | full research | **2026-11-17** | `IEEEtran` conf | ~13 pp + refs | double + Ethics§ | ⧗ needs gates |
| — | Black Hat EU Arsenal | tool | demo | confirm | — | — | — | ☐ optional |
| — | OWASP AppSec Portugal | tool | talk | **2026-09-23** | — | — | — | ☐ optional |

## Readiness (as of ~2026-08-13)

Readiness = how close the relevant draft is to submittable **for that venue's
track**, given its bar and the two research gates (G1 hand-labelling, G2 corpus
scaling). "Deadline feasible?" judges whether we can realistically close the gap in
time.

| Venue (deadline) | Track bar | Readiness | Gaps to submittable | Deadline feasible? |
|---|---|---|---|---|
| **SSR 2026** (Sep 15) — Paper B SoK | SoK: no measurement required | **~85%** | proofread, Overleaf compile, length trim, confirm track | **Yes** — the realistic target |
| **SSR 2026** (Sep 15) — Paper A Vision | Vision: forward-looking OK | **~80%** | reframed intro done; proofread + compile + trim; confirm Vision track exists in final CFP | **Yes** |
| **AIDC 2026** (Sep 25) — Paper B | workshop / early work | **~80%** | confirm CFP (limit/blind); reuse SoK body | **Yes** |
| **SaTML 2027** (Sep 29) — Paper B | SoK/position | **~80%** | confirm limits; reuse SoK body | **Yes** |
| **NDSS 2027 summer** (Aug 19) — Paper A | full research | **~35%** | needs G1 **and** G2 | **No** (~1 wk) → aim fall |
| **USENIX Sec '27 Cy1** (Aug 25) — Paper A | full research | **~35%** | needs G1 **and** G2 | **No** (~2 wks) → aim Cy2 |
| **SECITC 2026** (Sep 7) — Paper A | full research | **~35%** | needs G1 + G2; trim to 12 pp | **No** (too tight) |
| **IEEE S&P 2027** (Nov 17) — Paper A | full research, top-tier | **~40%** | G1 + G2 + heavier writing/relatedwork; a co-author strongly helps | Stretch |
| **NDSS 2027 fall** (~Nov) — Paper A | full research, top-tier | **~40%** | G1 + G2 + polish | Stretch |
| **IMC 2027 Cy1** (~Nov) — Paper A | full research, measurement-core | **~40%** | G1 + G2 + polish (best topical fit) | Stretch |
| **USENIX Sec '27 Cy2** (Jan 26 2027) — Paper A | full research, top-tier | **~45%** | G1 + G2 + polish; **most runway** | **Yes, if G1/G2 done** |
| **IMC 2027 Cy2** (~Apr 2027) — Paper A | full research | **~45%** | G1 + G2 + polish; most runway | **Yes, if G1/G2 done** |

**Bottom line.** Two clusters. (a) *Ready-ish now* — the SoK/Vision/workshop tracks
(SSR, AIDC, SaTML) at ~80–85%: only human polish + Overleaf, no gates. (b) *Blocked
on gates* — every full-research Paper A venue at ~35–45%: needs **G1** (an hour of
human labelling) and **G2** (a scaled corpus, needs a token + background crawl). The
readiness number for those cannot rise past ~45% no matter how much drafting is
done, because the missing piece is *results*, not prose. Aim the near-deadline
full-paper venues (Aug 19 / Aug 25 / Sep 7) at their *later* cycles, and target
**USENIX Sec '27 Cy2 (Jan 26)** or **IMC 2027** as the top-venue Paper-A home once
G1/G2 close.

**Fastest honest path:** SSR 2026 (Sep 15) — rows 1 & 2. SoK (B) and Vision (A)
tracks accept our current honest state without fabrication. The `⧗` rows are
full-paper venues blocked on the two research gates below.

---

## Shared research gates (block all `⧗` full-paper rows)

- **G1 — hand-labelled validation** ⧗ (human). Label a stratified sample from
  `bench/measurement/out/labeling-worksheet.jsonl` (kind-aware pre-classification +
  `nonauth_declared` already emitted) to get the gold-standard precision/recall;
  adjudicate the residual claims cases. *I can prep the packet; a human (ideally a
  co-author) is the annotator.*
- **G2 — scale the corpus** ⧗ (needs a fresh GH token + background crawl). From the
  428-repo preliminary crawl toward $10^3$–$10^4$ services; add OIDC/MCP sources;
  strengthen near-duplicate/fork handling. Then re-run prevalence + validation.

Vision (SSR row 2) and SoK (SSR row 1) do **not** require G1/G2 to be a legitimate
submission; the full-paper rows do.

---

## Per-submission detail + pending tasks

### 1) SSR 2026 — Paper B, "SoK: Authorization for Autonomous Agents"
- **Format:** Springer LNCS (`\documentclass[runningheads]{llncs}`), ≤23 pp incl.
  references (appendices excluded), double-blind, `splncs04` bib style.
  Wrapper: `paper-b/satml27`… → use an LNCS wrapper; see task below.
- **Pending:**
  - ☐ Add an LNCS wrapper `paper-b/ssr26/main.tex` (llncs).
  - ◑ Expand the SoK body: full 15-threat taxonomy table (data in
    `docs/threat-coverage.md`) with category / invariant / normative source.
  - ◑ Gap analysis: OAuth/OIDC coverage vs autonomous-caller gaps.
  - ☐ `refs.bib` (RFCs 8693/8707/9728/9700/7591, MCP spec, OWASP LLM/Agentic,
    CWEs, WIMSE/SPIFFE, prior agent-security work).
  - ☐ Related work + positioning (first systematic agent-authz treatment).
  - ☐ Ethics + Open Science (artifact = analyzer + taxonomy).
  - ☐ Trim/verify ≤23 pp; confirm SSR blind policy.

### 2) SSR 2026 — Paper A (Vision), "Authorization Contracts in the Wild"
- **Format:** as SSR above (LNCS, ≤23 pp, double-blind).
- **Pending:**
  - ◑ Reframe `paper-a/common/body.tex` for the **Vision** track: lead with the
    problem + preliminary evidence as motivation, then a concrete research agenda;
    keep the honest preliminary numbers as motivating, not headline, results.
  - ☐ Related work (JWT/OIDC attacks; IaC scanners; Pact; measurement studies).
  - ☐ `refs.bib`.
  - ☐ A prevalence figure + the discovery-validation table.
  - ☐ Confirm the Vision track exists in the final CFP; trim to ≤23 pp.

### 3) AIDC 2026 — Paper B (workshop)
- **Format:** ACM `acmart` (`sigconf`), confirm page limit + blind on the CFP.
- **Pending:** same content as SSR-B; wrapper exists (`paper-b/aidc26`). ☐ Confirm
  CFP specifics; ☐ shares refs.bib + taxonomy table.

### 4) SaTML 2027 — Paper B (SoK/position)
- **Format:** IEEE `IEEEtran` (conference), confirm page limit; welcomes SoK +
  position. Wrapper exists (`paper-b/satml27`).
- **Pending:** same content as SSR-B; ☐ confirm limits; ☐ shares refs.bib.

### 5) SECITC 2026 — Paper A (full research) ⧗
- **Format:** Springer LNCS, ≤12 pp excl. bib/appendices, double-blind, full
  research paper (no work published/submitted elsewhere). Wrapper exists.
- **Pending:** ⧗ G1 + G2 (needs the finished study); then trim to 12 pp LNCS.
  Note the no-concurrent-submission rule vs SSR/IMC.

### 6–9) IMC 2027 / USENIX Sec '27 / NDSS 2027 / IEEE S&P 2027 — Paper A ⧗
- Wrappers exist. Blocked on G1 + G2. Per-venue deltas: page limit + template only
  (content is shared `paper-a/common/body.tex`). S&P also needs an **Ethics
  Considerations** section (camera-ready, not counted). Confirm each page limit.

---

## Global to-do (do-able now, no token / no human labelling)

- ☑ `paper-a/common/refs.bib` and `paper-b/common/refs.bib` (real citations; every
  `\cite` resolves; bibliography enabled in all wrappers).
- ☑ Paper B SoK: full 15-threat taxonomy table (Table~1) + static-vs-runtime gap
  analysis + related work + SSR/SaTML/AIDC wrappers.
- ☑ Paper A related work (JWT attacks / IaC scanners / Pact / measurement) +
  a Vision research-agenda section.
- ☑ Ethics + Open Science + Artifact-availability sections (both papers).
- ☑ Prevalence figure — pgfplots bar chart with Wilson CIs
  (`paper-a/common/fig-prevalence.tex`, `\input` in Results; pgfplots added to the
  6 Paper A wrappers).
- ☑ G1 human-labelling packet: `bench/measurement/make_labeling_packet.py`
  (worksheet → focused CSV, 85 rows on the current corpus) +
  `grade_labels.py` (labels → gold-standard precision/recall) + `LABELING.md`
  (taxonomy + workflow). **Ready for a human (co-author) to run.**
- ☐ Deeper Vision reframing of Paper A's intro for SSR (lead with agenda).
- ☐ Reuse `docs/benchmark-roc.svg` as a second figure if going the drift angle.

## G1 is now unblockable by a human
`make_labeling_packet.py` turns the validation disagreements into an ~1-hour
spreadsheet task; `grade_labels.py` computes the final numbers. Running it (ideally
with a co-author as a second annotator for inter-rater agreement) closes G1 and
unblocks every `⧗` full-paper venue.
