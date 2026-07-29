# Threat coverage

> Auto-generated from the threat taxonomy in `internal/threats`. This is the
> **denominator**: coverage is measured against the documented universe of auth
> verifier threats (RFC 8725, the MCP spec, OAuth RFCs, OWASP LLM/Agentic Top
> 10s, CVEs, CWE, PortSwigger), not against a corpus we wrote. Every gap below is
> a named, cited threat Keyway does not yet detect — the roadmap, kept honest.
> The taxonomy spans two domains: **jwt** (mature) and **agent** (a new frontier).

**Coverage: 27 of 50 documented threats (54%).** 23 gaps remain.

## Coverage by domain

| Domain | Covered | Total | % |
|---|---|---|---|
| jwt | 21 | 35 | 60% |
| agent | 6 | 15 | 40% |

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
| token_binding | 2 | 2 |
| consent | 0 | 2 |
| delegation | 2 | 4 |
| scope | 2 | 3 |
| agent_identity | 0 | 2 |
| agency | 0 | 2 |

## Gaps (no detection yet)

| ID | Domain | Severity | Threat | Invariant | Sources |
|---|---|---|---|---|---|
| AGT-01 | agent | critical | prompt-injection-driven privilege escalation | High-impact actions MUST require per-action authorization or human-in-the-loop; tools MUST be least-privilege so injection cannot escalate. | [OWASP LLM01 Prompt Injection (2025)](https://owasp.org/www-project-top-10-for-large-language-model-applications/); [OWASP Agentic Top 10 (2025)](https://genai.owasp.org/2025/12/09/owasp-genai-security-project-releases-top-10-risks-and-mitigations-for-agentic-ai-security/) |
| CD-01 | agent | critical | confused deputy via static client ID | An OAuth proxy MUST obtain per-client user consent, enforce exact redirect_uri matching, and validate `state`; it MUST NOT let a static client ID + prior consent bypass approval. | [MCP spec: confused deputy](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices); [CSA: AI agent confused deputy](https://labs.cloudsecurityalliance.org/research/csa-research-note-ai-agent-confused-deputy-prompt-injection/); [CWE-441 Unintended Proxy or Intermediary (Confused Deputy)](https://cwe.mitre.org/data/definitions/441.html) |
| HDR-01 | jwt | critical | jku header → attacker-controlled JWKS | Key material MUST come from pre-configured trust, never from a URL inside the token being verified. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [JWT attacks: jku header injection](https://portswigger.net/web-security/jwt); [CWE-290 Authentication Bypass by Spoofing](https://cwe.mitre.org/data/definitions/290.html) |
| HDR-02 | jwt | critical | x5u header → attacker-controlled certificate | The verifier MUST ignore token-supplied x5u URLs. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [JWT attacks: x5u header injection](https://portswigger.net/web-security/jwt) |
| AGT-02 | agent | high | tool poisoning / rug-pull | A tool's definition/behavior change MUST invalidate prior approval and require re-consent; tool definitions SHOULD be integrity-pinned. | [CVE-2025-54136](https://nvd.nist.gov/vuln/detail/CVE-2025-54136); [MCP tool poisoning / rug-pull](https://www.truefoundry.com/blog/blog-mcp-tool-poisoning-gateway-defense) |
| AID-01 | agent | high | session used as authentication | Servers MUST NOT use sessions for authentication; session IDs MUST be non-deterministic and bound to user identity. | [MCP spec: session hijacking](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices); [CWE-384 Session Fixation](https://cwe.mitre.org/data/definitions/384.html) |
| ALG-02 | jwt | high | algorithm downgrade / unpinned alg | The verifier MUST accept only an explicit allowlist of algorithms, not whatever the token's header requests. | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CVE-2022-23540](https://nvd.nist.gov/vuln/detail/CVE-2022-23540) |
| CD-02 | agent | high | dynamic client registration abuse | DCR MUST validate and constrain client metadata (redirect_uris, grant types); prefer Client-ID Metadata Documents over open RFC 7591 registration. | [RFC 7591 OAuth 2.0 Dynamic Client Registration](https://datatracker.ietf.org/doc/html/rfc7591); [MCP spec: client registration](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices); [OAuth Client ID Metadata Documents](https://datatracker.ietf.org/doc/draft-ietf-oauth-client-id-metadata-document/) |
| CLM-10 | jwt | high | cross-type token substitution | Different JWT kinds MUST have mutually exclusive validation rules (distinct aud/typ/issuer scoping). | [§3.12 Use Mutually Exclusive Validation Rules for Different Kinds of JWTs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.12) |
| DEL-03 | agent | high | may_act not enforced on token exchange | Token exchange MUST honor `may_act`: only pre-authorized actors may obtain a delegated token for a subject. | [RFC 8693 OAuth 2.0 Token Exchange (may_act)](https://datatracker.ietf.org/doc/html/rfc8693) |
| HDR-04 | jwt | high | x5c embedded certificate chain | An embedded x5c chain MUST be validated against a pre-configured trust anchor, never trusted on its own. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4) |
| HDR-06 | jwt | high | kid SQL / command injection | kid MUST NOT be interpolated into SQL/commands; use parameterized lookups. | [JWT attacks: kid injection](https://portswigger.net/web-security/jwt); [CWE-89 SQL Injection](https://cwe.mitre.org/data/definitions/89.html) |
| JWKS-01 | jwt | high | JWKS fetched over plaintext HTTP | JWKS/OIDC metadata MUST be fetched over TLS. | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [CWE-319 Cleartext Transmission of Sensitive Information](https://cwe.mitre.org/data/definitions/319.html) |
| KEY-04 | jwt | high | weak/guessable HMAC secret | Symmetric secrets MUST have sufficient entropy; short/dictionary secrets MUST be treated as compromised. | [§3.5 Ensure Cryptographic Keys Have Sufficient Entropy](https://datatracker.ietf.org/doc/html/rfc8725#section-3.5); [CWE-521 Weak Password Requirements](https://cwe.mitre.org/data/definitions/521.html) |
| SCOPE-03 | agent | high | agent reuses the human's root credential | An agent MUST act with its own scoped identity and delegated (act) token, never the user's raw credential. | [Auth0: Auth for AI Agents (Token Vault)](https://auth0.com/blog/auth0-for-ai-agents-generally-available/); [MCP spec: authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices) |
| AID-02 | agent | medium | no distinct workload/agent identity | Each agent/workload MUST have a verifiable identity credential distinct from the human and from other agents. | [IETF WIMSE — Workload Identity](https://datatracker.ietf.org/doc/draft-ietf-wimse-workload-identity-practices/); [SPIFFE](https://spiffe.io/) |
| AUTHZ-03 | jwt | medium | authorized party (azp/client_id) not checked | Where policy scopes a token to a client, azp/client_id MUST be validated. | [API2:2023 Broken Authentication](https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/) |
| CLM-07 | jwt | medium | claim type confusion | Claim types MUST be validated; a mistyped claim MUST be a rejection, not a silent skip. | [§3.10 Do Not Trust Received Claims](https://datatracker.ietf.org/doc/html/rfc8725#section-3.10) |
| CLM-09 | jwt | medium | explicit typing not enforced (typ/cty) | Where multiple JWT types exist, typ SHOULD be validated (explicit typing). | [§3.11 Use Explicit Typing](https://datatracker.ietf.org/doc/html/rfc8725#section-3.11) |
| DEL-04 | agent | medium | impersonation where delegation is required | Where policy requires delegation, impersonation tokens MUST be rejected; the actor MUST remain attributable in the token and audit log. | [RFC 8693 OAuth 2.0 Token Exchange (delegation vs impersonation)](https://datatracker.ietf.org/doc/html/rfc8693); [OpenID: Identity Management for Agentic AI](https://openid.net/wp-content/uploads/2025/10/Identity-Management-for-Agentic-AI.pdf) |
| ENC-03 | jwt | medium | JWE decompression bomb | Compressed token payloads MUST be size-bounded, or compression disabled. | [§3.6 Avoid Compression of Encrypted Data](https://datatracker.ietf.org/doc/html/rfc8725#section-3.6) |
| JWKS-02 | jwt | medium | issuer discovery follows redirects to internal hosts | Discovery/JWKS fetches MUST NOT follow redirects to untrusted hosts. | [CWE-918 Server-Side Request Forgery (SSRF)](https://cwe.mitre.org/data/definitions/918.html) |
| KEY-03 | jwt | medium | weak asymmetric key accepted (RSA < 2048) | Signing keys MUST meet minimum strength (e.g. RSA ≥ 2048). | [§3.5 Ensure Cryptographic Keys Have Sufficient Entropy](https://datatracker.ietf.org/doc/html/rfc8725#section-3.5); [CWE-326 Inadequate Encryption Strength](https://cwe.mitre.org/data/definitions/326.html) |

## Covered

| ID | Domain | Severity | Threat | Detector | Sources |
|---|---|---|---|---|---|
| ALG-01 | jwt | critical | RS/HS key confusion | `probe:alg_confusion`, `harness:alg_confusion_rs_hs` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CVE-2016-5431](https://nvd.nist.gov/vuln/detail/CVE-2016-5431); [CVE-2022-23541](https://nvd.nist.gov/vuln/detail/CVE-2022-23541) |
| ALG-04 | jwt | critical | invalid ECDSA signature (0,0) — "psychic signature" | `harness:psychic_signature_es256` | [§3.4 Validate Cryptographic Inputs](https://datatracker.ietf.org/doc/html/rfc8725#section-3.4); [CVE-2022-21449](https://nvd.nist.gov/vuln/detail/CVE-2022-21449) |
| AUTHZ-02 | jwt | critical | identity header trusted without a token | `probe:header_bypass` | [CWE-290 Authentication Bypass by Spoofing](https://cwe.mitre.org/data/definitions/290.html) |
| HDR-03 | jwt | critical | embedded jwk self-signed token | `harness:embedded_jwk` | [CVE-2018-0114](https://nvd.nist.gov/vuln/detail/CVE-2018-0114); [JWT attacks: embedded jwk](https://portswigger.net/web-security/jwt); [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html) |
| MCP-01 | agent | critical | token passthrough (wrong-audience token accepted) | `analyzer:aud_mismatch`, `harness:resource_passthrough` | [MCP spec: token passthrough anti-pattern](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices); [RFC 8707 Resource Indicators for OAuth 2.0](https://datatracker.ietf.org/doc/html/rfc8707); [CWE-287 Improper Authentication](https://cwe.mitre.org/data/definitions/287.html) |
| SIG-01 | jwt | critical | none algorithm accepted | `probe:alg_none`, `harness:alg_none` | [§3.2 Use Appropriate Algorithms](https://datatracker.ietf.org/doc/html/rfc8725#section-3.2); [CVE-2015-9235](https://nvd.nist.gov/vuln/detail/CVE-2015-9235); [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html) |
| SIG-02 | jwt | critical | signature not verified | `probe:tampered_signature`, `harness:tampered_signature` | [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html); [CWE-345 Insufficient Verification of Data Authenticity](https://cwe.mitre.org/data/definitions/345.html) |
| SIG-03 | jwt | critical | empty signature with a signing alg | `harness:empty_signature_rs256` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CWE-347 Improper Verification of Cryptographic Signature](https://cwe.mitre.org/data/definitions/347.html) |
| ALG-03 | jwt | high | alg header case/whitespace variant | `harness:alg_none_variants` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [JWT attacks: none variants](https://portswigger.net/web-security/jwt) |
| AUTHZ-01 | jwt | high | sibling-audience token accepted | `probe:sibling_client_token` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9) |
| CLM-01 | jwt | high | expired token accepted (exp not validated) | `probe:expired`, `harness:expired` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9); [JWT Cheat Sheet: token expiration](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html) |
| CLM-03 | jwt | high | issuer not validated | `probe:wrong_issuer`, `harness:wrong_issuer` | [§3.8 Validate Issuer and Subject](https://datatracker.ietf.org/doc/html/rfc8725#section-3.8) |
| CLM-04 | jwt | high | audience not validated | `probe:wrong_audience`, `harness:wrong_audience` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9) |
| CLM-05 | jwt | high | required authorization claim not enforced | `probe:missing_required_claim` | [§3.10 Do Not Trust Received Claims](https://datatracker.ietf.org/doc/html/rfc8725#section-3.10) |
| DEL-01 | agent | high | missing or forged act (delegation) claim | `analyzer:missing_act` | [RFC 8693 OAuth 2.0 Token Exchange (act/may_act)](https://datatracker.ietf.org/doc/html/rfc8693); [On-Behalf-Of User for AI Agents](https://www.ietf.org/archive/id/draft-oauth-ai-agents-on-behalf-of-user-00.html) |
| DEL-02 | agent | high | delegation-chain / transitive-trust abuse | `harness:sibling_resource` | [RFC 8693 OAuth 2.0 Token Exchange](https://datatracker.ietf.org/doc/html/rfc8693); [OWASP Agentic Top 10 (2025)](https://genai.owasp.org/2025/12/09/owasp-genai-security-project-releases-top-10-risks-and-mitigations-for-agentic-ai-security/); [RFC 9700 OAuth 2.0 Security BCP](https://datatracker.ietf.org/doc/html/rfc9700) |
| HDR-05 | jwt | high | kid path traversal | `harness:kid_path_traversal` | [JWT attacks: kid path traversal](https://portswigger.net/web-security/jwt); [CWE-22 Improper Limitation of a Pathname (Path Traversal)](https://cwe.mitre.org/data/definitions/22.html) |
| KEY-01 | jwt | high | retired key still accepted | `probe:retired_key` | [§3.5 Ensure Cryptographic Keys Have Sufficient Entropy](https://datatracker.ietf.org/doc/html/rfc8725#section-3.5) |
| KEY-02 | jwt | high | unknown-kid rotation outage | `libdefaults:refreshes_on_unknown_kid`, `blast:rotate_key`, `probe:canary_key` | [JWKS rotation guidance](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html) |
| MCP-02 | agent | high | missing resource indicator / unbound audience | `analyzer:aud_unbound`, `harness:unbound_audience` | [RFC 8707 Resource Indicators for OAuth 2.0](https://datatracker.ietf.org/doc/html/rfc8707); [RFC 9728 OAuth 2.0 Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728); [MCP spec: authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices) |
| SCOPE-01 | agent | high | over-scoped agent credential | `analyzer:over_scope` | [MCP spec: scope minimization](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices); [OWASP LLM06 Excessive Agency (2025)](https://owasp.org/www-project-top-10-for-large-language-model-applications/) |
| SCOPE-02 | agent | high | non-expiring / long-lived agent token | `analyzer:non_expiring` | [2026 State of AI Agent Identity](https://securityboulevard.com/2026/07/the-agent-identity-problem-non-human-identities-outnumber-humans-45-to-1-and-ai-agents-are-making-it-worse/); [CWE-798 Use of Hard-coded Credentials](https://cwe.mitre.org/data/definitions/798.html) |
| CLM-02 | jwt | medium | not-yet-valid token accepted (nbf ignored) | `probe:not_yet_valid`, `harness:not_yet_valid` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9) |
| CLM-06 | jwt | medium | aud array/string type confusion | `harness:aud_array_confusion` | [§3.9 Use and Validate Audience](https://datatracker.ietf.org/doc/html/rfc8725#section-3.9); [API2:2023 Broken Authentication](https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/) |
| CLM-08 | jwt | medium | token with no expiry accepted | `harness:missing_exp` | [JWT Cheat Sheet: token expiration](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html) |
| ENC-01 | jwt | medium | malformed structure / extra segments accepted | `harness:extra_segments` | [§3.1 Perform Algorithm Verification](https://datatracker.ietf.org/doc/html/rfc8725#section-3.1); [CWE-20 Improper Input Validation](https://cwe.mitre.org/data/definitions/20.html) |
| ENC-02 | jwt | low | lax base64 / non-canonical encoding | `harness:non_base64url` | [§3.7 Use UTF-8](https://datatracker.ietf.org/doc/html/rfc8725#section-3.7) |

---

_Regenerate with `make coverage` (`keyway threats coverage`)._
