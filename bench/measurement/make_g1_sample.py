#!/usr/bin/env python3
"""Draw a STRATIFIED RANDOM sample for the G1 gold standard (population CIs).

The contested packet (make_labeling_packet.py) labels only the disagreements, so
it cannot give population precision/recall. This instead draws a self-weighting
random sample across the whole captured/declared population, stratified by
(field x side), so a human's labels yield unbiased precision AND recall with
Wilson confidence intervals that scale with the corpus.

  - precision universe = captured values  (side 'agree' or 'captured-only')
  - recall universe    = declared values  (side 'agree' or 'declared-only')
  - 'agree' rows serve both.

A fixed fraction is taken from every stratum (same rate everywhere), so the sample
is self-weighting: raw precision/recall on it estimate the population directly, and
grade_dual.py needs no reweighting.

  python3 bench/measurement/make_g1_sample.py \
      bench/measurement/out-val/labeling-worksheet.jsonl \
      bench/measurement/out-val/labeling-packet-sample.csv [--target 400] [--seed ...]
"""
import argparse
import csv
import hashlib
import json
import math
import collections

FIELDS = ("issuers", "audiences", "claims")
COLS = ["repo", "field", "value", "side", "in_nonauth_context",
        "has_jwt_consumer", "suggested_label", "human_label", "notes"]


def suggest(side, in_nonauth, has_consumer):
    if side == "agree":
        return "correct"
    if side == "captured-only":
        return "review(likely correct-extra or spurious)"
    if in_nonauth:
        return "parser-artifact"
    return "discovery-miss" if has_consumer else "no-consumer"


def wilson_halfwidth(p, n, z=1.96):
    if n == 0:
        return float("nan")
    lo = (p + z * z / (2 * n) - z * math.sqrt((p * (1 - p) + z * z / (4 * n)) / n)) / (1 + z * z / n)
    hi = (p + z * z / (2 * n) + z * math.sqrt((p * (1 - p) + z * z / (4 * n)) / n)) / (1 + z * z / n)
    return (hi - lo) / 2


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("worksheet", nargs="?", default="bench/measurement/out-val/labeling-worksheet.jsonl")
    ap.add_argument("out", nargs="?", default="bench/measurement/out-val/labeling-packet-sample.csv")
    ap.add_argument("--target", type=int, default=400, help="approx total rows to sample")
    ap.add_argument("--seed", default="keyway-g1-sample-2026")
    args = ap.parse_args()

    # Build the full universe of (repo, field, value) rows with their side + flags.
    universe = []
    for line in open(args.worksheet):
        r = json.loads(line)
        has_consumer = bool(r.get("captured_issuers"))
        nonauth = r.get("nonauth_declared", {})
        for f in FIELDS:
            declared = set(r.get(f"declared_{f}", []))
            captured = set(r.get(f"captured_{f}", []))
            na = set(nonauth.get(f, []))
            for v in (declared & captured):
                universe.append(_row(r["repo"], f, v, "agree", False, has_consumer))
            for v in (captured - declared):
                universe.append(_row(r["repo"], f, v, "captured-only", False, has_consumer))
            for v in (declared - captured):
                universe.append(_row(r["repo"], f, v, "declared-only", v in na, has_consumer))

    frac = min(1.0, args.target / max(1, len(universe)))
    # Same sampling rate in every (field, side) stratum => self-weighting, but each
    # stratum guaranteed its proportional share (deterministic by hash rank).
    strata = collections.defaultdict(list)
    for row in universe:
        strata[(row["field"], row["side"])].append(row)
    sample = []
    for key, rows in strata.items():
        rows.sort(key=lambda x: hashlib.sha256(
            f'{args.seed}|{x["repo"]}|{x["field"]}|{x["value"]}'.encode()).hexdigest())
        k = round(frac * len(rows))
        sample.extend(rows[:k])

    with open(args.out, "w", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=COLS)
        w.writeheader()
        w.writerows(sample)

    # Report the sample composition and the CI half-widths it can support.
    by_side = collections.Counter(r["side"] for r in sample)
    cap = by_side["agree"] + by_side["captured-only"]      # precision denominator
    dec = by_side["agree"] + by_side["declared-only"]      # recall denominator
    print(f"universe {len(universe)} rows; sampled {len(sample)} ({frac:.0%})")
    print(f"  by field: {dict(collections.Counter(r['field'] for r in sample))}")
    print(f"  by side:  {dict(by_side)}")
    print(f"  precision denominator (captured in sample) = {cap}; "
          f"Wilson 95% half-width at p=0.97 ~ +/-{wilson_halfwidth(0.97, cap):.1%}")
    print(f"  recall denominator (declared in sample)    = {dec}; "
          f"Wilson 95% half-width at p=0.94 ~ +/-{wilson_halfwidth(0.94, dec):.1%}")
    print(f"wrote {args.out}")


def _row(repo, field, value, side, in_nonauth, has_consumer):
    return {"repo": repo, "field": field, "value": value, "side": side,
            "in_nonauth_context": in_nonauth, "has_jwt_consumer": has_consumer,
            "suggested_label": suggest(side, in_nonauth, has_consumer),
            "human_label": "", "notes": ""}


if __name__ == "__main__":
    main()
