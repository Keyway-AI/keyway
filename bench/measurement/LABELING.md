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

# 3. a HUMAN fills the `human_label` column in a spreadsheet (taxonomy below)

# 4. compute the gold-standard numbers
python3 bench/measurement/grade_labels.py bench/measurement/out/labeling-packet.csv
```

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
