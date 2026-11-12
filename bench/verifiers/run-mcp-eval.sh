#!/usr/bin/env bash
# A7 / Paper B: fire the adversarial agent-token corpus at real MCP-server-class
# verifiers and record which accept tokens a correct MCP resource server must reject
# (headline: MCP-01/02 audience binding / token passthrough, SCOPE-02 non-expiry).
# Staging/localhost only. The MCP SDKs need Python >= 3.10, so this uses a dedicated
# uv venv rather than the 3.9 interpreter the library targets (run-eval.sh) use.
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
V="$ROOT/bench/verifiers"
OUT="$V/out"; mkdir -p "$OUT"
KEY=/tmp/keyway-eval-issuerkey.json
VENV="${MCP_VENV:-/tmp/mcpvenv}"
PY="$VENV/bin/python"
PIDS=()
cleanup(){ for p in "${PIDS[@]:-}"; do kill "$p" 2>/dev/null; done; }
trap cleanup EXIT

echo "== venv (needs uv + Python >=3.10) =="
if [ ! -x "$PY" ]; then
  command -v uv >/dev/null || { echo "uv not found; install uv or set MCP_VENV to a ready venv"; exit 1; }
  uv venv "$VENV" --python 3.13 || exit 1
fi
uv pip install --python "$PY" -r "$V/targets/mcp-requirements.txt" >"$OUT/mcp-deps.log" 2>&1 || { echo "dep install failed; see $OUT/mcp-deps.log"; exit 1; }

echo "== build =="
go build -o "$OUT/keyway" "$ROOT/cmd/keyway" || exit 1
go build -o "$OUT/issuer-server" "$V/issuer-server" || exit 1

echo "== free ports =="
for p in 8099 10011 10012; do lsof -ti tcp:$p 2>/dev/null | xargs -r kill -9 2>/dev/null; done

echo "== issuer =="
"$OUT/issuer-server" -addr :8099 -key-out "$KEY" >"$OUT/mcp-issuer.log" 2>&1 & PIDS+=($!)
sleep 2

echo "== MCP targets =="
AUDIENCE=mcp-rs-A PORT=10011 JWKS_URL=http://localhost:8099/.well-known/jwks.json ISSUER=http://localhost:8099 \
  "$PY" "$V/targets/mcp-fastmcp/server.py" >"$OUT/mcp-fastmcp.log" 2>&1 & PIDS+=($!)
AUDIENCE=mcp-sdk-rs-A PORT=10012 JWKS_URL=http://localhost:8099/.well-known/jwks.json ISSUER=http://localhost:8099 \
  "$PY" "$V/targets/mcp-python-sdk/server.py" >"$OUT/mcp-sdk.log" 2>&1 & PIDS+=($!)
sleep 5

echo "== readiness (401 = up and rejecting no-token) =="
for p in 10011 10012; do
  echo "  :$p -> $(curl -s -o /dev/null -w '%{http_code}' --max-time 4 http://localhost:$p/)"
done

echo "== conformance report (human) =="
"$OUT/keyway" probe --harness --report conformance \
  --issuer-url http://localhost:8099 --issuer-key "$KEY" \
  --allow localhost --allow 127.0.0.1 \
  --consumers "$V/consumers/consumers-mcp.json" 2>>"$OUT/mcp-harness.err" | tee "$OUT/mcp-conformance.txt"

echo "== conformance report (json) =="
"$OUT/keyway" probe --harness --report conformance-json \
  --issuer-url http://localhost:8099 --issuer-key "$KEY" \
  --allow localhost --allow 127.0.0.1 \
  --consumers "$V/consumers/consumers-mcp.json" 2>>"$OUT/mcp-harness.err" >"$OUT/mcp-conformance.json"
echo "wrote $OUT/mcp-conformance.{txt,json}"
