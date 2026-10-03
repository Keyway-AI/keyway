#!/usr/bin/env python3
"""Grade discovery under BOTH label sets and report their agreement (gate G1).

Reads a packet that carries an `ai_label` column (from ai_label.py) and a
`human_label` column (filled by a human). Computes, for each label set
independently:

  recall    = TP / (TP + discovery-miss)
  precision = TP / (TP + spurious + wrong-attribution)      TP = correct + correct-extra

excluding parser-artifact and no-consumer from the denominators (not discovery's
fault), and reports AI<->human agreement over the rows a human has labelled:
raw agreement and Cohen's kappa. The pair of numbers is the paper's sharper
result: a reproducible AI-only measurement plus a human-validated ground truth,
with a measured kappa saying how far the automated labeller can be trusted.

  python3 bench/measurement/grade_dual.py bench/measurement/out/labeling-packet-ai.csv
"""
import collections
import csv
import sys

VALID = {"correct", "correct-extra", "discovery-miss", "parser-artifact",
         "no-consumer", "spurious", "wrong-attribution"}


def pr(counts):
    tp = counts["correct"] + counts["correct-extra"]
    fn = counts["discovery-miss"]
    fp = counts["spurious"] + counts["wrong-attribution"]
    recall = tp / (tp + fn) if (tp + fn) else float("nan")
    precision = tp / (tp + fp) if (tp + fp) else float("nan")
    return tp, fn, fp, recall, precision


def report(name, labels):
    counts = collections.Counter(labels)
    tp, fn, fp, recall, precision = pr(counts)
    print(f"\n== {name} == ({len(labels)} labelled)")
    for k in sorted(counts):
        print(f"  {k:<18}{counts[k]}")
    print(f"  recall    = {recall:.1%}  (TP={tp}, discovery-miss={fn})")
    print(f"  precision = {precision:.1%}  (TP={tp}, FP={fp})")
    print(f"  excluded: parser-artifact {counts['parser-artifact']}, no-consumer {counts['no-consumer']}")
    return recall, precision


def cohen_kappa(pairs):
    """Cohen's kappa over (a, b) label pairs. Stdlib only."""
    n = len(pairs)
    if not n:
        return float("nan"), float("nan")
    agree = sum(1 for a, b in pairs if a == b)
    po = agree / n
    cats = {c for pair in pairs for c in pair}
    a_marg = collections.Counter(a for a, _ in pairs)
    b_marg = collections.Counter(b for _, b in pairs)
    pe = sum((a_marg[c] / n) * (b_marg[c] / n) for c in cats)
    kappa = (po - pe) / (1 - pe) if (1 - pe) else float("nan")
    return po, kappa


def main():
    path = sys.argv[1] if len(sys.argv) > 1 else "bench/measurement/out/labeling-packet-ai.csv"
    rows = list(csv.DictReader(open(path)))

    ai = [(r.get("ai_label") or "").strip().lower() for r in rows]
    hu = [(r.get("human_label") or "").strip().lower() for r in rows]
    bl = [(r.get("human_label_blind") or "").strip().lower() for r in rows]
    ai_valid = [x for x in ai if x in VALID]
    hu_valid = [x for x in hu if x in VALID]
    pending = sum(1 for x in ai if x in ("review", "error"))

    print(f"rows: {len(rows)}  |  AI-labelled: {len(ai_valid)}  "
          f"(+{pending} review/error)  |  human-labelled: {len(hu_valid)}")

    r_ai, p_ai = report("v1  AI-only", ai_valid)
    if hu_valid:
        r_hu, p_hu = report("v2  human-validated", hu_valid)
        pairs = [(a, h) for a, h in zip(ai, hu) if a in VALID and h in VALID]
        po, kappa = cohen_kappa(pairs)
        print(f"\n== AI <-> human agreement == ({len(pairs)} rows both labelled)")
        print(f"  raw agreement = {po:.1%}")
        print(f"  Cohen's kappa = {kappa:.3f}")
        print(f"\n  discovery recall    v1(AI) {r_ai:.1%}  ->  v2(human) {r_hu:.1%}")
        print(f"  discovery precision v1(AI) {p_ai:.1%}  ->  v2(human) {p_hu:.1%}")
    else:
        print("\n(no human_label values yet — fill them to get v2 + the AI/human kappa)")

    # Anchoring-caveat number: agreement on the BLIND subset the human labelled
    # before seeing the AI draft. Un-anchored, so it is the honest kappa to report.
    blind_pairs = [(a, b) for a, b in zip(ai, bl) if a in VALID and b in VALID]
    if blind_pairs:
        po_b, kappa_b = cohen_kappa(blind_pairs)
        print(f"\n== AI <-> human agreement, BLIND subset == ({len(blind_pairs)} rows, un-anchored)")
        print(f"  raw agreement = {po_b:.1%}")
        print(f"  Cohen's kappa = {kappa_b:.3f}  (report THIS as the honest agreement)")


if __name__ == "__main__":
    main()
