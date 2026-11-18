# Empirical evaluation plan — firing adversarial agent tokens at real verifiers

**Purpose.** Close the one gap every reviewer (SSR, WAITI #20, AIDC #43) flagged: we
describe a generator + analyzer but report *no results against real systems*. The
reviewers' exact ask — "fire the generated tokens at real verifiers (reference MCP
servers, OAuth middleware, hosted MCP servers in staging) and report which accept
them" — is runnable **today** with code already in this repo. This plan says precisely
how, with a target set, metrics, safety rules, and how the results fold into SACMAT
**#35** before its Oct 31 edit deadline.

Target completion: **Oct 30** (one day of slack before the deadline).

---

## 1. What this measures, and why it answers the reviews

Two things, from one run:

1. **Prevalence of the statically-checkable weaknesses in deployed verifiers.** For
   each threat the SoK calls *token-static* (T) or *server-dynamic* (S), we mint the
   adversarial token a defender would and fire it at a real verifier configured to be
   correct. If the verifier returns 2xx, it *accepted* a token it should have rejected
   — a live vulnerability. The fraction of verifiers that accept each attack is the
   number the `\gate{}` future-work currently promises.

2. **It breaks the "circular loop" objection (AIDC 43B).** The reviewer's point: our
   generator and analyzer are written from the same spec, so the analyzer rejecting the
   generator's tokens proves nothing. The fix is to fire the generated tokens at
   *independent third-party verifiers* and report acceptance. That is exactly what the
   harness does; the verifiers are other people's code.

This also validates the SoK's three-way spectrum empirically: the harness **is** the
conformance suite for mode **S** (it fires minted tokens at a candidate server), and
the analyzer covers mode **T**. The 3 human/model threats (H) stay out of scope, as the
paper claims.

---

## 2. What already exists (no new engine needed)

| Piece | Location | Role |
|---|---|---|
| Adversarial token generator | `internal/attack/generators.go` → `Corpus(GenContext)` | Mints the full attack corpus bound to a target's expected issuer/audience/required-claims, incl. the agent-auth audience-binding tokens (RFC 8707/9728) and the JWT family (alg confusion, `none`, tampered sig, `kid`/`jku`/`x5u`, encoding). |
| Firing + scoring harness | `internal/probe/harness.go` → `RunHarness` | Sends each token as `Authorization: Bearer`, reads the status, scores `Passed` = verifier did the correct thing (rejected attack / accepted control). |
| CLI | `keyway probe --harness` (`internal/cli/probe.go`) | Runs the harness against discovered/declared consumers. |
| Safety | `internal/probe` engine config | Default-deny host allowlist (`--allow`), production guard (`--i-know-this-is-production` required to leave staging), `--dry-run`, inter-probe delay, kill switch, token scrubbing in logs. |
| Synthetic issuer | `internal/issuer/generic` (`--issuer-url`) | Keyway-operated signing key + JWKS; no real user tokens are ever used. |

**So the only missing inputs are the targets and the run.**

---

## 3. The target set (the new work)

A verifier is a `model.Consumer`: an endpoint (`URL`, `SafeProbePath`) plus `Expects`
(issuer, audience(s), required claims). Each target is stood up in Docker on localhost,
configured to **trust the Keyway test issuer and require `aud=rs-A` + `exp`** (the
"correct" config — it *should* reject every attack token). Targets come from discovery
where a declarative config exists (Envoy/Istio are already parsed by
`internal/discovery`), else a hand-authored consumer manifest.

Aim for **8–12 verifiers across three classes** so the result generalizes:

**A. Proxy / gateway validators** (config-driven; discovery already supports the first two)
- Envoy `jwt_authn` filter — the reference proxy validator (`testdata/discovery/reference/envoy.yaml` is a starting config).
- Istio `RequestAuthentication` (runs Envoy jwt_authn under the hood).
- Kong JWT plugin.
- APISIX `jwt-auth` plugin.
- `oauth2-proxy`.

**B. Resource-server middleware** (one per major ecosystem; this is where defaults bite)
- Node: Auth0 `express-oauth2-jwt-bearer`.
- Python: FastAPI + `Authlib` (and a second run with `python-jose`).
- Java: Spring Security Resource Server.
- Go: `golang-jwt/jwt` + `MicahParks/keyfunc`.
- (Optional) PyJWT direct.

**C. MCP servers** (the agent-specific point the paper is about)
- Official MCP **TypeScript** SDK reference server with the spec's OAuth resource-server auth.
- Official MCP **Python** SDK (`mcp`) reference server.
- `FastMCP` with auth.
- (Optional, permission required) a hosted MCP server we operate in staging.

The audience-binding tokens (MCP-01/02) are the headline: the MCP authorization spec
*requires* a resource server to reject a token whose `aud` is not itself and to refuse
passthrough. Any target that returns 2xx for a wrong-`aud` token is a live passthrough
vulnerability — the paper's #1 threat, now measured.

---

## 4. Procedure

```bash
# 0. Build the CLI
make build            # or: go build -o keyway ./cmd/keyway

# 1. Stand up the staging targets + the Keyway test issuer
docker compose -f bench/verifiers/docker-compose.verifiers.yml up -d

# 2. (Where a declarative config exists) discover targets from it,
#    else use the hand-authored consumer manifest in bench/verifiers/consumers/.
#    Envoy/Istio parse via --path; middleware/MCP use the manifest.

# 3. Dry run first — build every token, send nothing, eyeball the corpus
keyway probe --harness --dry-run --path bench/verifiers/consumers/

# 4. The real run, staging-allowlisted (default-deny; localhost only)
keyway probe --harness \
  --allow localhost --allow 127.0.0.1 \
  --issuer-url http://localhost:8099 \
  --path bench/verifiers/consumers/ \
  > bench/verifiers/out/results.txt
```

The harness, per target: mints the corpus bound to that target's expected
issuer/audience/claims, fires each token, records status → **accept (2xx)** or
**reject**, and sets `Passed` (correct behavior). The **control** token (validly
signed, correct `aud`/claims) must be *accepted* — that confirms the target trusts our
issuer, so an attack rejection is meaningful and not just "unknown issuer."

### 4a. The one real setup task: issuer key wiring

For the claim-level attacks (aud/scope/exp/act) to be meaningful, each target must
**trust the Keyway test issuer** — i.e. validate signatures against its JWKS. So the
test issuer must serve a **stable** JWKS at the `--issuer-url`
(`.../.well-known/jwks.json`) whose private key the harness mints with. Today
`keyway probe --issuer-url X` generates an *ephemeral* key and nothing serves its JWKS
at `X`, so a real target would reject even the control (unknown key) and results would
be vacuous. Wire it one of two ways (small, D1):

1. **Committed test keypair + static JWKS (recommended).** Put a throwaway RSA keypair
   under `bench/verifiers/issuer/` (clearly test-only, never a real key), serve its
   public JWKS statically (the `keyway-issuer` nginx service in the compose), and add a
   `--issuer-key <pem>` flag so the harness mints with that same key. Then control and
   attack signatures validate against the served JWKS.
2. **Share a keystore** between a running `keyway serve` issuer and the probe run.

This is the only genuinely new plumbing; once it is in, control tokens pass and the run
is live. Self-contained forgeries (alg/none/sig/kid/jku) need no issuer trust and work
regardless.

---

## 5. What we report (maps straight into the SoK)

1. **Per-threat acceptance across verifiers** — the headline table. E.g. "of N
   verifiers, k accepted a token minted for a different resource (MCP-01/02 token
   passthrough); j accepted an over-broad scope (SCOPE-01); i accepted a token with no
   `exp` (SCOPE-02)." This is the prevalence the paper owes.
2. **Per-target matrix** — which threats each verifier mishandles (appendix table),
   grouped by class (proxy / middleware / MCP) so defaults-by-ecosystem are visible.
3. **Control sanity** — every target accepted the control (else its rejections are
   vacuous); report and exclude any target whose control failed.
4. **Analyzer false-positive / precision** — run `internal/agentauth.Analyze` on the
   control and a set of valid tokens; confirm zero spurious findings. Answers "no
   false-positive analysis" (AIDC 43B) and the precision question.
5. **Threat-model honesty** — the corpus distinguishes *self-contained* forgeries
   (alg/none/sig/kid — need no issuer key) from *validly-signed, bad-claim* tokens
   (aud/scope/exp/act — the defender mints these with issuer control). Report results
   under each, which directly answers 43B's "state the attacker capability per threat":
   the claim-level attacks are a **defender** red-team (issuer control), not something
   the injection attacker can forge.

**Headline the paper can then make (illustrative):** "Across N reference verifiers, X of
N accepted a token minted for a different resource — token passthrough is not
hypothetical but the default in M of the P ecosystems we tested." Replace X/N/M/P with
the measured values.

---

## 6. Ethics and safety (state this in the paper's ethics section)

- **Staging only.** The engine is default-deny on hosts; only `--allow`-listed
  localhost/staging hosts are probed, and leaving staging needs an explicit
  `--i-know-this-is-production` flag we do not use.
- **Self-hosted or owned targets.** Targets are reference implementations we stand up
  ourselves, or servers we operate. **No third-party hosted server is probed without
  prior permission.** Any hosted MCP server is opt-in and coordinated.
- **Synthetic tokens only.** Every token is minted by a Keyway-operated test issuer; no
  real user tokens, no real secrets.
- **Rate-limited and reversible.** Inter-probe delay, kill switch, response-body token
  scrubbing in logs; requests are GETs to a safe probe path, no mutation.
- **Coordinated disclosure.** Any identifiable high-severity finding in a named
  open-source verifier follows coordinated disclosure before the camera-ready names it;
  aggregate prevalence is reported without shaming a single project.

---

## 7. How the results fold into SACMAT #35 (before Oct 30)

1. Replace the `\gate{...}` prevalence note in *Threats to validity* and the Open
   Science artifact note with a real **Evaluation** subsection + the §5 tables.
2. Add the analyzer **false-positive/precision** result.
3. Update the abstract/intro from "we systematize …" to "we systematize **and
   evaluate** …": "we fire the generated tokens at N reference verifiers and find …".
   This is what moves the paper from "taxonomy + linter" (both reviewers' phrase) to a
   measured contribution.
4. Re-upload the PDF to SACMAT **#35** (editable until Oct 31) and bump the artifact
   (`bench/verifiers/` + results JSON) — which also answers the "no artifact for
   reviewers" ask.

---

## 8. Timeline to Oct 30

| Day | Work |
|---|---|
| **D1** | Stand up wave 1 (Envoy + Istio + 2–3 middleware) in `bench/verifiers/`; wire consumers; smoke-test `keyway probe --harness --dry-run`; first real run. |
| **D2** | Add MCP SDK reference servers + remaining middleware; full run; collect `results.json`; control-sanity + false-positive analysis. |
| **D3 (≤ Oct 30)** | Write the Evaluation section + tables; re-upload to #35; publish the `bench/verifiers/` artifact. |

**Risk valve:** if a target refuses to stand up in time, the study still stands with
the targets that did — even "5 targets is more evidence than the current zero" (AIDC
43B). The number of verifiers is the only thing that scales with effort; the harness,
scoring, and write-up are fixed.

---

## 9. Open decisions for Archit

- **Target count:** 8–12 is comfortable; the floor that satisfies the reviewers is ~5.
- **Hosted MCP servers:** include any you operate in staging? (Needs your go-ahead +
  the endpoint/expected-audience; everything else is self-hosted.)
- **Artifact visibility:** publish `bench/verifiers/` + `results.json` as an anonymous
  artifact for reviewers now (strengthens the submission), or at camera-ready only?
