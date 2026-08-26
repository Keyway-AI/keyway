#!/usr/bin/env bash
# Reproduce the offline-verifiable claims in the Keyway papers into <outdir>
# (default: artifact/). Everything here runs with NO network and NO token, so an
# artifact reviewer can re-run it on a clean checkout. The corpus-prevalence
# numbers (Paper A) need a GitHub crawl and are documented, not run, here — see
# REPRODUCE.md.
#
# Usage:  bash scripts/reproduce.sh [outdir]
set -euo pipefail
cd "$(dirname "$0")/.."
out="${1:-artifact}"
mkdir -p "$out"

echo "==> environment"
{
  go version
  echo "git-commit: $(git rev-parse HEAD)"
  echo "module: $(head -1 go.mod)"
  uname -sm
} | tee "$out/environment.txt"

echo
echo "==> drift-classifier benchmark"
echo "    (Paper A: 1,226-pair synthetic corpus, 100% recall / 0% FPR, held-out Youden 0.75)"
go run ./bench/harness --corpus ./bench/corpus --out "$out/bench"
cp "$out/bench/scorecard.json" "$out/scorecard.json"

echo
echo "==> threat-coverage taxonomy"
echo "    (Paper B: agent domain 6 of 15 statically checkable; the analyzer covers those 6)"
go run ./cmd/keyway threats coverage > "$out/threat-coverage.md"

echo
echo "==> offline test suite"
echo "    (Paper A negative control: 4 planted distractors / 0 leaked; template/dedup/co-occurrence; the agent analyzer)"
go test ./bench/measurement/... ./internal/agentauth/... ./internal/threats/... >"$out/tests.txt" 2>&1
tail -4 "$out/tests.txt"

echo
echo "==> claims map"
cat > "$out/CLAIMS.md" <<'EOF'
# Reproduced claims → outputs

| Paper claim | Output | How to check |
|---|---|---|
| Paper A — drift classifier: 1,226 pairs, 100% recall / 0% FPR, held-out Youden 0.75 | `scorecard.json` | `L3` layer TPR=1.0 FPR=0.0; `L3-adversarial` Youden=0.75 |
| Paper A — negative control: 4 planted distractors, 0 leaked | `tests.txt` | the `NegControl` test passes |
| Paper B — 6 of 15 agent threats statically checkable; analyzer covers them | `threat-coverage.md` | agent domain shows 6/15; DEL-02 lists `analyzer:delegation_chain` |
| Template / dedup / co-occurrence / static-vs-runtime tooling | `tests.txt` | the `render`, `dedup`, `cooccur` package tests pass |

**Not reproduced here (needs a GitHub token + crawl):** Paper A corpus prevalence
(n=102: unbound-audience 42.2%, no-claims 85.3%, no-alg-pin 100%) and the
extraction recall/validation numbers. See REPRODUCE.md for the exact crawl +
measure commands.
EOF
echo "    wrote $out/CLAIMS.md"

echo
echo "Artifact written to $out/ — see $out/CLAIMS.md for the claim→output map."
