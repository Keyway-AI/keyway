#!/usr/bin/env python3
"""Build a human-labelling packet (CSV) from the validation worksheet.

The automated validators (validate.py) are proxies. This packet turns the
contested cases into an hour of human adjudication that yields the gold-standard
precision/recall the paper needs (research gate G1). It focuses on the
disagreements between the independent parse and Keyway's discovery (the agreements
are trivially correct); a small calibration sample of agreements is included so an
annotator can confirm their labels track the obvious cases.

Run (after `go run ./bench/measurement --path ... --per-repo` + validate.py):
  python3 bench/measurement/make_labeling_packet.py \
      bench/measurement/out/labeling-worksheet.jsonl \
      bench/measurement/out/labeling-packet.csv

Then a human fills the `human_label` column using the taxonomy in LABELING.md, and
`grade_labels.py` (see LABELING.md) turns the labels into the final numbers.
The packet is written under out/ (gitignored) — it is derived from the local corpus.
"""
import csv
import json
import sys

WORKSHEET = sys.argv[1] if len(sys.argv) > 1 else "bench/measurement/out/labeling-worksheet.jsonl"
OUT = sys.argv[2] if len(sys.argv) > 2 else "bench/measurement/out/labeling-packet.csv"
FIELDS = ("issuers", "audiences", "claims")

# Label taxonomy (see LABELING.md). `human_label` must be one of these.
LABELS = "correct | discovery-miss | parser-artifact | no-consumer | spurious | wrong-attribution"


def suggest(side, in_nonauth, has_consumer):
    if side == "captured-only":
        return "review(likely correct-extra or spurious)"
    if in_nonauth:
        return "parser-artifact"
    return "discovery-miss" if has_consumer else "no-consumer"


def main():
    rows = [json.loads(l) for l in open(WORKSHEET)]
    out_rows, calib = [], []
    for r in rows:
        has_consumer = bool(r.get("captured_issuers"))
        nonauth = r.get("nonauth_declared", {})
        for f in FIELDS:
            declared = set(r.get(f"declared_{f}", []))
            captured = set(r.get(f"captured_{f}", []))
            na = set(nonauth.get(f, []))
            for v in sorted(declared - captured):
                out_rows.append({
                    "repo": r["repo"], "field": f, "value": v, "side": "declared-only",
                    "in_nonauth_context": v in na, "has_jwt_consumer": has_consumer,
                    "suggested_label": suggest("declared-only", v in na, has_consumer),
                    "human_label": "", "notes": "",
                })
            for v in sorted(captured - declared):
                out_rows.append({
                    "repo": r["repo"], "field": f, "value": v, "side": "captured-only",
                    "in_nonauth_context": False, "has_jwt_consumer": has_consumer,
                    "suggested_label": suggest("captured-only", False, has_consumer),
                    "human_label": "", "notes": "",
                })
            # calibration: a few agreements per field (both sides captured it)
            for v in sorted(declared & captured)[:1]:
                calib.append({
                    "repo": r["repo"], "field": f, "value": v, "side": "agree",
                    "in_nonauth_context": False, "has_jwt_consumer": has_consumer,
                    "suggested_label": "correct", "human_label": "", "notes": "calibration",
                })

    # cap calibration to ~15% of the contested volume so the packet stays focused
    calib = calib[: max(5, len(out_rows) // 6)]
    all_rows = out_rows + calib
    cols = ["repo", "field", "value", "side", "in_nonauth_context",
            "has_jwt_consumer", "suggested_label", "human_label", "notes"]
    with open(OUT, "w", newline="") as fh:
        w = csv.DictWriter(fh, fieldnames=cols)
        w.writeheader()
        w.writerows(all_rows)
    print(f"wrote {OUT}: {len(out_rows)} contested + {len(calib)} calibration rows")
    print(f"label taxonomy: {LABELS}")
    print("Open in a spreadsheet, fill `human_label`, then grade with grade_labels.py (see LABELING.md).")


if __name__ == "__main__":
    main()
