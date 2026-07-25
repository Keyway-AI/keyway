package model

import "time"

// ConsumerKind classifies a token-validating component.
type ConsumerKind string

const (
	ConsumerService      ConsumerKind = "service"
	ConsumerGatewayRoute ConsumerKind = "gateway_route"
	ConsumerEdgeFunction ConsumerKind = "edge_function"
	ConsumerClient       ConsumerKind = "client" // mobile/SPA — not probeable
)

// Expectations is what a consumer requires of a token to accept it.
type Expectations struct {
	Issuers        []string `json:"issuers"`
	Audiences      []string `json:"audiences"`
	Algorithms     []string `json:"algorithms"`
	RequiredClaims []string `json:"required_claims"`
	ClockSkewSec   int      `json:"clock_skew_sec"`
}

// BehaviorSource records how a piece of behaviour was determined, in ascending
// order of confidence: config < library_default < observed < probed.
type BehaviorSource string

const (
	SrcConfig         BehaviorSource = "config"
	SrcLibraryDefault BehaviorSource = "library_default"
	SrcObserved       BehaviorSource = "observed"
	SrcProbed         BehaviorSource = "probed"
)

// JWKSBehavior captures how a consumer fetches and caches keys — the mechanism
// behind most rotation outages.
type JWKSBehavior struct {
	// JWKSURI is the endpoint the consumer fetches signing keys from (the
	// rotation endpoint). Captured from Istio jwtRules.jwksUri / Envoy remote_jwks.
	JWKSURI               string         `json:"jwks_uri,omitempty"`
	CacheTTLSec           *int           `json:"cache_ttl_sec,omitempty"`
	RefreshIntervalSec    *int           `json:"refresh_interval_sec,omitempty"`
	RefreshesOnUnknownKID *bool          `json:"refreshes_on_unknown_kid,omitempty"`
	LastObservedRefresh   *time.Time     `json:"last_observed_refresh,omitempty"`
	Source                BehaviorSource `json:"source"`
}

// LibraryInfo identifies the JWT library a consumer uses.
type LibraryInfo struct {
	Name    string `json:"name"`    // e.g. "MicahParks/keyfunc"
	Version string `json:"version"` // e.g. "v1.9.0"
	Lang    string `json:"lang"`
}

// Endpoint is a probeable HTTP target for a consumer.
type Endpoint struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	// SafeProbePath is a request known to succeed with a valid token. The probe target.
	SafeProbePath string `json:"safe_probe_path"`
}

// ProvenanceRecord ties a piece of derived state back to its source of evidence.
type ProvenanceRecord struct {
	Source     string    `json:"source"`  // "istio:RequestAuthentication/foo"
	Locator    string    `json:"locator"` // file path, resource ref, or URL
	ObservedAt time.Time `json:"observed_at"`
	Confidence float64   `json:"confidence"` // 0..1
}

// Consumer is a component that validates tokens. Derived automatically — Keyway
// never asks the user to author these.
type Consumer struct {
	ID       string `json:"id"`
	StableID string `json:"stable_id"`
	// Aliases are alternative stable identities for the SAME workload discovered
	// by a different source (e.g. Kubernetes keys a workload by its service
	// account while Istio keys it by service name). The aggregator merges two
	// consumers whose identity sets — {StableID} ∪ Aliases — intersect, so a
	// single logical service seen two ways becomes one record (KI-28).
	Aliases      []string                      `json:"aliases,omitempty"`
	Kind         ConsumerKind                  `json:"kind"`
	Name         string                        `json:"name"`
	Namespace    string                        `json:"namespace,omitempty"`
	OwnerTeam    string                        `json:"owner_team,omitempty"`
	Endpoints    []Endpoint                    `json:"endpoints"`
	Expects      Expectations                  `json:"expects"`
	JWKSBehavior JWKSBehavior                  `json:"jwks_behavior"`
	Library      *LibraryInfo                  `json:"library,omitempty"`
	Provenance   map[string][]ProvenanceRecord `json:"provenance"`
	Confidence   map[string]float64            `json:"confidence"`
	Probeable    bool                          `json:"probeable"`
}
