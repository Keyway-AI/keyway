package attack

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

// Policy is what a correctly-configured verifier enforces: a pinned algorithm and
// trust key, the expected issuer/audience, and any required claims.
type Policy struct {
	TrustedKey     *rsa.PublicKey
	Issuer         string
	Audience       string
	RequiredClaims []string
	Now            time.Time
	Skew           time.Duration
}

// Oracle is the reference verifier — a from-spec-correct implementation of the
// invariants, built on go-jose (a mature, independent JOSE library). It pins the
// algorithm to RS256 and the key to the configured trust key, ignores any key
// material carried inside the token (jku/x5u/jwk/x5c/kid), and validates claims.
// Using a real library as the oracle keeps corpus validation from being circular:
// "a known-correct verifier rejects every attack token."
type Oracle struct{ Policy Policy }

// Verify returns Accept only when go-jose confirms an RS256 signature under the
// trusted key AND every claim invariant holds; otherwise Reject.
func (o Oracle) Verify(_ context.Context, token string) (Verdict, error) {
	// Pin the algorithm at parse time: anything but RS256 (none, HS*, ES*, …) is
	// refused before any crypto runs — the single most important invariant.
	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return Reject, nil
	}
	// Verify against the pre-configured trust key only. Any key embedded in the
	// token header is never consulted, so jku/x5u/jwk/x5c/kid attacks fail here.
	payload, err := jws.Verify(o.Policy.TrustedKey)
	if err != nil {
		return Reject, nil
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Reject, nil
	}
	if o.validClaims(claims) {
		return Accept, nil
	}
	return Reject, nil
}

func (o Oracle) validClaims(c map[string]any) bool {
	now := o.Policy.Now
	if now.IsZero() {
		now = time.Now()
	}
	skew := o.Policy.Skew

	// exp: required and not in the past.
	exp, ok := numClaim(c, "exp")
	if !ok {
		return false
	}
	if now.After(time.Unix(int64(exp), 0).Add(skew)) {
		return false
	}
	// nbf: if present, must have started.
	if nbf, ok := numClaim(c, "nbf"); ok {
		if now.Before(time.Unix(int64(nbf), 0).Add(-skew)) {
			return false
		}
	}
	// iss: exact match.
	if o.Policy.Issuer != "" {
		if iss, _ := c["iss"].(string); iss != o.Policy.Issuer {
			return false
		}
	}
	// aud: must contain the expected value (string OR array encoding).
	if o.Policy.Audience != "" && !audienceContains(c["aud"], o.Policy.Audience) {
		return false
	}
	// required application claims must be present and non-empty.
	for _, rc := range o.Policy.RequiredClaims {
		v, present := c[rc]
		if !present || v == nil || v == "" {
			return false
		}
	}
	return true
}

func numClaim(c map[string]any, key string) (float64, bool) {
	// JSON numbers unmarshal to float64; a non-numeric value (type confusion) is
	// treated as absent, which is itself a rejection.
	f, ok := c[key].(float64)
	return f, ok
}

// audienceContains handles both aud shapes: a string, or an array of strings.
func audienceContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}
