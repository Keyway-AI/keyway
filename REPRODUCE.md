# Reproducing the Keyway papers

This repository is the artifact for the two Keyway papers. One command reproduces
every claim that can be verified **offline** (no network, no credentials) on a
clean checkout:

```bash
make reproduce         # or: bash scripts/reproduce.sh artifact
```

It writes to `artifact/` (gitignored) and prints a claim→output map, also saved as
`artifact/CLAIMS.md`.

## Requirements

- Go **1.25+** (the script records the exact toolchain in `artifact/environment.txt`).
- No network, no token, no database for the offline claims below.

## What `make reproduce` verifies offline

| Paper claim | Output | Check |
|---|---|---|
| **Paper A** — drift classifier over a 1,226-pair synthetic corpus: 100% recall / 0% FPR, held-out **Youden 0.75** | `artifact/scorecard.json` | `L3` TPR=1.0, FPR=0.0; `L3-adversarial` Youden=0.75 |
| **Paper A** — non-circular negative control: **4 planted distractors, 0 leaked** | `artifact/tests.txt` | the `NegControl` test passes |
| **Paper B** — **6 of 15** agent threats statically checkable, and the analyzer covers them | `artifact/threat-coverage.md` | agent domain shows 6/15; DEL-02 lists `analyzer:delegation_chain` |
| Instrument tooling (template rendering, canonical dedup, co-occurrence, static/runtime frontier) | `artifact/tests.txt` | `render`, `dedup`, `cooccur` package tests pass |

## What needs a GitHub token (not run by `make reproduce`)

**Paper A's corpus-prevalence numbers** — n=102 distinct configs: unbound audience
42.2%, no required claims 85.3%, no algorithm pinning 100% (Wilson 95% CIs), plus
the extraction recall/validation figures — are derived from a read-only crawl of
public GitHub. That step needs a token and network, so it is documented here rather
than run automatically:

```bash
# 1. crawl public config into bench/measurement/corpus (read-only, public repos only)
GH_TOKEN=<token> MAX_PAGES=2 bash bench/measurement/crawl.sh

# 2. measure prevalence (canonical dedup + template resolution + co-occurrence + frontier)
go run ./bench/measurement --path bench/measurement/corpus \
  --per-repo --exclude-examples --dedup --resolve-templates --out bench/measurement/out

# 3. validate extraction against an independent parser
go run ./bench/measurement --path bench/measurement/corpus --per-repo --out bench/measurement/out
python3 bench/measurement/validate.py bench/measurement/corpus bench/measurement/out
```

The crawl is a snapshot; `bench/measurement/sources.tsv` records the repo + blob SHA
+ licence of every artifact in the preliminary run for attribution. See
[`bench/measurement/FINDINGS.md`](bench/measurement/FINDINGS.md) for methodology,
caveats, and the honest "why this is preliminary" notes.

## Also useful

- `make bench` — the drift benchmark alone.
- `make coverage` — regenerate `docs/threat-coverage.md`.
- `make mutation` — mutation-test the drift detector (slow; proves the corpus is not overfit).
