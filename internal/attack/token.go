// Package attack is Keyway's generative, invariant-based JWT attack harness. It
// replaces a fixed list of hand-written attack tokens with taxonomy-driven
// generators: each generator emits one or more concrete tokens tagged with the
// threat (from internal/threats) it exercises and the verdict a *correct* verifier
// must return. A reference verifier (oracle, built on go-jose) encodes the
// security invariants, so the corpus is self-validating — every "must reject"
// token is provably rejected by a known-correct implementation — and Evaluate can
// then fire the same corpus at a live target and flag any it wrongly accepts.
package attack

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// b64 is the JWS segment encoding: base64url, no padding.
var b64 = base64.RawURLEncoding

func encodeSegment(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return b64.EncodeToString(raw), nil
}

// signingInput builds the "header.payload" string that a JWS signs over.
func signingInput(header, claims any) (string, error) {
	h, err := encodeSegment(header)
	if err != nil {
		return "", err
	}
	p, err := encodeSegment(claims)
	if err != nil {
		return "", err
	}
	return h + "." + p, nil
}

// assemble joins a signing input with an (already base64url-encoded) signature.
func assemble(input, sigB64 string) string { return input + "." + sigB64 }

// --- signing primitives (hand-rolled so we can also produce *malformed* JWS
// that a well-behaved library would refuse to emit) --------------------------

func signRS256(input string, priv *rsa.PrivateKey) (string, error) {
	sum := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return b64.EncodeToString(sig), nil
}

func signHS256(input string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(input))
	return b64.EncodeToString(mac.Sum(nil))
}

// jwkJSON marshals a public key as a JWK object, for embedding in a jwk/x5u-style
// header (the HDR-03 embedded-key attack).
func jwkJSON(pub any) (json.RawMessage, error) {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return json.RawMessage(fmt.Sprintf(
			`{"kty":"RSA","n":%q,"e":%q}`,
			b64.EncodeToString(k.N.Bytes()),
			b64.EncodeToString(big.NewInt(int64(k.E)).Bytes()),
		)), nil
	default:
		return nil, fmt.Errorf("attack: unsupported key type %T for jwk embedding", pub)
	}
}
