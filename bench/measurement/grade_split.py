#!/usr/bin/env python3
"""Held-out evaluation of discovery accuracy (train/test, ML-style).

Discovery's extraction/attribution rules were developed by inspecting failure
cases; to report an unbiased accuracy — and, in particular, how much the broader
attribution rules OVER-ATTRIBUTE — we split the labelled contested packet by
repository into a 70% train and a 30% held-out test split (repo-level, so no row
from a repo leaks across splits) and report metrics on each separately. The test
split is the headline: it is measured on repos whose configs did not inform the
rules.

  recall       = TP / (TP + discovery-miss)
  precision    = TP / (TP + spurious + wrong-attribution)      TP = correct + correct-extra
  over-attrib. = (spurious + wrong-attribution) / captured-only   (of discovery's own additions)

  python3 bench/measurement/grade_split.py bench/measurement/out/labeling-packet-ai.csv [--seed S] [--label-col ai_label]
"""
import argparse
import collections
import csv
import hashlib

VALID = {"correct", "correct-extra", "discovery-miss", "parser-artifact",
         "no-consumer", "spurious", "wrong-attribution"}


def split_of(repo, seed, test_frac=0.30):
    h = int(hashlib.sha256(f"{seed}|{repo}".encode()).hexdigest()[:12], 16)
    return "test" if (h % 1000) / 1000.0 < test_frac else "train"


def metrics(rows, label_col):
    c = collections.Counter((r.get(label_col) or "").strip().lower() for r in rows)
    tp = c["correct"] + c["correct-extra"]
    fn = c["discovery-miss"]
    fp = c["spurious"] + c["wrong-attribution"]
    recall = tp / (tp + fn) if (tp + fn) else float("nan")
    precision = tp / (tp + fp) if (tp + fp) else float("nan")
    caponly = [r for r in rows if r["side"] == "captured-only"]
    over = sum(1 for r in caponly if (r.get(label_col) or "").strip().lower()
               in ("spurious", "wrong-attribution"))
    overrate = over / len(caponly) if caponly else float("nan")
    return dict(n=len(rows), tp=tp, fn=fn, fp=fp, recall=recall, precision=precision,
                caponly=len(caponly), over=over, overrate=overrate, counts=c)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("csv", nargs="?", default="bench/measurement/out/labeling-packet-ai.csv")
    ap.add_argument("--seed", default="keyway-g1-2026")
    ap.add_argument("--label-col", default="ai_label", help="ai_label (v1) or human_label (v2)")
    args = ap.parse_args()

    rows = [r for r in csv.DictReader(open(args.csv))
            if (r.get(args.label_col) or "").strip().lower() in VALID]
    for r in rows:
        r["_split"] = split_of(r["repo"], args.seed)

    repos = {r["repo"]: r["_split"] for r in rows}
    n_tr = sum(1 for s in repos.values() if s == "train")
    print(f"labelled rows: {len(rows)} | repos: {len(repos)} "
          f"(train {n_tr} / test {len(repos) - n_tr}) | col={args.label_col} seed={args.seed}\n")

    for split in ("train", "test", "all"):
        sub = rows if split == "all" else [r for r in rows if r["_split"] == split]
        m = metrics(sub, args.label_col)
        tag = "  [held-out]" if split == "test" else ""
        print(f"== {split.upper()} =={tag}  (n={m['n']})")
        print(f"   recall       = {m['recall']:.1%}  (TP={m['tp']}, miss={m['fn']})")
        print(f"   precision    = {m['precision']:.1%}  (TP={m['tp']}, FP={m['fp']})")
        print(f"   over-attrib. = {m['overrate']:.1%}  ({m['over']} of {m['caponly']} captured-only "
              f"are spurious/wrong-attribution)")
        print()


if __name__ == "__main__":
    main()
