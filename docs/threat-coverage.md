# JWT/JWKS/OIDC threat coverage

> Auto-generated from the threat taxonomy in `internal/threats`. This is the
> **denominator**: coverage is measured against the documented universe of JWT
> verifier threats (RFC 8725, OWASP, CVEs, CWE, PortSwigger), not against a
> corpus we wrote. Every gap below is a named, cited threat Keyway does not yet
> detect — the roadmap, kept honest.

**Coverage: 21 of 35 documented threats (60%).** 14 gaps remain.

## Coverage by category

| Category | Covered | Total |
|---|---|---|
| signature | 3 | 3 |
| algorithm | 3 | 4 |
| header_key_injection | 2 | 6 |
| claims_validation | 7 | 10 |
| authorization | 2 | 3 |
| key_management | 2 | 4 |
| jwks | 0 | 2 |
| encoding_parsing | 2 | 3 |

## Gaps (no detection yet)

| ID | Severity | Threat | Invariant | Sources |
|---|---|---|---|---|
| HDR-01 | critical | jku header → attacker-controlled JWKS | Key material MUST come from pre-configured trust, never from a URL inside the token being verified. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [JWT attacks: jku header injection](https://portswigger.net/web-security/jwt); [CWE-290 Authentication Bypass by Spoofing](https://cwe.mitre.org/data/definitions/290.html) |
| HDR-02 | critical | x5u header → attacker-controlled certificate | The verifier MUST ignore token-supplied x5u URLs. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [JWT attacks: x5u header injection](https://portswigger.net/web-security/jwt) |
| ALG-02 | high | algorithm downgrade / unpinned alg | The verifier MUST accept only an explicit allowlist of algorithms, not whatever the token's header requests. | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CVE-2022-23540](https://nvd.nist.gov/vuln/detail/CVE-2022-23540) |
| CLM-10 | high | cross-type token substitution | Different JWT kinds MUST have mutually exclusive validation rules (distinct aud/typ/issuer scoping). | [§3.12 Use Mutually Exclusive Validation Rules for Different Kinds of JWTs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.12) |
| HDR-04 | high | x5c embedded certificate chain | An embedded x5c chain MUST be validated against a pre-configured trust anchor, never trusted on its own. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4) |
| HDR-06 | high | kid SQL / command injection | kid MUST NOT be interpolated into SQL/commands; use parameterized lookups. | [JWT attacks: kid injection](https://portswigger.net/web-security/jwt); [CWE-89 SQL Injection](https://cwe.mitre.org/data/definitions/89.html) |
| JWKS-01 | high | JWKS fetched over plaintext HTTP | JWKS/OIDC metadata MUST be fetched over TLS. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [CWE-319 Cleartext Transmission of Sensitive Information](https://cwe.mitre.org/data/definitions/319.html) |
| KEY-04 | high | weak/guessable HMAC secret | Symmetric secrets MUST have sufficient entropy; short/dictionary secrets MUST be treated as compromised. | [§3.5 Ensure Cryptographic Keys Have Sufficient Entropy](https://datatracker.ietf.org/doc/html/rfc8725#section-3.5); [CWE-521 Weak Password Requirements](https://cwe.mitre.org/data/definitions/521.html) |
| AUTHZ-03 | medium | authorized party (azp/client_id) not checked | Where policy scopes a token to a client, azp/client_id MUST be validated. | [API2:2023 Broken Authentication](https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/) |
| CLM-07 | medium | claim type confusion | Claim types MUST be validated; a mistyped claim MUST be a rejection, not a silent skip. | [§3.10 Do Not Trust Received Claims](https://datatracker.ietf.org/doc/html/rfc8725#section-3.10) |
| CLM-09 | medium | explicit typing not enforced (typ/cty) | Where multiple JWT types exist, typ SHOULD be validated (explicit typing). | [§3.11 Use Explicit Typing](https://datatracker.ietf.org/doc/html/rfc8725#section-3.11) |
| ENC-03 | medium | JWE decompression bomb | Compressed token payloads MUST be size-bounded, or compression disabled. | [§3.6 Avoid Compression of Encrypted Data](https://datatracker.ietf.org/doc/html/rfc8725#section-3.6) |
| JWKS-02 | medium | issuer discovery follows redirects to internal hosts | Discovery/JWKS fetches MUST NOT follow redirects to untrusted hosts. | [CWE-918 Server-Side Request Forgery (SSRF)](https://cwe.mitre.org/data/definitions/918.html) |
| KEY-03 | medium | weak asymmetric key accepted (RSA < 2048) | Signing keys MUST meet minimum strength (e.g. RSA ≥ 2048). | [§3.5 Ensure Cryptographic Keys Have Sufficient Entropy](https://datatracker.ietf.org/doc/html/rfc8725#section-3.5); [CWE-326 Inadequate Encryption Strength](https://cwe.mitre.org/data/definitions/326.html) |

## Covered

| ID | Severity | Threat | Detector | Sources |
|---|---|---|---|---|
| ALG-01 | critical | RS/HS key confusion | `probe:alg_confusion`, `harness:alg_confusion_rs_hs` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CVE-2016-5431](https://nvd.nist.gov/vuln/detail/CVE-2016-5431); [CVE-2022-23541](https://nvd.nist.gov/vuln/detail/CVE-2022-23541) |
| ALG-04 | critical | invalid ECDSA signature (0,0) — "psychic signature" | `harness:psychic_signature_es256` | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [CVE-2022-21449](https://nvd.nist.gov/vuln/detail/CVE-2022-21449) |
| AUTHZ-02 | critical | identity header trusted without a token | `probe:header_bypass` | [CWE-290 Authentication Bypass by Spoofing](https://cwe.mitre.org/data/definitions/290.html) |
| HDR-03 | critical | embedded jwk self-signed token | `harness:embedded_jwk` | [CVE-2018-0114](https://nvd.nist.gov/vuln/detail/CVE-2018-0114); [JWT attacks: embedded jwk](https://portswigger.net/web-security/jwt); [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html) |
| SIG-01 | critical | none algorithm accepted | `probe:alg_none`, `harness:alg_none` | [§3.2 Use Appropriate Algorithms](https://datatracker.ietf.org/doc/html/rfc8725#section-3.2); [CVE-2015-9235](https://nvd.nist.gov/vuln/detail/CVE-2015-9235); [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html) |
| SIG-02 | critical | signature not verified | `probe:tampered_signature`, `harness:tampered_signature` | [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html); [CWE-345 Insufficient Verification of Data Authenticity](https://cwe.mitre.org/data/definitions/345.html) |
| SIG-03 | critical | empty signature with a signing alg | `harness:empty_signature_rs256` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html) |
| ALG-03 | high | alg header case/whitespace variant | `harness:alg_none_variants` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [JWT attacks: none variants](https://portswigger.net/web-security/jwt) |
| AUTHZ-01 | high | sibling-audience token accepted | `probe:sibling_client_token` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9) |
| CLM-01 | high | expired token accepted (exp not validated) | `probe:expired`, `harness:expired` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9); [JWT Cheat Sheet: token expiration](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html) |
| CLM-03 | high | issuer not validated | `probe:wrong_issuer`, `harness:wrong_issuer` | [§3.8 Validate Issuer and Subject](https://datatracker.ietf.org/doc/html/rfc8725#section-3.8) |
| CLM-04 | high | audience not validated | `probe:wrong_audience`, `harness:wrong_audience` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9) |
| CLM-05 | high | required authorization claim not enforced | `probe:missing_required_claim` | [§3.10 Do Not Trust Received Claims](https://datatracker.ietf.org/doc/html/rfc8725#section-3.10) |
| HDR-05 | high | kid path traversal | `harness:kid_path_traversal` | [JWT attacks: kid path traversal](https://portswigger.net/web-security/jwt); [CWE-22 Improper Limitation of a Pathname (Path Traversal)](https://cwe.mitre.org/data/definitions/22.html) |
| KEY-01 | high | retired key still accepted | `probe:retired_key` | [§3.5 Ensure Cryptographic Keys Have Sufficient Entropy](https://datatracker.ietf.org/doc/html/rfc8725#section-3.5) |
| KEY-02 | high | unknown-kid rotation outage | `libdefaults:refreshes_on_unknown_kid`, `blast:rotate_key`, `probe:canary_key` | [JWKS rotation guidance](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html) |
| CLM-02 | medium | not-yet-valid token accepted (nbf ignored) | `probe:not_yet_valid`, `harness:not_yet_valid` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9) |
| CLM-06 | medium | aud array/string type confusion | `harness:aud_array_confusion` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9); [API2:2023 Broken Authentication](https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/) |
| CLM-08 | medium | token with no expiry accepted | `harness:missing_exp` | [JWT Cheat Sheet: token expiration](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html) |
| ENC-01 | medium | malformed structure / extra segments accepted | `harness:extra_segments` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CWE-20 Improper Input Validation](https://cwe.mitre.org/data/definitions/20.html) |
| ENC-02 | low | lax base64 / non-canonical encoding | `harness:non_base64url` | [§3.7 Use UTF-8](https://datatracker.ietf.org/doc/html/rfc8725#section-3.7) |

---

_Regenerate with `keyway threats coverage`._
