#!/usr/bin/env python3
"""Validation: measure Keyway discovery precision/recall on the REAL corpus.

Ground truth here is an INDEPENDENT YAML parse of each config file (what a plain
walk finds declared: issuers, audiences, required-claim names). Captured = what
Keyway's discovery emitted (bench/measurement/out/dataset.jsonl, produced by the
measurement harness). We compare the two per field and aggregate.

This independent-parse proxy is a real, defensible validation signal now; a
hand-labelled sample (the gold standard reviewers expect) is the next step, and
this script also emits a labelling worksheet to bootstrap it.

Usage:
  go run ./bench/measurement --path bench/measurement/corpus --per-file   # → out/dataset.jsonl
  python3 bench/measurement/validate.py bench/measurement/corpus bench/measurement/out
"""
import glob
import json
import os
import re
import sys

import yaml  # PyYAML

CORPUS = sys.argv[1] if len(sys.argv) > 1 else "bench/measurement/corpus"
OUT = sys.argv[2] if len(sys.argv) > 2 else "bench/measurement/out"
CLAIM_RE = re.compile(r"request\.auth\.claims\[([^\]]+)\]")


def walk(node, issuers, auds, claims):
    """Independently collect declared issuers, audiences, and claim names."""
    if isinstance(node, dict):
        if isinstance(node.get("issuer"), str):
            issuers.add(node["issuer"].strip())
            a = node.get("audiences")
            if isinstance(a, list):
                auds.update(x.strip() for x in a if isinstance(x, str))
        for v in node.values():
            walk(v, issuers, auds, claims)
    elif isinstance(node, list):
        for v in node:
            walk(v, issuers, auds, claims)
    elif isinstance(node, str):
        claims.update(CLAIM_RE.findall(node))


def independent_parse(path):
    issuers, auds, claims = set(), set(), set()
    try:
        with open(path) as fh:
            for doc in yaml.safe_load_all(fh):
                walk(doc, issuers, auds, claims)
    except Exception:
        pass
    return issuers, auds, claims


def load_captured(out_dir):
    """Group Keyway's discovered fields by source filename."""
    cap = {}
    ds = os.path.join(out_dir, "dataset.jsonl")
    if not os.path.exists(ds):
        sys.exit(f"missing {ds} — run the measurement harness first (see docstring)")
    with open(ds) as fh:
        for line in fh:
            r = json.loads(line)
            src = r.get("source", "")
            e = cap.setdefault(src, {"issuers": set(), "audiences": set(), "claims": set()})
            e["issuers"].update(r.get("issuers") or [])
            e["audiences"].update(r.get("audiences") or [])
            e["claims"].update(r.get("required_claims") or [])
    return cap


def basename_key(path):
    # The harness records source = filepath.Base(corpus file); match on that.
    return os.path.basename(path)


def main():
    files = sorted(glob.glob(os.path.join(CORPUS, "*.yaml")) + glob.glob(os.path.join(CORPUS, "*.yml")))
    captured = load_captured(OUT)

    agg = {f: {"tp": 0, "fp": 0, "fn": 0} for f in ("issuers", "audiences", "claims")}
    worksheet = []
    files_with_truth = 0

    for path in files:
        di, da, dc = independent_parse(path)
        if not (di or da or dc):
            continue  # nothing JWT-ish declared → not a validation unit
        files_with_truth += 1
        cap = captured.get(basename_key(path), {"issuers": set(), "audiences": set(), "claims": set()})
        declared = {"issuers": di, "audiences": da, "claims": dc}
        for field in agg:
            d, c = declared[field], cap.get(field, set())
            agg[field]["tp"] += len(d & c)
            agg[field]["fn"] += len(d - c)
            agg[field]["fp"] += len(c - d)
        worksheet.append({
            "file": basename_key(path),
            "declared_issuers": sorted(di), "captured_issuers": sorted(cap.get("issuers", [])),
            "declared_audiences": sorted(da), "captured_audiences": sorted(cap.get("audiences", [])),
            "declared_claims": sorted(dc), "captured_claims": sorted(cap.get("claims", [])),
            "human_label": "",  # to be filled: correct | discovery-miss | parse-miss | mismatch
        })

    report = {"files_with_declared_truth": files_with_truth, "fields": {}}
    print(f"\nValidation vs independent parse — {files_with_truth} files with declared auth config\n")
    print(f"{'FIELD':<12}{'RECALL':>10}{'PRECISION':>12}{'TP':>7}{'FP':>6}{'FN':>6}")
    for field, m in agg.items():
        tp, fp, fn = m["tp"], m["fp"], m["fn"]
        recall = tp / (tp + fn) if (tp + fn) else 1.0
        precision = tp / (tp + fp) if (tp + fp) else 1.0
        report["fields"][field] = {"tp": tp, "fp": fp, "fn": fn, "recall": recall, "precision": precision}
        print(f"{field:<12}{recall:>9.1%}{precision:>12.1%}{tp:>7}{fp:>6}{fn:>6}")

    os.makedirs(OUT, exist_ok=True)
    with open(os.path.join(OUT, "validation.json"), "w") as fh:
        json.dump(report, fh, indent=2)
    # Emit a labelling worksheet (a stratified head; the paper labels the full sample).
    with open(os.path.join(OUT, "labeling-worksheet.jsonl"), "w") as fh:
        for row in worksheet:
            fh.write(json.dumps(row) + "\n")
    print(f"\nwrote {OUT}/validation.json and {OUT}/labeling-worksheet.jsonl "
          f"({len(worksheet)} rows for human labelling)")
    print("NOTE: independent-parse proxy, not human ground truth — a hand-labelled")
    print("sample is the gold standard the paper needs (worksheet emitted above).")


if __name__ == "__main__":
    main()
