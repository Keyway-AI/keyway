# bench/verifiers — firing adversarial tokens at real verifiers

This directory holds the **empirical evaluation** for the agent-authorization SoK: a
reproducible set of real verifiers (MCP servers, OAuth/JWT middleware, proxies) that we
fire the adversarial token corpus at, recording which **accept** a token they should
reject. Full methodology, target list, metrics, ethics, and timeline:
[`docs/research/empirical-eval-plan.md`](../../docs/research/empirical-eval-plan.md).

The engine is already in the repo — `internal/attack.Corpus` (generator) +
`keyway probe --harness` (fires + scores, with a default-deny staging allowlist). The
only inputs are the **targets** and the **run**.

## The idea in one line

Stand up each verifier **configured to be correct** (trusts the Keyway test issuer,
requires `aud=rs-A` + `exp`). It should reject every attack token. If it returns `2xx`,
it accepted a token it should have rejected — a live vulnerability. Aggregate across
targets = the prevalence the paper owes.

## Run it

```bash
# 1. Build
make build                       # -> ./keyway

# 2. Stand up the Keyway test issuer + the staging verifiers
docker compose -f bench/verifiers/docker-compose.verifiers.yml up -d

# 3. Point the harness at the targets (copy the template, fill real ports)
cp bench/verifiers/consumers/consumers.example.json bench/verifiers/consumers/consumers.json

# 4. Dry run (builds every token, sends nothing)
./keyway probe --harness --dry-run --consumers bench/verifiers/consumers/consumers.json

# 5. Real run — staging allowlist only (default-deny), synthetic issuer
mkdir -p bench/verifiers/out
./keyway probe --harness \
  --allow localhost --allow 127.0.0.1 \
  --issuer-url http://localhost:8099 \
  --consumers bench/verifiers/consumers/consumers.json \
  | tee bench/verifiers/out/results.txt
```

Each row is `CONSUMER  PROBE(threat)  STATUS  RESULT`. `RESULT=FAIL` means the verifier
did the **wrong** thing — it accepted an attack (or rejected the control). Those are the
findings. The `CONTROL` token must `pass` (be accepted) for a target's results to count;
if it fails, the target does not trust our issuer and its rejections are vacuous.

## Target classes (fill out to 8–12; floor is ~5)

| Class | Targets | Discovery |
|---|---|---|
| Proxy / gateway | Envoy `jwt_authn`, Istio `RequestAuthentication`, Kong, APISIX, oauth2-proxy | Envoy/Istio parse via `--path`; others via the consumer manifest |
| Resource-server middleware | Node `express-oauth2-jwt-bearer`, Python FastAPI+Authlib / python-jose, Spring Security RS, Go `golang-jwt`+`keyfunc` | consumer manifest |
| MCP servers | MCP TypeScript SDK, MCP Python SDK, FastMCP | consumer manifest |

Add a target by: (1) a service in `docker-compose.verifiers.yml` configured to trust
`http://localhost:8099` (the Keyway test issuer JWKS) and require `aud=rs-A`; (2) an
entry in `consumers.json` with its endpoint + `expects`.

## What the corpus fires (maps to the SoK threats)

- **Audience binding / passthrough** (MCP-01/02) — validly signed, `aud` = another
  resource or absent. A correct MCP resource server MUST reject (RFC 8707/9728).
- **Scope / expiry** (SCOPE-01/02) — over-broad scope; missing/expired `exp`.
- **Delegation hygiene** (DEL-01/02) — missing / malformed `act` chain.
- **JWT family** (self-contained forgeries, no issuer key needed) — `alg` RS/HS
  confusion, `alg:none` variants, psychic ES256 signature, tampered signature, `kid`
  path traversal, `jku`/`x5u` key injection, structural/encoding corruption.

The first three are *validly-signed, bad-claim* tokens the **defender** mints with
issuer control (a red-team capability, not what an injection attacker can forge); the
JWT family are self-contained forgeries. Report results under both — it is exactly the
attacker-capability distinction the SoK's threat model draws.

## Safety (enforced by the engine)

- Default-deny host allowlist (`--allow`); leaving staging needs
  `--i-know-this-is-production` (never used here).
- Synthetic tokens only, minted by the Keyway test issuer. No real user tokens.
- Inter-probe delay, kill switch, response-body token scrubbing.
- No third-party hosted server without prior permission; coordinated disclosure for any
  identifiable high-severity finding.

## Output

`out/results.txt` (human) and, with a DB configured, persisted `ProbeResult`s. Turn
these into the per-threat acceptance table and the per-target matrix for the paper's
Evaluation section (see the plan, §5).
