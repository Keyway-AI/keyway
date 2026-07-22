# The 13 Probes

Keyway verifies each derived contract by minting synthetic tokens and observing how a
consumer responds. A baseline claim set is used unless a probe overrides it (PRD §6.2):

```json
{
  "iss": "<issuer url>",
  "aud": "<consumer's first audience>",
  "sub": "keyway-synthetic-principal",
  "iat": "<now>", "exp": "<now+5m>", "nbf": "<now>",
  "jti": "<uuid>"
}
```

| # | ID | Mutation | Expect | Needs key | Severity |
|---|---|---|---|---|---|
| 1 | `valid_token` | none — baseline, active key | 2xx | yes | info |
| 2 | `expired` | `exp = now - 3600` | 401 | yes | high |
| 3 | `not_yet_valid` | `nbf = now + 3600` | 401 | yes | medium |
| 4 | `wrong_issuer` | `iss = https://issuer.keyway.invalid` | 401 | yes | high |
| 5 | `wrong_audience` | `aud = keyway-invalid-audience` | 401 | yes | high |
| 6 | `alg_none` | header `{"alg":"none"}`, empty signature | 401 | no | critical |
| 7 | `alg_confusion` | HS256 with the RSA public-key PEM as the HMAC secret | 401 | no | critical |
| 8 | `tampered_signature` | flip the final signature byte | 401 | yes | high |
| 9 | `missing_required_claim` | one sub-probe per required claim, omitted in turn | 401/403 | yes | medium |
| 10 | `retired_key` | sign with a `retired` key | 401 | yes | high |
| 11 | `sibling_client_token` | valid token minted for a *different* consumer's audience | 401 | yes | high |
| 12 | `header_bypass` | no bearer token; only `X-User-Id`/`X-Forwarded-User` headers | 401 | no | critical |
| 13 | `canary_key` | sign with the `announced` (canary) key | 2xx | yes | info |

## Implementation notes

- **Probe 6** builds the compact JWS manually as `base64url(header) + "." + base64url(payload) + "."`
  because most JOSE libraries refuse to emit `alg=none`.
- **Probe 7**'s HMAC secret is the *exact* PEM-encoded public key including header, footer, and
  trailing newline — the classic confusion mismatch. Do not trim.
- **Probe 9** expands to N sub-results (capped at 8 claims, prioritising claims used in
  authorization decisions — PRD OPEN-3). Each is reported individually.
- **Probe 12** sends no bearer token at all; it catches services that trust identity headers.
- **Probe 13** is a no-op (`ErrNoAnnouncedKey`) when no key is in the `announced` state.

## Safety

- Staging only. The engine refuses hosts not on the allowlist unless
  `--i-know-this-is-production` is passed.
- A consumer returning 5xx on the baseline valid-token probe is marked `unverified` and
  skipped — a broken service is not a contract finding.
- Minted tokens are never persisted; only `jti` + probe ID are logged (PRD OPEN-4).
