// Package apitypes holds the public request/response shapes for the Keyway HTTP
// API (PRD §12). These are the contract with the web UI and external clients;
// keep them stable and versioned.
package apitypes

import (
	"time"

	"github.com/nometria/keyway/internal/model"
)

// SnapshotResponse is returned by POST /v1/snapshots.
type SnapshotResponse struct {
	VersionID  string `json:"version_id"`
	Hash       string `json:"hash"`
	IsBaseline bool   `json:"is_baseline"`
}

// RunProbesRequest is the body of POST /v1/probes/run.
type RunProbesRequest struct {
	ConsumerIDs []string `json:"consumer_ids,omitempty"`
	ProbeIDs    []string `json:"probe_ids,omitempty"`
}

// RunProbesResponse acknowledges a probe run and carries its results.
type RunProbesResponse struct {
	RunID   string              `json:"run_id"`
	Results []model.ProbeResult `json:"results"`
}

// ConsumerProbesResponse is a consumer's recent probe history.
type ConsumerProbesResponse struct {
	ConsumerID string              `json:"consumer_id"`
	Results    []model.ProbeResult `json:"results"`
}

// IssuerList is the registered issuers with their live key state, so the UI can
// drive the canary lifecycle.
type IssuerList struct {
	Issuers []model.Issuer `json:"issuers"`
}

// CanaryStatusResponse reports an issuer's keys and which is the announced canary.
type CanaryStatusResponse struct {
	Issuer       string      `json:"issuer"`
	Keys         []model.Key `json:"keys"`
	AnnouncedKID string      `json:"announced_kid"`
}

// BlastRadiusRequest is the body of POST /v1/blast-radius.
type BlastRadiusRequest struct {
	Proposal ChangeProposal `json:"proposal"`
}

// ChangeProposal describes a proposed change to evaluate (PRD §10.1).
type ChangeProposal struct {
	Kind         string `json:"kind"` // rotate_key|remove_claim|change_issuer|drop_algorithm|retire_key
	IssuerID     string `json:"issuer_id"`
	KID          string `json:"kid,omitempty"`
	ClaimName    string `json:"claim_name,omitempty"`
	NewIssuerURL string `json:"new_issuer_url,omitempty"`
	Algorithm    string `json:"algorithm,omitempty"`
}

// AnnounceKeyRequest is the body of POST /v1/canary/announce.
type AnnounceKeyRequest struct {
	IssuerID string `json:"issuer_id"`
	Alg      string `json:"alg"`
}

// PromoteKeyRequest is the body of POST /v1/canary/promote.
type PromoteKeyRequest struct {
	IssuerID string `json:"issuer_id"`
	KID      string `json:"kid"`
}

// CoverageResponse summarizes how much of the contract Keyway can vouch for.
type CoverageResponse struct {
	Total         int `json:"total"`
	Resolved      int `json:"resolved"`
	LowConfidence int `json:"low_confidence"`
	Unresolved    int `json:"unresolved"`
}

// HealthResponse is returned by GET /v1/health.
type HealthResponse struct {
	Status  string    `json:"status"`
	Version string    `json:"version"`
	Time    time.Time `json:"time"`
}

// ErrorResponse is the uniform error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// ConsumerList wraps a paginated consumer listing.
type ConsumerList struct {
	Consumers []model.Consumer `json:"consumers"`
	Total     int              `json:"total"`
}

// ChangeList wraps a change-event listing.
type ChangeList struct {
	Changes []model.ChangeEvent `json:"changes"`
	Total   int                 `json:"total"`
}

// --- threat coverage (GET /v1/threats/coverage) -----------------------------

// CoverageDomain / CoverageCategory / CoverageThreat / ThreatCoverageResponse
// mirror the web client's coverage view.
type CoverageDomain struct {
	Domain  string `json:"domain"`
	Covered int    `json:"covered"`
	Total   int    `json:"total"`
	Percent int    `json:"percent"`
}

type CoverageCategory struct {
	Category string `json:"category"`
	Covered  int    `json:"covered"`
	Total    int    `json:"total"`
}

type CoverageSource struct {
	Ref string `json:"ref"`
	URL string `json:"url"`
}

type CoverageThreat struct {
	ID        string           `json:"id"`
	Domain    string           `json:"domain"`
	Category  string           `json:"category"`
	Severity  model.Severity   `json:"severity"`
	Title     string           `json:"title"`
	Invariant string           `json:"invariant"`
	Sources   []CoverageSource `json:"sources"`
	Detectors []string         `json:"detectors"`
}

type ThreatCoverageResponse struct {
	Total      int                `json:"total"`
	Covered    int                `json:"covered"`
	Gaps       int                `json:"gaps"`
	Percent    int                `json:"percent"`
	Domains    []CoverageDomain   `json:"domains"`
	Categories []CoverageCategory `json:"categories"`
	Threats    []CoverageThreat   `json:"threats"`
}

// --- agent token inspection (POST /v1/agent/inspect) ------------------------

type AgentInspectRequest struct {
	Token              string   `json:"token"`
	Audience           string   `json:"audience,omitempty"`
	RequireDelegation  bool     `json:"require_delegation,omitempty"`
	MaxLifetimeSeconds int      `json:"max_lifetime_seconds,omitempty"`
	AllowedScopes      []string `json:"allowed_scopes,omitempty"`
}

// AgentInspectResponse.Findings is []agentauth.Finding, which already carries the
// snake_case JSON tags; the transport keeps it as `any` to avoid importing the
// analyzer into this leaf package.
type AgentInspectResponse struct {
	Findings any `json:"findings"`
	Count    int `json:"count"`
}
