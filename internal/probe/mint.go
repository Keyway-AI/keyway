package probe

import (
	"time"

	"github.com/architsharma/keyway/internal/model"
	"github.com/google/uuid"
)

// BaselineClaims builds the standard claim set for a minted token (PRD §6.2).
// Every required claim gets a placeholder value. Note: the returned map is fresh
// per call so probes can mutate it without cross-contamination.
func BaselineClaims(iss model.Issuer, c model.Consumer, now time.Time) map[string]any {
	aud := ""
	if len(c.Expects.Audiences) > 0 {
		aud = c.Expects.Audiences[0]
	}
	claims := map[string]any{
		"iss": iss.IssuerURL,
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

// TODO(M3): mint.go will also host the raw alg=none construction (probe 6),
// the HS256/RS256-public-key confusion secret derivation (probe 7), and the
// tampered-signature byte flip (probe 8). Those must build the compact JWS
// manually where go-jose refuses (PRD §6.2 implementation notes).
