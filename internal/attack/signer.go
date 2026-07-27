package attack

import (
	"crypto/rand"
	"crypto/rsa"
	"time"
)

// TrustedSigner is the seam between the harness and the *trusted issuer*. The
// harness signs its control token and its claim-level attacks (bad claims, valid
// signature) through this, so in production those tokens are signed by the real
// issuer's key (via the probe engine's MintFunc) and a live target's signature
// check passes — leaving the claim validation as the thing actually under test.
// In tests it is backed by a local key so the corpus can be validated offline.
type TrustedSigner interface {
	// Sign returns a compact JWS validly signed by the trusted issuer over claims.
	Sign(claims map[string]any) (string, error)
	// PublicKeyPEM returns the issuer's public signing key (PKIX PEM), used to
	// build the RS/HS confusion attack. Empty means that generator is skipped.
	PublicKeyPEM() string
}

// localSigner signs with an in-process RSA key (RS256). It is the default backing
// for NewContext so the corpus self-validates against the oracle offline.
type localSigner struct{ key *rsa.PrivateKey }

func (s localSigner) Sign(claims map[string]any) (string, error) {
	input, err := signingInput(header("RS256", map[string]any{"kid": "trusted"}), claims)
	if err != nil {
		return "", err
	}
	sig, err := signRS256(input, s.key)
	if err != nil {
		return "", err
	}
	return assemble(input, sig), nil
}

func (s localSigner) PublicKeyPEM() string {
	pem, err := publicKeyPEM(&s.key.PublicKey)
	if err != nil {
		return ""
	}
	return string(pem)
}

// NewContext builds a GenContext with a local trusted signer and fresh attacker
// key, plus the matching Policy for the reference oracle (same trust key). Used
// for offline corpus validation.
func NewContext() (GenContext, Policy, error) {
	issuer, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return GenContext{}, Policy{}, err
	}
	ctx, err := newContext(localSigner{issuer}, "https://issuer.keyway.test", "payments-api", []string{"scope"}, time.Now())
	if err != nil {
		return GenContext{}, Policy{}, err
	}
	pol := Policy{
		TrustedKey: &issuer.PublicKey, Issuer: ctx.Issuer, Audience: ctx.Audience,
		RequiredClaims: ctx.RequiredClaims, Now: ctx.Now, Skew: 60 * time.Second,
	}
	return ctx, pol, nil
}

// NewLiveContext builds a GenContext for scanning a real target: claim-level
// attacks are signed by the given trusted signer (the real issuer), while
// forgeries use a fresh attacker key. issuer/audience/required come from the
// consumer's expected contract so the control token is one the target accepts.
func NewLiveContext(trusted TrustedSigner, issuer, audience string, required []string, now time.Time) (GenContext, error) {
	return newContext(trusted, issuer, audience, required, now)
}

func newContext(trusted TrustedSigner, issuer, audience string, required []string, now time.Time) (GenContext, error) {
	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return GenContext{}, err
	}
	return GenContext{
		Trusted: trusted, AttackerKey: attacker,
		Issuer: issuer, Audience: audience, RequiredClaims: required, Now: now,
	}, nil
}
