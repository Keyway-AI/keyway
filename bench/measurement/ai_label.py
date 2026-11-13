#!/usr/bin/env python3
"""Independent AI first-pass labeller for the G1 validation packet.

An LLM labeller that reads the **raw source config** for each contested
`(repo, field, value)` row and assigns a label from the LABELING.md taxonomy,
*independently of Keyway's discovery output* (it never sees Keyway's reasoning,
only the raw YAML and which side found the value), so using it to grade Keyway's
discovery is not circular.

It produces the **AI-only label set (v1)**: an `ai_label` plus a calibrated
`ai_confidence` and a one-line `ai_rationale`. A human then fills `human_label`
(the **human-validated set, v2**) using the AI draft as a reference, and
`grade_dual.py` reports precision/recall under both sets plus their agreement
(Cohen's kappa). See LABELING.md.

  export ANTHROPIC_API_KEY=...   # or `ant auth login`
  python3 bench/measurement/ai_label.py \
      bench/measurement/out/labeling-packet.csv \
      bench/measurement/out/labeling-packet-ai.csv \
      --corpus bench/measurement/corpus [--model claude-opus-5-5] [--limit N]

For the scaled corpus (gate G2), the same labeller runs over the whole population
(hand-labelling 10^3-10^4 rows is infeasible; an AI pass plus a human-validated
sample is). Use a cheaper model there (e.g. --model claude-sonnet-5-5).
"""
import argparse
import csv
import glob
import json
import os
import sys
import threading
from concurrent.futures import ThreadPoolExecutor

VALID = {"correct", "correct-extra", "discovery-miss", "parser-artifact",
         "no-consumer", "spurious", "wrong-attribution"}

RUBRIC = """\
You are building a gold-standard validation sample for a measurement study of how
well a tool ("the discoverer") extracts JWT authentication config from real
Kubernetes/Istio/Envoy YAML. Genuine auth config lives in Istio
RequestAuthentication (issuers/audiences), Istio AuthorizationPolicy
(when[].key request.auth.claims[...] -> required claims), and Envoy http
jwt_authn filters (providers: issuer/audiences). Values scraped from ConfigMaps,
annotations, comments, unrelated env vars, or example/placeholder text are NOT
genuine auth config.

You are given ONE value the independent reviewers flagged, the field it belongs to
(issuers | audiences | claims), and which side saw it:
  - "declared-only": an independent parse found this value but the discoverer did
    NOT capture it. Decide whether the discoverer missed a real value or whether
    the value is not genuine auth config / has no consumer to attach to.
  - "captured-only": the discoverer captured this value but the independent parse
    did not. Decide whether the captured value is genuinely real or not.
  - "agree": both captured it (calibration). Almost always "correct".

Judge ONLY from the raw config below and general knowledge of these schemas. Do
not assume the discoverer is right or wrong. Choose exactly one label:

  correct           : the value is genuine auth config AND correctly captured/attributed.
  correct-extra     : captured-only, and the value IS genuinely real (the independent
                      parse missed it). [counts as a true positive]
  discovery-miss    : declared-only, a genuine auth value in a real auth resource that
                      the discoverer should have captured but did not. [false negative]
  parser-artifact   : the value was scraped from a non-auth context (ConfigMap key,
                      annotation, comment, unrelated field) -> not genuine. [excluded]
  no-consumer       : a genuinely real value, but there is no RequestAuthentication /
                      jwt_authn consumer in this repo's config for it to attach to. [excluded]
  spurious          : captured-only, but the value is NOT genuine auth config. [false positive]
  wrong-attribution : a genuine value the discoverer attached to the WRONG consumer. [false positive]

Return ONLY a JSON object, no prose, no code fence:
{"label": "<one label>", "confidence": <0.0-1.0>, "rationale": "<one sentence>"}
"""

MAX_CFG = 48000   # cap total config chars fed per row
MAX_FILE = 16000  # cap per file


def load_config(corpus_dir, repo):
    """Concatenate the raw config files for a repo (repo uses '_' for '/')."""
    paths = sorted(glob.glob(os.path.join(corpus_dir, f"{repo}__*")))
    chunks, total = [], 0
    for p in paths:
        try:
            text = open(p, encoding="utf-8", errors="replace").read()
        except OSError:
            continue
        if len(text) > MAX_FILE:
            text = text[:MAX_FILE] + "\n...[truncated]...\n"
        name = os.path.basename(p)
        chunks.append(f"### FILE: {name}\n{text}")
        total += len(text)
        if total > MAX_CFG:
            chunks.append("...[remaining files omitted]...")
            break
    return "\n\n".join(chunks) if chunks else "(no source files found for this repo in the corpus)"


def label_row(client, model, row, config_text):
    user = (
        f"FIELD: {row['field']}\n"
        f"VALUE: {row['value']}\n"
        f"SIDE: {row['side']}\n"
        f"in_nonauth_context (heuristic): {row['in_nonauth_context']}\n"
        f"repo_has_a_jwt_consumer: {row['has_jwt_consumer']}\n\n"
        f"RAW CONFIG FOR repo {row['repo']}:\n{config_text}"
    )
    msg = client.messages.create(
        model=model,
        max_tokens=4096,  # headroom: Opus 5.5 always thinks before the small JSON answer
        system=RUBRIC,
        messages=[{"role": "user", "content": user}],
    )
    text = "".join(b.text for b in msg.content if b.type == "text").strip()
    # tolerant parse: take the first {...} block
    start, end = text.find("{"), text.rfind("}")
    obj = json.loads(text[start:end + 1]) if start >= 0 and end > start else {}
    label = str(obj.get("label", "")).strip().lower()
    if label not in VALID:
        label = "review"
    conf = obj.get("confidence", "")
    try:
        conf = round(float(conf), 2)
    except (TypeError, ValueError):
        conf = ""
    return label, conf, str(obj.get("rationale", "")).strip()[:300]


OUT_COLS = ["repo", "field", "value", "side", "in_nonauth_context",
            "has_jwt_consumer", "suggested_label", "ai_label", "ai_confidence",
            "ai_rationale", "human_label", "notes"]


def key_of(r):
    return (r["repo"], r["field"], r["value"])


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("packet", nargs="?", default="bench/measurement/out/labeling-packet.csv")
    ap.add_argument("out", nargs="?", default="bench/measurement/out/labeling-packet-ai.csv")
    ap.add_argument("--corpus", default="bench/measurement/corpus")
    ap.add_argument("--model", default="claude-opus-5-5")
    ap.add_argument("--workers", type=int, default=6, help="parallel API calls")
    ap.add_argument("--limit", type=int, default=0, help="label only the first N rows (testing)")
    ap.add_argument("--fresh", action="store_true", help="ignore any prior output and relabel all")
    args = ap.parse_args()

    try:
        import anthropic
    except ImportError:
        sys.exit("pip install anthropic  (then set ANTHROPIC_API_KEY or run `ant auth login`)")
    client = anthropic.Anthropic()

    rows = list(csv.DictReader(open(args.packet)))
    if args.limit:
        rows = rows[:args.limit]

    # RESUME: keep prior rows whose ai_label is already a final (valid) label; redo
    # everything else (blank / error / review) plus rows never attempted. A re-run
    # therefore never repeats completed work — restart after a kill and it continues.
    done = {}
    if not args.fresh and os.path.exists(args.out):
        for r in csv.DictReader(open(args.out)):
            if (r.get("ai_label") or "").strip().lower() in VALID:
                done[key_of(r)] = r
    todo = [r for r in rows if key_of(r) not in done]
    print(f"{len(rows)} packet rows | already done: {len(done)} | to label now: {len(todo)} "
          f"| model={args.model} workers={args.workers}", flush=True)

    # Re-persist the preserved rows first (in packet order), then append new results
    # as they complete (lock-guarded, flushed), so a kill never loses progress.
    fh = open(args.out, "w", newline="")
    w = csv.DictWriter(fh, fieldnames=OUT_COLS)
    w.writeheader()
    for r in rows:
        if key_of(r) in done:
            w.writerow({c: done[key_of(r)].get(c, "") for c in OUT_COLS})
    fh.flush()

    cfg_cache, cfg_lock, io_lock = {}, threading.Lock(), threading.Lock()
    counter = [len(done)]

    def get_cfg(repo):
        with cfg_lock:
            if repo not in cfg_cache:
                cfg_cache[repo] = load_config(args.corpus, repo)
            return cfg_cache[repo]

    def work(r):
        try:
            lab, conf, why = label_row(client, args.model, r, get_cfg(r["repo"]))
        except Exception as e:  # a failed row is marked 'error' (redone on next resume)
            lab, conf, why = "error", "", f"{type(e).__name__}: {e}"[:200]
        rec = {
            "repo": r["repo"], "field": r["field"], "value": r["value"], "side": r["side"],
            "in_nonauth_context": r["in_nonauth_context"], "has_jwt_consumer": r["has_jwt_consumer"],
            "suggested_label": r.get("suggested_label", ""), "ai_label": lab,
            "ai_confidence": conf, "ai_rationale": why, "human_label": "", "notes": r.get("notes", ""),
        }
        with io_lock:
            w.writerow(rec)
            fh.flush()
            counter[0] += 1
            print(f"[{counter[0]}/{len(rows)}] {r['repo']} {r['field']}={r['value'][:36]} -> {lab} ({conf})", flush=True)

    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as ex:
        list(ex.map(work, todo))
    fh.close()

    final = list(csv.DictReader(open(args.out)))
    pending = sum(1 for r in final if (r.get("ai_label") or "") in ("", "error", "review"))
    print(f"\nwrote {args.out}: {len(final)} rows ({pending} still error/review — "
          f"re-run the same command to retry just those)")
    print("Next: a human fills `human_label` (taxonomy in LABELING.md), then")
    print("  python3 bench/measurement/grade_dual.py", args.out)


if __name__ == "__main__":
    main()
