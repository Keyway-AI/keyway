#!/usr/bin/env bash
# Empirical eval: start the Keyway test issuer + 8 real verifiers, fire the adversarial
# corpus at them, and record which accept tokens they should reject. Staging/localhost
# only. See docs/research/empirical-eval-plan.md.
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
V="$ROOT/bench/verifiers"
OUT="$V/out"; mkdir -p "$OUT"
PYDEPS="${PYDEPS:-/tmp/pydeps}"
PY="${PY:-/usr/bin/python3}"
KEY=/tmp/keyway-eval-issuerkey.json
PIDS=()
cleanup(){ for p in "${PIDS[@]:-}"; do kill "$p" 2>/dev/null; done; docker rm -f keyway-envoy >/dev/null 2>&1; }
trap cleanup EXIT

echo "== build =="
go build -o "$OUT/keyway" "$ROOT/cmd/keyway" || exit 1
go build -o "$OUT/issuer-server" "$V/issuer-server" || exit 1
( cd "$V/targets/go-golangjwt" && go build -o "$OUT/t-golangjwt" . ) || exit 1
( cd "$V/targets/go-jwx" && go build -o "$OUT/t-jwx" . ) || exit 1
( cd "$V/targets/go-gojose" && go build -o "$OUT/t-gojose" . ) || exit 1

echo "== issuer =="
"$OUT/issuer-server" -addr :8099 -key-out "$KEY" >"$OUT/issuer.log" 2>&1 & PIDS+=($!)
sleep 1.5

echo "== targets =="
PORT=10001 "$OUT/t-golangjwt" >"$OUT/go-golangjwt.log" 2>&1 & PIDS+=($!)
PORT=10002 "$OUT/t-jwx"       >"$OUT/go-jwx.log" 2>&1 & PIDS+=($!)
PORT=10010 "$OUT/t-gojose"    >"$OUT/go-gojose.log" 2>&1 & PIDS+=($!)
( cd "$V/targets/node-jsonwebtoken" && PORT=10003 node index.js ) >"$OUT/node-jsonwebtoken.log" 2>&1 & PIDS+=($!)
( cd "$V/targets/node-jose" && PORT=10004 node index.mjs ) >"$OUT/node-jose.log" 2>&1 & PIDS+=($!)
PYTHONPATH="$PYDEPS" PORT=10005 "$PY" "$V/targets/py-pyjwt/server.py"   >"$OUT/py-pyjwt.log" 2>&1 & PIDS+=($!)
PYTHONPATH="$PYDEPS" PORT=10006 "$PY" "$V/targets/py-authlib/server.py" >"$OUT/py-authlib.log" 2>&1 & PIDS+=($!)
PYTHONPATH="$PYDEPS" PORT=10008 "$PY" "$V/targets/py-jose/server.py"    >"$OUT/py-jose.log" 2>&1 & PIDS+=($!)
docker run -d --rm --name keyway-envoy -p 10007:10007 \
  -v "$V/envoy/envoy-host.yaml:/etc/envoy/envoy.yaml:ro" \
  envoyproxy/envoy:v1.31-latest >/dev/null 2>&1
sleep 4

echo "== readiness (401 = up and rejecting no-token; anything else = investigate) =="
for pp in 10001:/api/protected 10002:/api/protected 10003:/ 10004:/ 10005:/ 10006:/ 10007:/ 10008:/; do
  port=${pp%%:*}; path=${pp#*:}
  code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 3 "http://localhost:$port$path")
  echo "  :$port$path -> $code"
done

echo "== run harness =="
"$OUT/keyway" probe --harness \
  --issuer-url http://localhost:8099 --issuer-key "$KEY" \
  --allow localhost --allow 127.0.0.1 \
  --consumers "$V/consumers/consumers.json" 2>"$OUT/harness.err" | tee "$OUT/results.txt"
