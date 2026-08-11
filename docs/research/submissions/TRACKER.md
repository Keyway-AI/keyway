# Submission tracker

Living checklist for every target venue: format/style, deadline, and the pending
tasks that stand between the current drafts and a real submission. Update the
**Status** and check boxes as work lands. Dates researched 2026-08-08 — re-confirm
on each CFP.

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
- ☐ Ethics / Open Science / Artifact-availability sections (both papers).
- ☐ Figures: prevalence bar with CIs; reuse `docs/benchmark-roc.svg`.
- ☐ Prep the human-labelling packet for G1 (spreadsheet + snippets).
- ☐ Deeper Vision reframing of Paper A's intro for SSR (lead with agenda).
