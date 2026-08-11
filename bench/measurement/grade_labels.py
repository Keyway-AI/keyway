#!/usr/bin/env python3
"""Turn a filled labelling packet into gold-standard precision/recall (gate G1).

Reads the CSV produced by make_labeling_packet.py after a human has filled the
`human_label` column (taxonomy in LABELING.md), and computes:

  recall    = TP / (TP + discovery-miss)
  precision = TP / (TP + spurious + wrong-attribution)

excluding parser-artifact and no-consumer rows from the denominators (they are not
discovery's responsibility). Rows with a blank human_label are reported and skipped.

  python3 bench/measurement/grade_labels.py bench/measurement/out/labeling-packet.csv
"""
import collections
import csv
import sys

CSV = sys.argv[1] if len(sys.argv) > 1 else "bench/measurement/out/labeling-packet.csv"
VALID = {"correct", "discovery-miss", "parser-artifact", "no-consumer",
         "spurious", "wrong-attribution", "correct-extra"}


def main():
    counts = collections.Counter()
    per_field = collections.defaultdict(collections.Counter)
    unlabeled = bad = 0
    for row in csv.DictReader(open(CSV)):
        lab = (row.get("human_label") or "").strip().lower()
        if not lab:
            unlabeled += 1
            continue
        if lab not in VALID:
            bad += 1
            continue
        counts[lab] += 1
        per_field[row["field"]][lab] += 1

    tp = counts["correct"] + counts["correct-extra"]
    fn = counts["discovery-miss"]
    fp = counts["spurious"] + counts["wrong-attribution"]
    recall = tp / (tp + fn) if (tp + fn) else float("nan")
    precision = tp / (tp + fp) if (tp + fp) else float("nan")

    print(f"\nGold-standard (human-labelled) — {sum(counts.values())} labelled "
          f"({unlabeled} blank, {bad} invalid)\n")
    for k in sorted(counts):
        print(f"  {k:<18}{counts[k]}")
    print(f"\n  recall    = {recall:.1%}   (TP={tp}, discovery-miss={fn})")
    print(f"  precision = {precision:.1%}   (TP={tp}, FP={fp})")
    print("  excluded from denominators: parser-artifact "
          f"{counts['parser-artifact']}, no-consumer {counts['no-consumer']}")
    if unlabeled:
        print(f"\n  {unlabeled} rows still need a human_label.")


if __name__ == "__main__":
    main()
