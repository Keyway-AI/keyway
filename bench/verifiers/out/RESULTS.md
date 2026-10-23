# Empirical eval results — adversarial tokens vs. 8 real verifiers

Run: `bench/verifiers/run-eval.sh` (see `docs/research/empirical-eval-plan.md`). Each
verifier was stood up on localhost, configured to be correct (verify RS256 signature
against the Keyway test-issuer JWKS, require `iss=http://localhost:8099`, `aud=rs-A`,
and require `exp` where the library provides the option). The harness minted the
adversarial corpus bound to each target and fired it; a `2xx` on an attack token is a
finding. **All 8 controls were accepted (200)**, so every target trusted the issuer and
its rejections are meaningful.

## Targets (8 verifiers, 3 ecosystems, 8 distinct libraries)

| # | Target | Library | Ecosystem |
|---|---|---|---|
| 1 | go-golangjwt | golang-jwt/jwt v5 + MicahParks/keyfunc | Go |
| 2 | go-jwx | lestrrat-go/jwx v2 | Go |
| 3 | go-gojose | go-jose/go-jose v4 | Go |
| 4 | node-jsonwebtoken | jsonwebtoken + jwks-rsa | Node |
| 5 | node-jose | jose (panva) | Node |
| 6 | py-pyjwt | PyJWT + PyJWKClient | Python |
| 7 | py-authlib | Authlib | Python |
| 8 | py-jose | python-jose | Python |

(Envoy `jwt_authn` is wired as a 9th proxy target in `envoy/envoy-host.yaml` +
`consumers.json`; include it when the image is available.)

## Findings — 5 of 8 verifiers accepted a token they should have rejected

| Target | Accepted attack | Threat | Meaning |
|---|---|---|---|
| go-jwx | ENC-02 non-base64url | encoding | accepts a structurally malformed token |
| go-gojose | CLM-08 missing `exp` | SCOPE-02 | accepts a token with **no expiry** |
| node-jsonwebtoken | CLM-08 missing `exp` | SCOPE-02 | accepts a token with **no expiry** |
| node-jose | CLM-08 missing `exp` | SCOPE-02 | accepts a token with **no expiry** |
| py-jose | MCP-02 unbound audience | MCP-01/02 | accepts a token with **no bound audience** (token passthrough) |

Clean (rejected every attack): **go-golangjwt, py-pyjwt, py-authlib**.

## Per-threat prevalence (of 8)

| Threat | Accepting | Note |
|---|---|---|
| CLM-08 missing `exp` (SCOPE-02) | **3/8** | go-jose, jsonwebtoken, panva/jose provide no built-in "require exp"; their standard verify accepts a token with no `exp`. A correct resource server must reject it (RFC 8725). |
| MCP-02 unbound audience (MCP-01/02) | **1/8** | python-jose accepts a token with no `aud` even when `audience="rs-A"` is passed — unbound-audience passthrough, the SoK's headline threat. Independently reproduced outside the harness. |
| ENC-02 structural (encoding) | **1/8** | lestrrat-go/jwx accepts a non-base64url-corrupted token. |

Every other adversarial token (alg confusion, `alg:none` variants, psychic ES256,
tampered/empty signature, `kid`/`jku`/`x5u` injection, wrong issuer/audience, expired,
not-yet-valid, aud-array confusion, extra-segment) was rejected by all 8.

## Honest notes (for the paper and reviewers)

- **Config is each library's standard "validate an access token" usage** (signature +
  issuer + audience + RS256). We did not add non-default hardening. `exp` is required
  where the library offers the option (golang-jwt, jwx, PyJWT, python-jose); three
  libraries offer no such option and accept a no-`exp` token — that is the finding.
- The two most security-relevant findings (missing `exp`, unbound `aud`) were
  **independently reproduced** with a standalone script, not just observed through the
  harness.
- These are **defaults/footguns**, not exotic bugs, which is exactly the SoK's point:
  the token-level weaknesses are the common case, not the corner case.
- Raw run: `out/results.txt`. Reproduce: `bench/verifiers/run-eval.sh`.
