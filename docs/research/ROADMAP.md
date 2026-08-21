# Post-submission roadmap

Both SSR 2026 papers are submitted (2026-08-19, EasyChair):

- **Submission 20** — *SoK: Authorization for Autonomous Agents* (SoK track)
- **Submission 21** — *Vision: Measuring JWT Authorization Contracts in the Wild* (Vision track)

Next dates: notification **2026-10-05**, camera-ready **2026-10-15**, conference
**Dec 13–15**. Submissions are editable on EasyChair until **2026-09-15**.

This file tracks what remains **from our side**: evaluation/research to turn the
preliminary results into citable ones, engineering features that support that, and
manual/ops work. The two research gates (**G1** hand-labelling, **G2** corpus
scaling) are the critical path: every full-research venue (USENIX, IMC, NDSS, S&P)
is blocked on them. See [`submissions/TRACKER.md`](submissions/TRACKER.md) for
venue deadlines and [`../../bench/measurement/FINDINGS.md`](../../bench/measurement/FINDINGS.md)
for the methodology gaps this addresses.

Legend: ☐ todo · ◑ in progress · ☑ done · ⧗ blocked (needs a human/token)

---

## A. Evaluation / research

| # | Task | Owner | Blocked on | Status |
|---|---|---|---|---|
| A1 | **G1 — hand-label the validation sample** (~85 rows; packet ready via `make_labeling_packet.py`). Gold-standard precision/recall. Decide single- vs two-annotator (κ). | Human | — | ⧗ |
| A2 | **G2 — scale corpus 428 → 10³–10⁴ repos** (crawl + re-run); add OIDC/MCP sources; report selection bias. | Me (drive) | fresh GH token | ⧗ |
| A3 | Re-run prevalence + validation on the scaled, labelled corpus. | Me | A1, A2 | ☐ |
| A4 | Co-occurrence analysis of weaknesses (e.g. unbound-audience ∧ no-claims). Shipped the `cooccur` package: pairwise joint prevalence, conditionals P(B\|A)/P(A\|B), and lift, printed and in `summary.json`. | Me | — | ☑ |
| A5 | Quantify the static-vs-runtime frontier (RQ4): what fraction is decidable from config vs needs live probing. Shipped: each check tagged `static_decidable`, and a rollup (`static_runtime_frontier` in summary.json + printed) reporting how many check types and raised flags are config-conclusive vs carry the library-default caveat (P3). | Me | — | ☑ |
| A6 | Drift-in-the-wild: longitudinal analysis over real commit history (classifier already validated). | Me | — | ☐ |
| A7 | **Paper B empirical**: build an MCP-server / agent-token corpus and measure prevalence of the 6 statically-checkable weaknesses. | Me | maybe token | ☐ |

## B. Features / engineering

| # | Task | Owner | Status |
|---|---|---|---|
| B1 | **Render Helm/kustomize before discovery** — ~1 in 5 corpus configs are under-resolved templates (17% Helm, ~2% kustomize). Biggest recall lever. Shipped as the `render` package + `--resolve-templates` (neutralizes standalone templated files today; `helm template`/`kustomize build` runs on whole trees once G2 fetches them) + `out/templating.json` coverage report. | Me | ☑ |
| B2 | Near-duplicate / fork-aware dedup (current dedup is exact-signature only). Shipped the `dedup` package: canonical signature (folds issuer host/slash, algorithm case, order) + a reported Jaccard near-duplicate diagnostic. Fork-of-upstream collapse still needs GitHub fork metadata from the G2 crawl. | Me | ☑ |
| B3 | Broaden discovery sources (more gateways / OIDC / MCP) for G2 representativeness. | Me | ☐ |
| B4 | Package the agent-auth analyzer as a public, documented CLI/Action (Paper B "we release an open-source analyzer"). | Me | ☐ |
| B5 | One-command reproducible artifact bundle with pinned deps (for the DOI + Artifact Evaluation). | Me | ☐ |

## C. Manual / ops / calendar

| # | Task | Owner | When | Status |
|---|---|---|---|---|
| C1 | **Rotate the GitHub PAT** pasted earlier in chat (exposed in transcript). | Human | now | ⧗ |
| C2 | Confirm EasyChair confirmation emails arrived (both papers). | Human | now | ☐ |
| C3 | Overleaf visual proof of both PDFs; optionally swap a polished PDF before the deadline. | Human | before 2026-09-15 | ☐ |
| C4 | If accepted (notify 2026-10-05) → camera-ready by 2026-10-15: de-anonymize the author block (uncomment the real author/institute lines in the SSR wrappers). | Me | Oct | ☐ |
| C5 | DOI-pinned Zenodo snapshot of instrument + harness + corpus manifest. | Human upload; I prep | camera-ready | ☐ |
| C6 | Artifact-Evaluation "Available/Reproduced" badges; register ≥1 author. | Human | if accepted | ☐ |
| C7 | Extended full Paper A → **USENIX Sec '27 Cy2 (2027-01-26)** or IMC, once G1+G2 are done. | Me (draft) | after A1–A3 | ☐ |

---

**Critical path:** A1 (G1) + A2 (G2). Everything that makes the papers
full-research grade hangs off those. The B-tasks improve the numbers regardless of
the gates and need no token, so they run in parallel.
