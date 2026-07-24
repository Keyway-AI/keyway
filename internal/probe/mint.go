package probe

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nometria/keyway/internal/model"
	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
)

// BaselineClaims builds the standard claim set for a minted token (PRD §6.2).
// Every required claim gets a placeholder value. The returned map is fresh per
// call so probes can mutate it without cross-contamination.
func BaselineClaims(iss model.Issuer, c model.Consumer, now time.Time) map[string]any {
	aud := ""
	if len(c.Expects.Audiences) > 0 {
		aud = c.Expects.Audiences[0]
	}
	issURL := iss.IssuerURL
	if len(c.Expects.Issuers) > 0 {
		issURL = c.Expects.Issuers[0]
	}
	claims := map[string]any{
		"iss": issURL,
		"aud": aud,
		"sub": "keyway-synthetic-principal",
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"nbf": now.Unix(),
		"jti": uuid.NewString(),
	}
	for _, name := range c.Expects.RequiredClaims {
		if _, exists := claims[name]; !exists {
			claims[name] = "keyway-placeholder"
		}
	}
	return claims
}

// b64 is the raw (unpadded) base64url encoding used by JWS.
var b64 = base64.RawURLEncoding

// RawUnsignedToken builds a compact JWT with header {"alg":"none"} and an empty
// signature segment (probe 6). Most JOSE libraries refuse to emit alg=none, so
// the string is assembled by hand: base64url(header) + "." + base64url(payload) + "."
func RawUnsignedToken(claims map[string]any) (string, error) {
	header := map[string]any{"alg": "none", "typ": "JWT"}
	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	return b64.EncodeToString(hb) + "." + b64.EncodeToString(pb) + ".", nil
}

// SignHS256 signs claims with HMAC-SHA256 using the given secret (probe 7). The
// attack presents an RS256 issuer's PUBLIC key PEM bytes as the HMAC secret; a
// server that naively trusts the token's alg header will verify it.
func SignHS256(secret []byte, claims map[string]any) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: secret},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", fmt.Errorf("probe: hs256 signer: %w", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	obj, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("probe: hs256 sign: %w", err)
	}
	return obj.CompactSerialize()
}

// TamperSignature flips the final byte of a compact JWS signature (probe 8),
// producing a structurally valid but cryptographically invalid token.
func TamperSignature(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("probe: not a compact JWS")
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("probe: decode signature: %w", err)
	}
	if len(sig) == 0 {
		return "", fmt.Errorf("probe: empty signature")
	}
	sig[len(sig)-1] ^= 0xFF // flip the final byte
	parts[2] = b64.EncodeToString(sig)
	return strings.Join(parts, "."), nil
}
