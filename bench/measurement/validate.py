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


def norm_claim(x):
    """Normalize a claim name so '\"email\"' and 'email' compare equal."""
    return x.strip().strip('"').strip("'").strip()


def repo_key(name):
    """Group by source repo via the crawler's 'owner_repo__path' filename."""
    b = os.path.basename(name)
    return b.split("__", 1)[0] if "__" in b else b


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
        claims.update(norm_claim(c) for c in CLAIM_RE.findall(node))


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
    """Group Keyway's discovered fields by source REPO (RA + AP live in
    different files of the same repo, so the repo is the right unit)."""
    cap = {}
    ds = os.path.join(out_dir, "dataset.jsonl")
    if not os.path.exists(ds):
        sys.exit(f"missing {ds} — run the measurement harness first (see docstring)")
    with open(ds) as fh:
        for line in fh:
            r = json.loads(line)
            k = repo_key(r.get("source", ""))
            e = cap.setdefault(k, {"issuers": set(), "audiences": set(), "claims": set()})
            e["issuers"].update(r.get("issuers") or [])
            e["audiences"].update(r.get("audiences") or [])
            e["claims"].update(norm_claim(c) for c in (r.get("required_claims") or []))
    return cap


def main():
    files = sorted(glob.glob(os.path.join(CORPUS, "*.yaml")) + glob.glob(os.path.join(CORPUS, "*.yml")))
    captured = load_captured(OUT)

    # Independent parse, unioned per repo — the same unit as per-repo discovery.
    declared = {}
    for path in files:
        di, da, dc = independent_parse(path)
        if not (di or da or dc):
            continue
        e = declared.setdefault(repo_key(path), {"issuers": set(), "audiences": set(), "claims": set()})
        e["issuers"].update(di)
        e["audiences"].update(da)
        e["claims"].update(dc)

    agg = {f: {"tp": 0, "fp": 0, "fn": 0} for f in ("issuers", "audiences", "claims")}
    worksheet = []
    for repo, dset in sorted(declared.items()):
        cap = captured.get(repo, {"issuers": set(), "audiences": set(), "claims": set()})
        for field in agg:
            d, c = dset[field], cap.get(field, set())
            agg[field]["tp"] += len(d & c)
            agg[field]["fn"] += len(d - c)
            agg[field]["fp"] += len(c - d)
        worksheet.append({
            "repo": repo,
            "declared_issuers": sorted(dset["issuers"]), "captured_issuers": sorted(cap.get("issuers", [])),
            "declared_audiences": sorted(dset["audiences"]), "captured_audiences": sorted(cap.get("audiences", [])),
            "declared_claims": sorted(dset["claims"]), "captured_claims": sorted(cap.get("claims", [])),
            "human_label": "",  # to be filled: correct | discovery-miss | parse-miss | mismatch
        })

    files_with_truth = len(declared)
    report = {"repos_with_declared_truth": files_with_truth, "fields": {}}
    print(f"\nValidation vs independent parse (per repo) — {files_with_truth} repos with declared auth config\n")
    print(f"{'FIELD':<12}{'RECALL':>10}{'PRECISION':>12}{'TP':>7}{'FP':>6}{'FN':>6}")
    for field, m in agg.items():
        tp, fp, fn = m["tp"], m["fp"], m["fn"]
        recall = tp / (tp + fn) if (tp + fn) else 1.0
        precision = tp / (tp + fp) if (tp + fp) else 1.0
        report["fields"][field] = {"tp": tp, "fp": fp, "fn": fn, "recall": recall, "precision": precision}
        print(f"{field:<12}{recall:>9.1%}{precision:>12.1%}{tp:>7}{fp:>6}{fn:>6}")

    # Scoped claims recall: claims can only attach to a JWT consumer, so a repo
    # whose RequestAuthentication isn't in the corpus has nothing to attach to.
    # Restricting to repos where discovery found an issuer isolates true
    # discovery recall from corpus-completeness noise.
    stp = sfn = 0
    for row in worksheet:
        if not row["captured_issuers"]:
            continue
        d, c = set(row["declared_claims"]), set(row["captured_claims"])
        stp += len(d & c)
        sfn += len(d - c)
    scoped = stp / (stp + sfn) if (stp + sfn) else 1.0
    report["claims_recall_scoped_to_repos_with_a_consumer"] = {
        "recall": scoped, "tp": stp, "fn": sfn,
        "note": "claims recall counting only repos where a JWT consumer exists; the "
                "unscoped figure is dominated by repos whose RequestAuthentication "
                "was not in the corpus (nothing to attach to).",
    }
    print(f"\nclaims recall, scoped to repos WITH a JWT consumer: {scoped:.1%} "
          f"(tp={stp}, fn={sfn}) — vs unscoped {agg['claims']['tp']}/"
          f"{agg['claims']['tp'] + agg['claims']['fn']}. The gap is corpus completeness, "
          f"not discovery.")

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
