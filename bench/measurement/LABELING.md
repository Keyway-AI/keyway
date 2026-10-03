# Human-labelling packet (research gate G1)

The gold-standard validation the paper needs. The automated validator
(`validate.py`) is a proxy; this is the ~hour of human adjudication that produces
the definitive discovery **precision/recall** on real data — the number that
replaces the self-authored benchmark as the accuracy claim.

## Workflow

```bash
# 1. produce the worksheet (per-repo discovery + kind-aware validation)
go run ./bench/measurement --path bench/measurement/corpus --per-repo
python3 bench/measurement/validate.py bench/measurement/corpus bench/measurement/out

# 2. build the labelling packet (focuses on the contested cases + a calibration sample)
python3 bench/measurement/make_labeling_packet.py \
    bench/measurement/out/labeling-worksheet.jsonl \
    bench/measurement/out/labeling-packet.csv

# 3. AI first pass: an independent LLM drafts a label per row from the RAW config
#    (never from Keyway's output). Produces the AI-only label set (v1).
export ANTHROPIC_API_KEY=...        # or `ant auth login`
python3 bench/measurement/ai_label.py \
    bench/measurement/out/labeling-packet.csv \
    bench/measurement/out/labeling-packet-ai.csv

# 4. build the human-ready workbook (dropdowns, a BLIND-first subset, the AI draft
#    parked on the right as a reference)
python3 bench/measurement/make_g1_xlsx.py \
    bench/measurement/out/labeling-packet-ai.csv \
    bench/measurement/out/g1-packet.xlsx

# 5. a HUMAN fills it: first `human_label_blind` on the BLIND rows (without looking
#    at the AI columns), then `human_label` on every row using the AI draft as a
#    reference. Save the Label sheet back to CSV (e.g. labeling-packet-human.csv).

# 6. grade BOTH sets and report their agreement (Cohen's kappa), incl. the honest
#    un-anchored kappa from the blind subset
python3 bench/measurement/grade_dual.py bench/measurement/out/labeling-packet-human.csv
```

(The older single-pass flow — skip step 3, have the human fill `human_label` on
`labeling-packet.csv`, then `grade_labels.py` — still works.)

## Two label sets: AI-only (v1) and human-validated (v2)

Hand-labelling is the bottleneck for gate G1, and it does not scale to the G2
corpus (10^3-10^4 rows). So we label twice and report both:

- **v1 — AI-only.** `ai_label.py` sends each contested `(repo, field, value)` to an
  LLM together with the **raw source config** and the taxonomy below, and records
  `ai_label` + `ai_confidence` + `ai_rationale`. The labeller sees only the raw YAML
  and which side found the value — **never Keyway's discovery output or its
  reasoning** — so using these labels to grade Keyway's discovery is *not* circular,
  exactly as the independent-parse proxy is not.
- **v2 — human-validated.** A human fills `human_label`, using the AI draft and its
  rationale as a starting point, and correcting it. This is the gold standard.

`grade_dual.py` reports discovery precision/recall under each set and the
**AI<->human agreement (raw + Cohen's kappa)**. The pair is the sharper result: a
reproducible, scalable AI measurement, plus a human ground truth, plus a measured
kappa that says how far the automated labeller can be trusted — which is what
licenses running the AI pass alone over the full G2 corpus (with a human-validated
subsample) once discovery is scaled.

**Anchoring caveat (report this).** Seeing the AI draft can bias the human toward
it, inflating kappa. For an honest agreement number, label a random subset
**blind** (ignore / hide the `ai_label` column) and compute kappa on that subset;
report it alongside the full-sample number. Two human annotators on that subset
(a co-author) strengthens it further.

## What each row is

One row per `(repo, field, value)` where the independent parse and Keyway's
discovery **disagree** (plus ~15% agreement rows as calibration). Columns:
`side` (declared-only / captured-only / agree), `in_nonauth_context` (the parser
scraped it from a ConfigMap/annotation), `has_jwt_consumer`, and a
`suggested_label` you confirm or correct.

## Label taxonomy (fill `human_label` with exactly one)

| Label | Use when | Effect |
|---|---|---|
| `correct` | the value is real auth config **and** correctly captured/attributed | TP |
| `correct-extra` | captured-only, but the value **is** real (the parser missed it) | TP |
| `discovery-miss` | real auth value in a genuine auth resource that Keyway **failed** to capture | FN (hurts recall) |
| `parser-artifact` | the parser scraped a non-auth value (ConfigMap key, annotation, comment) — not real | excluded |
| `no-consumer` | real claim/value, but its RequestAuthentication isn't in the corpus (nothing to attach to) | excluded |
| `spurious` | Keyway captured a value that is **not** real auth config | FP (hurts precision) |
| `wrong-attribution` | Keyway captured a real value but attached it to the **wrong** consumer | FP (hurts precision) |

Then: `recall = TP / (TP + discovery-miss)`,
`precision = TP / (TP + spurious + wrong-attribution)`; `parser-artifact` and
`no-consumer` are excluded from both denominators (they are not discovery's fault).

## Rigor notes for the paper

- **Two annotators** on a subset with an inter-rater agreement (e.g. Cohen's κ)
  strengthens the claim; a co-author is the natural second labeller.
- Label from the **source file** (open the repo path from `sources.tsv`), not from
  the packet alone, when a row is ambiguous.
- Report the exact labelled N and the resulting precision/recall with CIs; this is
  what closes gate G1 and unblocks the full-paper venues.
