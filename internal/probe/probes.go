// Package probe mints synthetic tokens and verifies consumer behavior against
// real staging endpoints (PRD §6). Minted tokens are NEVER persisted.
package probe

import (
	"errors"
	"time"

	"github.com/Keyway-AI/keyway/internal/model"
)

// MintFunc signs claims with the named key (empty kid = active key).
type MintFunc func(kid string, claims map[string]any) (string, error)

// ErrProbeNotApplicable signals that a probe cannot run for a consumer (e.g. no
// canary key announced, no retired key, no sibling audience). The engine records
// it as skipped, never as a finding.
var ErrProbeNotApplicable = errors.New("probe not applicable")

// MintContext is passed to a probe's Mutate function. It exposes everything a
// probe needs to construct its Authorization header without knowing how signing
// works.
type MintContext struct {
	Issuer   model.Issuer
	Consumer model.Consumer
	// Claims is the baseline claim set (PRD §6.2), fresh per probe invocation.
	Claims map[string]any
	Now    time.Time
	Mint   MintFunc

	// Key IDs resolved from the issuer's JWKS.
	ActiveKID    string
	AnnouncedKID string // canary; empty if none announced
	RetiredKID   string // empty if none retired
	// ActiveKeyPEM is the active key's PEM-encoded public key (probe 7 secret).
	ActiveKeyPEM string
	// SiblingAudience is another consumer's audience (probe 11); empty if none.
	SiblingAudience string
	// OmitClaim, when set, is the required claim to drop (probe 9 sub-probes).
	OmitClaim string
}

// Expectation encodes what a PASS looks like for a probe.
type Expectation struct {
	AcceptStatuses []int
	ShouldAccept   bool
}

// Accepts reports whether a status code is a PASS for this expectation.
func (e Expectation) Accepts(status int) bool {
	for _, s := range e.AcceptStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// MutateFunc returns the Authorization header value and any extra headers.
type MutateFunc func(ctx MintContext) (authHeader string, extraHeaders map[string]string, err error)

// Probe is one verification behavior.
type Probe struct {
	ID                 string
	Name               string
	Description        string
	RequiresPrivateKey bool
	AppliesTo          func(c model.Consumer) bool
	Mutate             MutateFunc
	Expect             Expectation
	Severity           model.Severity
}

// Probe ID constants — stable identifiers referenced by evidence and the API.
const (
	ProbeValidToken         = "valid_token"
	ProbeExpired            = "expired"
	ProbeNotYetValid        = "not_yet_valid"
	ProbeWrongIssuer        = "wrong_issuer"
	ProbeWrongAudience      = "wrong_audience"
	ProbeAlgNone            = "alg_none"
	ProbeAlgConfusion       = "alg_confusion"
	ProbeTamperedSignature  = "tampered_signature"
	ProbeMissingClaim       = "missing_required_claim"
	ProbeRetiredKey         = "retired_key"
	ProbeSiblingClientToken = "sibling_client_token"
	ProbeHeaderBypass       = "header_bypass"
	ProbeCanaryKey          = "canary_key"
)

var (
	accept2xx = Expectation{AcceptStatuses: []int{200, 201, 202, 204}, ShouldAccept: true}
	reject401 = Expectation{AcceptStatuses: []int{401}, ShouldAccept: false}
	reject4xx = Expectation{AcceptStatuses: []int{401, 403}, ShouldAccept: false}
)

const (
	invalidIssuer   = "https://issuer.keyway.invalid"
	invalidAudience = "keyway-invalid-audience"
)

func bearer(token string) (string, map[string]string, error) {
	return "Bearer " + token, nil, nil
}

// Definitions returns the 13 probe definitions with their mutation logic (PRD
// §6.2). Probe 9 (missing_required_claim) is expanded into per-claim sub-probes
// by the engine; its Mutate drops the single claim named in ctx.OmitClaim.
func Definitions() []Probe {
	return []Probe{
		{
			ID: ProbeValidToken, Name: "Valid token",
			Description:        "Baseline claims signed with the active key.",
			RequiresPrivateKey: true, Expect: accept2xx, Severity: model.SeverityInfo,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeExpired, Name: "Expired token",
			Description:        "exp in the past must be rejected.",
			RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				ctx.Claims["exp"] = ctx.Now.Add(-time.Hour).Unix()
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeNotYetValid, Name: "Not yet valid",
			Description:        "nbf in the future must be rejected.",
			RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityMedium,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				ctx.Claims["nbf"] = ctx.Now.Add(time.Hour).Unix()
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeWrongIssuer, Name: "Wrong issuer",
			Description:        "Foreign iss must be rejected.",
			RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				ctx.Claims["iss"] = invalidIssuer
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeWrongAudience, Name: "Wrong audience",
			Description:        "Foreign aud must be rejected.",
			RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				ctx.Claims["aud"] = invalidAudience
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeAlgNone, Name: "alg=none",
			Description:        "Unsigned token must be rejected.",
			RequiresPrivateKey: false, Expect: reject401, Severity: model.SeverityCritical,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				tok, err := RawUnsignedToken(ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeAlgConfusion, Name: "Algorithm confusion",
			Description:        "RS256 public key used as HS256 secret must be rejected.",
			RequiresPrivateKey: false, Expect: reject401, Severity: model.SeverityCritical,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				if ctx.ActiveKeyPEM == "" {
					return "", nil, ErrProbeNotApplicable
				}
				// The exact PEM bytes, including header/footer/newline, are the secret.
				tok, err := SignHS256([]byte(ctx.ActiveKeyPEM), ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeTamperedSignature, Name: "Tampered signature",
			Description:        "Flipped signature byte must be rejected.",
			RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				tampered, err := TamperSignature(tok)
				if err != nil {
					return "", nil, err
				}
				return bearer(tampered)
			},
		},
		{
			ID: ProbeMissingClaim, Name: "Missing required claim",
			Description:        "One sub-probe per required claim, each omitted in turn.",
			RequiresPrivateKey: true, Expect: reject4xx, Severity: model.SeverityMedium,
			AppliesTo: func(c model.Consumer) bool { return len(c.Expects.RequiredClaims) > 0 },
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				if ctx.OmitClaim == "" {
					return "", nil, ErrProbeNotApplicable
				}
				delete(ctx.Claims, ctx.OmitClaim)
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeRetiredKey, Name: "Retired key",
			Description:        "Token signed with a retired key must be rejected.",
			RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				if ctx.RetiredKID == "" {
					return "", nil, ErrProbeNotApplicable
				}
				tok, err := ctx.Mint(ctx.RetiredKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeSiblingClientToken, Name: "Sibling client token",
			Description:        "A valid token minted for a different consumer's audience must be rejected.",
			RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				if ctx.SiblingAudience == "" {
					return "", nil, ErrProbeNotApplicable
				}
				ctx.Claims["aud"] = ctx.SiblingAudience
				tok, err := ctx.Mint(ctx.ActiveKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
		{
			ID: ProbeHeaderBypass, Name: "Header bypass",
			Description:        "No bearer token, only trusted-looking identity headers, must be rejected.",
			RequiresPrivateKey: false, Expect: reject401, Severity: model.SeverityCritical,
			Mutate: func(MintContext) (string, map[string]string, error) {
				return "", map[string]string{
					"X-User-Id":            "keyway-test",
					"X-Forwarded-User":     "keyway-test",
					"X-Authenticated-User": "keyway-test",
				}, nil
			},
		},
		{
			ID: ProbeCanaryKey, Name: "Canary key",
			Description:        "Token signed with the announced (canary) key should be accepted once the consumer picks it up.",
			RequiresPrivateKey: true, Expect: accept2xx, Severity: model.SeverityInfo,
			Mutate: func(ctx MintContext) (string, map[string]string, error) {
				if ctx.AnnouncedKID == "" {
					return "", nil, ErrProbeNotApplicable
				}
				tok, err := ctx.Mint(ctx.AnnouncedKID, ctx.Claims)
				if err != nil {
					return "", nil, err
				}
				return bearer(tok)
			},
		},
	}
}
