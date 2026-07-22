// Package model holds Keyway's core domain types. It has NO dependencies on
// other internal packages — keep it that way (see CONTRIBUTING.md).
package model

import (
	"encoding/json"
	"time"
)

// IssuerType enumerates the token issuers Keyway can work with.
type IssuerType string

const (
	IssuerKeycloak    IssuerType = "keycloak"
	IssuerK8sSA       IssuerType = "k8s_sa"
	IssuerGenericOIDC IssuerType = "generic_oidc"
	IssuerAuth0       IssuerType = "auth0"
	IssuerOkta        IssuerType = "okta"
	IssuerEntra       IssuerType = "entra"
)

// KeyStatus is the lifecycle state of a signing key in the rotation flow.
type KeyStatus string

const (
	// KeyAnnounced is published in JWKS but not yet used for signing. The canary state.
	KeyAnnounced KeyStatus = "announced"
	// KeyActive is currently used for signing.
	KeyActive KeyStatus = "active"
	// KeyRetiring is no longer signing, still published for validation of outstanding tokens.
	KeyRetiring KeyStatus = "retiring"
	// KeyRetired is removed from JWKS.
	KeyRetired KeyStatus = "retired"
)

// Key is a single signing key observed in (or managed within) an issuer's JWKS.
type Key struct {
	KID               string     `json:"kid"`
	Alg               string     `json:"alg"`
	Use               string     `json:"use"`
	PublicKeyPEM      string     `json:"public_key_pem"`
	Status            KeyStatus  `json:"status"`
	FirstSeenInJWKS   time.Time  `json:"first_seen_in_jwks"`
	InSigningUseSince *time.Time `json:"in_signing_use_since,omitempty"`
	RetiredAt         *time.Time `json:"retired_at,omitempty"`
}

// Issuer is a token-issuing authority Keyway tracks.
type Issuer struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Type               IssuerType      `json:"type"`
	IssuerURL          string          `json:"issuer_url"`
	JWKSURI            string          `json:"jwks_uri"`
	DiscoveryDoc       json.RawMessage `json:"discovery_doc,omitempty"`
	Keys               []Key           `json:"keys"`
	ControlsPrivateKey bool            `json:"controls_private_key"`
	ClaimSchema        []ClaimObs      `json:"claim_schema"`
}

// ClaimObs records claims observed in tokens issued by this issuer.
type ClaimObs struct {
	Name         string    `json:"name"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	PresenceRate float64   `json:"presence_rate"` // 0..1 across sampled tokens
}
