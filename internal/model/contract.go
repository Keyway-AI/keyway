package model

import "time"

// EdgeState is the verification status of an issuer→consumer relationship.
type EdgeState string

const (
	EdgeVerified   EdgeState = "verified"   // all applicable probes passed
	EdgeDivergent  EdgeState = "divergent"  // probe result contradicts derived contract
	EdgeUnverified EdgeState = "unverified" // not probeable or not yet probed
)

// Edge is a directed relationship: a consumer validates tokens from an issuer.
type Edge struct {
	IssuerID     string       `json:"issuer_id"`
	ConsumerID   string       `json:"consumer_id"`
	Expects      Expectations `json:"expects"`
	LastVerified *time.Time   `json:"last_verified,omitempty"`
	VerifyState  EdgeState    `json:"verify_state"`
}

// ContractVersion is an immutable snapshot of the whole derived contract graph.
type ContractVersion struct {
	ID          string     `json:"id"`
	Hash        string     `json:"hash"` // sha256 of canonical form
	CreatedAt   time.Time  `json:"created_at"`
	IsBaseline  bool       `json:"is_baseline"`
	Issuers     []Issuer   `json:"issuers"`
	Consumers   []Consumer `json:"consumers"`
	Edges       []Edge     `json:"edges"`
	TriggerKind string     `json:"trigger_kind"` // scheduled|deploy|commit|manual
	TriggerRef  string     `json:"trigger_ref,omitempty"`
}
