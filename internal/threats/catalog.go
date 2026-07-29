// Package threats is Keyway's JWT/JWKS/OIDC threat taxonomy — the *denominator*
// against which detection coverage is measured. Each entry is grounded in an
// external, authoritative source (RFC 8725 "JWT Best Current Practices", a CVE, a
// CWE, the OWASP cheat sheets, or the PortSwigger JWT catalog), states the
// verifier invariant a correct implementation must uphold, and records which
// Keyway detector (if any) exercises it. A threat with no detector is an honest,
// named coverage gap — the point of this package is that gaps are visible and
// countable, not hidden by only-testing-what-we-already-check.
package threats

import "github.com/nometria/keyway/internal/model"

// Domain groups threats by the problem space they belong to. The taxonomy spans
// two: the classic JWT/JWKS/OIDC verifier surface, and the newer auth surface of
// AI agents (MCP, on-behalf-of delegation, agent identity). Coverage is reported
// per domain so a nascent frontier (agent auth) does not dilute or inflate the
// mature one (jwt).
type Domain string

const (
	DomainJWT   Domain = "jwt"   // JWT/JWKS/OIDC verifier threats
	DomainAgent Domain = "agent" // AI-agent auth: MCP, delegation, agent identity
)

// Category groups threats by the part of the verification pipeline they attack.
type Category string

const (
	CatSignature Category = "signature"            // is the signature actually verified?
	CatAlgorithm Category = "algorithm"            // alg selection / confusion / downgrade
	CatHeaderKey Category = "header_key_injection" // attacker-controlled key via jku/x5u/jwk/x5c/kid
	CatClaims    Category = "claims_validation"    // exp/nbf/iss/aud/typ/… enforcement
	CatKeyMgmt   Category = "key_management"       // key strength, rotation, retirement
	CatJWKS      Category = "jwks"                 // key-set delivery/refresh
	CatEncoding  Category = "encoding_parsing"     // structural / decoding weaknesses
	CatAuthz     Category = "authorization"        // token accepted but for the wrong principal/scope

	// Agent-auth categories.
	CatTokenBinding  Category = "token_binding"  // audience/resource-indicator binding, passthrough
	CatConsent       Category = "consent"        // confused-deputy, redirect/DCR, consent reuse
	CatDelegation    Category = "delegation"     // on-behalf-of, act/may_act, token exchange, chains
	CatScope         Category = "scope"          // over-scope, non-minimization, non-expiry
	CatAgentIdentity Category = "agent_identity" // agent vs user identity, sessions, workload identity
	CatAgency        Category = "agency"         // excessive agency: injection→tool misuse, tool poisoning
)

// DetectorKind identifies the family of Keyway detector that covers a threat.
type DetectorKind string

const (
	DetProbe       DetectorKind = "probe"       // a dynamic attack-token probe (internal/probe)
	DetHarness     DetectorKind = "harness"     // a generative attack-token check (internal/attack)
	DetDiff        DetectorKind = "diff"        // a contract-change classifier rule (internal/diff)
	DetLibDefaults DetectorKind = "libdefaults" // a known library-default finding (internal/libdefaults)
	DetBlast       DetectorKind = "blast"       // a blast-radius resolver (internal/blastradius)
)

// Detection links a threat to a concrete Keyway detector by stable ID. The
// coverage cross-check test verifies each probe ID actually exists in the probe
// registry, so this mapping cannot silently reference a detector that is gone.
type Detection struct {
	Kind DetectorKind
	ID   string
}

// Source is an external citation. Kind is the authority; Ref is the specific
// clause/identifier; URL is where to read it.
type Source struct {
	Kind string
	Ref  string
	URL  string
}

// Threat is one documented way an auth verifier can be attacked.
type Threat struct {
	ID          string // stable, e.g. "SIG-01"
	Title       string // one-line name
	Domain      Domain // set by Catalog(); jwt or agent
	Category    Category
	Severity    model.Severity
	Description string      // what the attack does
	Invariant   string      // the rule a correct verifier MUST uphold
	Sources     []Source    // ≥1 authoritative citation
	Detections  []Detection // empty ⇒ coverage gap
}

// Covered reports whether any Keyway detector exercises this threat.
func (t Threat) Covered() bool { return len(t.Detections) > 0 }

// --- citation helpers -------------------------------------------------------

func rfc8725(section, title string) Source {
	return Source{Kind: "RFC 8725", Ref: "§" + section + " " + title,
		URL: "https://datatracker.ietf.org/doc/html/rfc8725#section-" + section}
}
func cve(id string) Source {
	return Source{Kind: "CVE", Ref: id, URL: "https://nvd.nist.gov/vuln/detail/" + id}
}
func cwe(num, title string) Source {
	return Source{Kind: "CWE", Ref: "CWE-" + num + " " + title,
		URL: "https://cwe.mitre.org/data/definitions/" + num + ".html"}
}
func portswigger(anchor string) Source {
	return Source{Kind: "PortSwigger", Ref: "JWT attacks: " + anchor,
		URL: "https://portswigger.net/web-security/jwt"}
}
func owasp(ref, url string) Source { return Source{Kind: "OWASP", Ref: ref, URL: url} }

// rfc cites a generic RFC by number and title (RFC 8693/8707/9728/9700/7591, …).
func rfc(num, title string) Source {
	return Source{Kind: "RFC " + num, Ref: "RFC " + num + " " + title,
		URL: "https://datatracker.ietf.org/doc/html/rfc" + num}
}

// mcp cites the Model Context Protocol authorization / security spec.
func mcp(ref string) Source {
	return Source{Kind: "MCP", Ref: "MCP spec: " + ref,
		URL: "https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices"}
}

// draft cites an IETF/OpenID draft or whitepaper by title + URL.
func draft(ref, url string) Source { return Source{Kind: "Draft", Ref: ref, URL: url} }

func probeDet(id string) Detection      { return Detection{Kind: DetProbe, ID: id} }
func harnessDet(id string) Detection    { return Detection{Kind: DetHarness, ID: id} }
func libDefaultDet(id string) Detection { return Detection{Kind: DetLibDefaults, ID: id} }
func blastDet(id string) Detection      { return Detection{Kind: DetBlast, ID: id} }

// Catalog returns the full threat taxonomy across both domains, with each
// threat's Domain stamped. It is deliberately broader than what Keyway currently
// detects: the uncovered entries are the roadmap.
func Catalog() []Threat {
	out := make([]Threat, 0, 64)
	for _, t := range jwtThreats() {
		t.Domain = DomainJWT
		out = append(out, t)
	}
	for _, t := range agentThreats() {
		t.Domain = DomainAgent
		out = append(out, t)
	}
	return out
}

// jwtThreats is the JWT/JWKS/OIDC verifier threat surface.
func jwtThreats() []Threat {
	return []Threat{
		// ---- signature -----------------------------------------------------
		{
			ID: "SIG-01", Title: "none algorithm accepted", Category: CatSignature, Severity: model.SeverityCritical,
			Description: "Verifier accepts a token whose header sets alg=none (no signature), trusting arbitrary claims.",
			Invariant:   "A token with alg=none MUST be rejected unless unsecured JWTs are explicitly, separately enabled.",
			Sources:     []Source{rfc8725("3.2", "Use Appropriate Algorithms"), cve("CVE-2015-9235"), cwe("347", "Improper Verification of Cryptographic Signature")},
			Detections:  []Detection{probeDet("alg_none"), harnessDet("alg_none")},
		},
		{
			ID: "SIG-02", Title: "signature not verified", Category: CatSignature, Severity: model.SeverityCritical,
			Description: "Verifier decodes and trusts claims without cryptographically verifying the signature at all.",
			Invariant:   "Claims MUST NOT be trusted unless the signature verifies under a trusted key.",
			Sources:     []Source{cwe("347", "Improper Verification of Cryptographic Signature"), cwe("345", "Insufficient Verification of Data Authenticity")},
			Detections:  []Detection{probeDet("tampered_signature"), harnessDet("tampered_signature")},
		},
		{
			ID: "SIG-03", Title: "empty signature with a signing alg", Category: CatSignature, Severity: model.SeverityCritical,
			Description: "Token declares a real alg (e.g. RS256) but carries an empty signature segment; some libraries treat verification as vacuously true.",
			Invariant:   "A signing alg with an empty/missing signature MUST fail verification.",
			Sources:     []Source{rfc8725("3.1", "Perform Algorithm Verification"), cwe("347", "Improper Verification of Cryptographic Signature")},
			Detections:  []Detection{harnessDet("empty_signature_rs256")},
		},

		// ---- algorithm -----------------------------------------------------
		{
			ID: "ALG-01", Title: "RS/HS key confusion", Category: CatAlgorithm, Severity: model.SeverityCritical,
			Description: "Verifier configured for RS256 also accepts HS256, letting an attacker sign with the (public) RSA key as the HMAC secret.",
			Invariant:   "The accepted algorithm MUST be pinned; an asymmetric-keyed verifier MUST reject symmetric algs.",
			Sources:     []Source{rfc8725("3.1", "Perform Algorithm Verification"), cve("CVE-2016-5431"), cve("CVE-2022-23541")},
			Detections:  []Detection{probeDet("alg_confusion"), harnessDet("alg_confusion_rs_hs")},
		},
		{
			ID: "ALG-02", Title: "algorithm downgrade / unpinned alg", Category: CatAlgorithm, Severity: model.SeverityHigh,
			Description: "Verifier accepts any alg in a family (or a weaker one than expected) because it reads alg from the token instead of pinning it.",
			Invariant:   "The verifier MUST accept only an explicit allowlist of algorithms, not whatever the token's header requests.",
			Sources:     []Source{rfc8725("3.1", "Perform Algorithm Verification"), cve("CVE-2022-23540")},
			// gap
		},
		{
			ID: "ALG-03", Title: "alg header case/whitespace variant", Category: CatAlgorithm, Severity: model.SeverityHigh,
			Description: "Bypass of an alg blocklist via casing or padding, e.g. \"nONE\", \"None\", \"none \".",
			Invariant:   "Algorithm matching MUST be exact; normalized variants of \"none\" MUST all be rejected.",
			Sources:     []Source{rfc8725("3.1", "Perform Algorithm Verification"), portswigger("none variants")},
			Detections:  []Detection{harnessDet("alg_none_variants")},
		},
		{
			ID: "ALG-04", Title: "invalid ECDSA signature (0,0) — \"psychic signature\"", Category: CatAlgorithm, Severity: model.SeverityCritical,
			Description: "ECDSA verifier accepts r=0,s=0 (or other degenerate values) as a valid signature.",
			Invariant:   "Cryptographic inputs MUST be validated; degenerate ECDSA values MUST be rejected.",
			Sources:     []Source{rfc8725("3.4", "Validate Cryptographic Inputs"), cve("CVE-2022-21449")},
			Detections:  []Detection{harnessDet("psychic_signature_es256")},
		},

		// ---- header key injection -----------------------------------------
		{
			ID: "HDR-01", Title: "jku header → attacker-controlled JWKS", Category: CatHeaderKey, Severity: model.SeverityCritical,
			Description: "Verifier fetches the key set from the URL in the token's jku header, so the attacker supplies their own signing key (and an SSRF primitive).",
			Invariant:   "Key material MUST come from pre-configured trust, never from a URL inside the token being verified.",
			Sources:     []Source{rfc8725("3.4", "Validate Cryptographic Inputs"), portswigger("jku header injection"), cwe("290", "Authentication Bypass by Spoofing")},
			// gap
		},
		{
			ID: "HDR-02", Title: "x5u header → attacker-controlled certificate", Category: CatHeaderKey, Severity: model.SeverityCritical,
			Description: "As jku, but the token's x5u header points verification at an attacker-hosted X.509 cert.",
			Invariant:   "The verifier MUST ignore token-supplied x5u URLs.",
			Sources:     []Source{rfc8725("3.4", "Validate Cryptographic Inputs"), portswigger("x5u header injection")},
			// gap
		},
		{
			ID: "HDR-03", Title: "embedded jwk self-signed token", Category: CatHeaderKey, Severity: model.SeverityCritical,
			Description: "Verifier trusts a public key embedded in the token's jwk header, letting the attacker sign with a matching private key.",
			Invariant:   "A key embedded in the token header MUST NOT be used to verify that token.",
			Sources:     []Source{cve("CVE-2018-0114"), portswigger("embedded jwk"), cwe("347", "Improper Verification of Cryptographic Signature")},
			Detections:  []Detection{harnessDet("embedded_jwk")},
		},
		{
			ID: "HDR-04", Title: "x5c embedded certificate chain", Category: CatHeaderKey, Severity: model.SeverityHigh,
			Description: "Verifier trusts a certificate chain embedded in the token's x5c header without anchoring it to a configured trust store.",
			Invariant:   "An embedded x5c chain MUST be validated against a pre-configured trust anchor, never trusted on its own.",
			Sources:     []Source{rfc8725("3.4", "Validate Cryptographic Inputs")},
			// gap
		},
		{
			ID: "HDR-05", Title: "kid path traversal", Category: CatHeaderKey, Severity: model.SeverityHigh,
			Description: "kid is used to build a filesystem path to a key; ../ escapes let the attacker point at a predictable file (e.g. /dev/null → empty HMAC key).",
			Invariant:   "kid MUST be treated as an opaque lookup key, never interpolated into a path or query.",
			Sources:     []Source{portswigger("kid path traversal"), cwe("22", "Improper Limitation of a Pathname (Path Traversal)")},
			Detections:  []Detection{harnessDet("kid_path_traversal")},
		},
		{
			ID: "HDR-06", Title: "kid SQL / command injection", Category: CatHeaderKey, Severity: model.SeverityHigh,
			Description: "kid is interpolated into a SQL query or command used to fetch the key, enabling injection.",
			Invariant:   "kid MUST NOT be interpolated into SQL/commands; use parameterized lookups.",
			Sources:     []Source{portswigger("kid injection"), cwe("89", "SQL Injection")},
			// gap
		},

		// ---- claims validation --------------------------------------------
		{
			ID: "CLM-01", Title: "expired token accepted (exp not validated)", Category: CatClaims, Severity: model.SeverityHigh,
			Description: "Verifier does not check exp, so revoked/aged tokens keep working.",
			Invariant:   "exp MUST be validated against current time (with bounded skew).",
			Sources:     []Source{rfc8725("3.9", "Use and Validate Audience"), owasp("JWT Cheat Sheet: token expiration", "https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html")},
			Detections:  []Detection{probeDet("expired"), harnessDet("expired")},
		},
		{
			ID: "CLM-02", Title: "not-yet-valid token accepted (nbf ignored)", Category: CatClaims, Severity: model.SeverityMedium,
			Description: "Verifier ignores nbf, accepting tokens before their validity window.",
			Invariant:   "nbf MUST be validated against current time (with bounded skew).",
			Sources:     []Source{rfc8725("3.9", "Use and Validate Audience")},
			Detections:  []Detection{probeDet("not_yet_valid"), harnessDet("not_yet_valid")},
		},
		{
			ID: "CLM-03", Title: "issuer not validated", Category: CatClaims, Severity: model.SeverityHigh,
			Description: "Verifier accepts a well-formed token from any issuer, so a token minted by a different (attacker-controlled) IdP is trusted.",
			Invariant:   "iss MUST match the expected issuer exactly.",
			Sources:     []Source{rfc8725("3.8", "Validate Issuer and Subject")},
			Detections:  []Detection{probeDet("wrong_issuer"), harnessDet("wrong_issuer")},
		},
		{
			ID: "CLM-04", Title: "audience not validated", Category: CatClaims, Severity: model.SeverityHigh,
			Description: "Verifier does not check aud, so a token minted for service A is accepted by service B (token replay across audiences).",
			Invariant:   "aud MUST contain this service's expected value.",
			Sources:     []Source{rfc8725("3.9", "Use and Validate Audience")},
			Detections:  []Detection{probeDet("wrong_audience"), harnessDet("wrong_audience")},
		},
		{
			ID: "CLM-05", Title: "required authorization claim not enforced", Category: CatClaims, Severity: model.SeverityHigh,
			Description: "An app-required claim (scope/role/tenant) is not enforced, so a valid but under-privileged token is accepted.",
			Invariant:   "Every claim the policy requires MUST be present and checked.",
			Sources:     []Source{rfc8725("3.10", "Do Not Trust Received Claims")},
			Detections:  []Detection{probeDet("missing_required_claim")},
		},
		{
			ID: "CLM-06", Title: "aud array/string type confusion", Category: CatClaims, Severity: model.SeverityMedium,
			Description: "aud may be a string or an array; a verifier that only handles one shape can be bypassed by sending the other.",
			Invariant:   "Audience matching MUST handle both the string and array encodings of aud.",
			Sources:     []Source{rfc8725("3.9", "Use and Validate Audience"), owasp("API2:2023 Broken Authentication", "https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/")},
			Detections:  []Detection{harnessDet("aud_array_confusion")},
		},
		{
			ID: "CLM-07", Title: "claim type confusion", Category: CatClaims, Severity: model.SeverityMedium,
			Description: "Claims sent with unexpected types (exp as a string, aud as an object) slip past a naive validator.",
			Invariant:   "Claim types MUST be validated; a mistyped claim MUST be a rejection, not a silent skip.",
			Sources:     []Source{rfc8725("3.10", "Do Not Trust Received Claims")},
			// gap
		},
		{
			ID: "CLM-08", Title: "token with no expiry accepted", Category: CatClaims, Severity: model.SeverityMedium,
			Description: "A token that simply omits exp is accepted as never-expiring.",
			Invariant:   "A token lacking exp MUST be rejected where expiry is required by policy.",
			Sources:     []Source{owasp("JWT Cheat Sheet: token expiration", "https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html")},
			Detections:  []Detection{harnessDet("missing_exp")},
		},
		{
			ID: "CLM-09", Title: "explicit typing not enforced (typ/cty)", Category: CatClaims, Severity: model.SeverityMedium,
			Description: "Verifier ignores the typ header, allowing one kind of JWT (e.g. a refresh or a nested JWE) to be used where another is expected.",
			Invariant:   "Where multiple JWT types exist, typ SHOULD be validated (explicit typing).",
			Sources:     []Source{rfc8725("3.11", "Use Explicit Typing")},
			// gap
		},
		{
			ID: "CLM-10", Title: "cross-type token substitution", Category: CatClaims, Severity: model.SeverityHigh,
			Description: "An access token, ID token, and refresh token share a key; one is accepted in place of another because validation rules aren't mutually exclusive.",
			Invariant:   "Different JWT kinds MUST have mutually exclusive validation rules (distinct aud/typ/issuer scoping).",
			Sources:     []Source{rfc8725("3.12", "Use Mutually Exclusive Validation Rules for Different Kinds of JWTs")},
			// gap
		},

		// ---- authorization -------------------------------------------------
		{
			ID: "AUTHZ-01", Title: "sibling-audience token accepted", Category: CatAuthz, Severity: model.SeverityHigh,
			Description: "A validly-signed token issued for a sibling service is accepted, crossing a trust boundary.",
			Invariant:   "aud (and issuer scoping) MUST confine a token to its intended service.",
			Sources:     []Source{rfc8725("3.9", "Use and Validate Audience")},
			Detections:  []Detection{probeDet("sibling_client_token")},
		},
		{
			ID: "AUTHZ-02", Title: "identity header trusted without a token", Category: CatAuthz, Severity: model.SeverityCritical,
			Description: "A gateway/service trusts an identity header (X-User, X-Forwarded-*) without a verified token behind it.",
			Invariant:   "Identity MUST derive from a verified token, not a spoofable header.",
			Sources:     []Source{cwe("290", "Authentication Bypass by Spoofing")},
			Detections:  []Detection{probeDet("header_bypass")},
		},
		{
			ID: "AUTHZ-03", Title: "authorized party (azp/client_id) not checked", Category: CatAuthz, Severity: model.SeverityMedium,
			Description: "A token from a different OAuth client is accepted because azp/client_id is not validated.",
			Invariant:   "Where policy scopes a token to a client, azp/client_id MUST be validated.",
			Sources:     []Source{owasp("API2:2023 Broken Authentication", "https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/")},
			// gap
		},

		// ---- key management ------------------------------------------------
		{
			ID: "KEY-01", Title: "retired key still accepted", Category: CatKeyMgmt, Severity: model.SeverityHigh,
			Description: "A key removed from active signing is still honored by verifiers, extending the blast radius of a leaked key.",
			Invariant:   "A retired/removed key MUST stop verifying once its grace period ends.",
			Sources:     []Source{rfc8725("3.5", "Ensure Cryptographic Keys Have Sufficient Entropy")},
			Detections:  []Detection{probeDet("retired_key")},
		},
		{
			ID: "KEY-02", Title: "unknown-kid rotation outage", Category: CatKeyMgmt, Severity: model.SeverityHigh,
			Description: "Verifier caches JWKS and does not refresh on an unknown kid, so a routine key rotation breaks all verification (availability, and pressure to disable checks).",
			Invariant:   "On an unknown kid the verifier SHOULD refresh JWKS before rejecting.",
			Sources:     []Source{owasp("JWKS rotation guidance", "https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html")},
			Detections:  []Detection{libDefaultDet("refreshes_on_unknown_kid"), blastDet("rotate_key"), probeDet("canary_key")},
		},
		{
			ID: "KEY-03", Title: "weak asymmetric key accepted (RSA < 2048)", Category: CatKeyMgmt, Severity: model.SeverityMedium,
			Description: "Verifier accepts tokens signed by an undersized RSA key, which is feasible to attack.",
			Invariant:   "Signing keys MUST meet minimum strength (e.g. RSA ≥ 2048).",
			Sources:     []Source{rfc8725("3.5", "Ensure Cryptographic Keys Have Sufficient Entropy"), cwe("326", "Inadequate Encryption Strength")},
			// gap
		},
		{
			ID: "KEY-04", Title: "weak/guessable HMAC secret", Category: CatKeyMgmt, Severity: model.SeverityHigh,
			Description: "HS256 secret has low entropy and is brute-forceable offline from a single captured token.",
			Invariant:   "Symmetric secrets MUST have sufficient entropy; short/dictionary secrets MUST be treated as compromised.",
			Sources:     []Source{rfc8725("3.5", "Ensure Cryptographic Keys Have Sufficient Entropy"), cwe("521", "Weak Password Requirements")},
			// gap
		},

		// ---- jwks ----------------------------------------------------------
		{
			ID: "JWKS-01", Title: "JWKS fetched over plaintext HTTP", Category: CatJWKS, Severity: model.SeverityHigh,
			Description: "Verifier loads its key set over http://, letting a network attacker swap in their own keys.",
			Invariant:   "JWKS/OIDC metadata MUST be fetched over TLS.",
			Sources:     []Source{rfc8725("3.4", "Validate Cryptographic Inputs"), cwe("319", "Cleartext Transmission of Sensitive Information")},
			// gap
		},
		{
			ID: "JWKS-02", Title: "issuer discovery follows redirects to internal hosts", Category: CatJWKS, Severity: model.SeverityMedium,
			Description: "OIDC discovery / JWKS fetch follows redirects, so a compromised issuer can point the verifier at internal metadata (SSRF).",
			Invariant:   "Discovery/JWKS fetches MUST NOT follow redirects to untrusted hosts.",
			Sources:     []Source{cwe("918", "Server-Side Request Forgery (SSRF)")},
			// gap (note: Keyway's OWN client is hardened for this — SEC-02/03 — but it is not yet a probe against consumers)
		},

		// ---- encoding / parsing -------------------------------------------
		{
			ID: "ENC-01", Title: "malformed structure / extra segments accepted", Category: CatEncoding, Severity: model.SeverityMedium,
			Description: "Tokens with the wrong number of segments or trailing data are accepted by a lax parser.",
			Invariant:   "A JWS MUST have exactly three base64url segments; anything else MUST be rejected.",
			Sources:     []Source{rfc8725("3.1", "Perform Algorithm Verification"), cwe("20", "Improper Input Validation")},
			Detections:  []Detection{harnessDet("extra_segments")},
		},
		{
			ID: "ENC-02", Title: "lax base64 / non-canonical encoding", Category: CatEncoding, Severity: model.SeverityLow,
			Description: "Standard-base64 or padded input is accepted where only base64url is valid, enabling smuggling past filters.",
			Invariant:   "Segments MUST be strict base64url without padding.",
			Sources:     []Source{rfc8725("3.7", "Use UTF-8")},
			Detections:  []Detection{harnessDet("non_base64url")},
		},
		{
			ID: "ENC-03", Title: "JWE decompression bomb", Category: CatEncoding, Severity: model.SeverityMedium,
			Description: "A compressed JWE expands to exhaust memory (a \"billion laughs\" for tokens).",
			Invariant:   "Compressed token payloads MUST be size-bounded, or compression disabled.",
			Sources:     []Source{rfc8725("3.6", "Avoid Compression of Encrypted Data")},
			// gap
		},
	}
}
