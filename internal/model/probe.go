package model

import "time"

// Severity ranks the operational risk of a finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ProbeResult is the outcome of running one probe against one endpoint.
// Note: minted tokens are NEVER stored here — only the jti and probe ID
// identify a run (PRD OPEN-4).
type ProbeResult struct {
	ID          string    `json:"id"`
	ProbeID     string    `json:"probe_id"`
	ConsumerID  string    `json:"consumer_id"`
	EndpointURL string    `json:"endpoint_url"`
	StatusCode  int       `json:"status_code"`
	LatencyMs   int       `json:"latency_ms"`
	Passed      bool      `json:"passed"`
	RawResponse string    `json:"raw_response"` // truncated to 512 bytes
	RunAt       time.Time `json:"run_at"`
}
