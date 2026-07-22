// Package probe mints synthetic tokens and verifies consumer behaviour against
// real staging endpoints (PRD §6). Minted tokens are NEVER persisted.
package probe

import (
	"time"

	"github.com/architsharma/keyway/internal/model"
)

// MintContext is passed to a probe's Mutate function. It exposes everything a
// probe needs to construct its Authorization header without knowing how signing
// works.
type MintContext struct {
	Issuer   model.Issuer
	Consumer model.Consumer
	// Claims is the baseline claim set (PRD §6.2), pre-populated per probe run.
	Claims map[string]any
	Now    time.Time
	// Mint signs the given claims with the named key and returns a compact JWT.
	Mint func(kid string, claims map[string]any) (string, error)
	// Convenience key IDs resolved from the issuer's JWKS.
	ActiveKID    string
	AnnouncedKID string // canary; empty if none announced
	RetiredKID   string // empty if none retired
}

// Expectation encodes what a PASS looks like for a probe.
type Expectation struct {
	// AcceptStatuses is the set of HTTP status codes considered a PASS.
	AcceptStatuses []int
	// ShouldAccept records whether the contract says this token SHOULD be accepted.
	ShouldAccept bool
}

// Probe is one verification behaviour.
type Probe struct {
	ID                 string
	Name               string
	Description        string
	RequiresPrivateKey bool
	// AppliesTo filters which consumers a probe runs against; nil means all.
	AppliesTo func(c model.Consumer) bool
	// Mutate returns the Authorization header value and any extra headers.
	// Implemented in M3 (see PROGRESS.md); nil until then.
	Mutate   func(ctx MintContext) (authHeader string, extraHeaders map[string]string, err error)
	Expect   Expectation
	Severity model.Severity
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

// accept2xx / reject401 helpers for readability.
var (
	accept2xx = Expectation{AcceptStatuses: []int{200, 201, 202, 204}, ShouldAccept: true}
	reject401 = Expectation{AcceptStatuses: []int{401}, ShouldAccept: false}
	reject4xx = Expectation{AcceptStatuses: []int{401, 403}, ShouldAccept: false}
)

// Definitions returns the 13 probe definitions with their metadata exactly per
// PRD §6.2. The Mutate token-construction logic is wired in M3; the metadata
// (severity, expected outcome, key requirement) is authoritative now.
func Definitions() []Probe {
	return []Probe{
		{ID: ProbeValidToken, Name: "Valid token", Description: "Baseline claims signed with the active key.", RequiresPrivateKey: true, Expect: accept2xx, Severity: model.SeverityInfo},
		{ID: ProbeExpired, Name: "Expired token", Description: "exp in the past must be rejected.", RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh},
		{ID: ProbeNotYetValid, Name: "Not yet valid", Description: "nbf in the future must be rejected.", RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityMedium},
		{ID: ProbeWrongIssuer, Name: "Wrong issuer", Description: "Foreign iss must be rejected.", RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh},
		{ID: ProbeWrongAudience, Name: "Wrong audience", Description: "Foreign aud must be rejected.", RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh},
		{ID: ProbeAlgNone, Name: "alg=none", Description: "Unsigned token must be rejected.", RequiresPrivateKey: false, Expect: reject401, Severity: model.SeverityCritical},
		{ID: ProbeAlgConfusion, Name: "Algorithm confusion", Description: "RS256 public key used as HS256 secret must be rejected.", RequiresPrivateKey: false, Expect: reject401, Severity: model.SeverityCritical},
		{ID: ProbeTamperedSignature, Name: "Tampered signature", Description: "Flipped signature byte must be rejected.", RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh},
		{ID: ProbeMissingClaim, Name: "Missing required claim", Description: "One sub-probe per required claim, each omitted in turn.", RequiresPrivateKey: true, Expect: reject4xx, Severity: model.SeverityMedium},
		{ID: ProbeRetiredKey, Name: "Retired key", Description: "Token signed with a retired key must be rejected.", RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh},
		{ID: ProbeSiblingClientToken, Name: "Sibling client token", Description: "A valid token minted for a different consumer's audience must be rejected.", RequiresPrivateKey: true, Expect: reject401, Severity: model.SeverityHigh},
		{ID: ProbeHeaderBypass, Name: "Header bypass", Description: "No bearer token, only trusted-looking identity headers, must be rejected.", RequiresPrivateKey: false, Expect: reject401, Severity: model.SeverityCritical},
		{ID: ProbeCanaryKey, Name: "Canary key", Description: "Token signed with the announced (canary) key should be accepted once the consumer picks it up.", RequiresPrivateKey: true, Expect: accept2xx, Severity: model.SeverityInfo},
	}
}
