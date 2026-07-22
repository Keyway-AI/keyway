package model

import "time"

// ChangeClass is the semantic direction of a contract change.
type ChangeClass string

const (
	ChangeWidened  ChangeClass = "widened" // now accepts what it previously rejected
	ChangeNarrowed ChangeClass = "narrowed"
	ChangeNeutral  ChangeClass = "neutral"
	ChangeUnknown  ChangeClass = "unknown"
)

// ChangeEvent is a single field-level change between two contract versions.
type ChangeEvent struct {
	ID          string       `json:"id"`
	FromVersion string       `json:"from_version"`
	ToVersion   string       `json:"to_version"`
	ConsumerID  string       `json:"consumer_id"`
	Field       string       `json:"field"` // dotted path e.g. "expects.audiences"
	OldValue    any          `json:"old_value"`
	NewValue    any          `json:"new_value"`
	Class       ChangeClass  `json:"class"`
	Severity    Severity     `json:"severity"`
	Confidence  float64      `json:"confidence"`
	Evidence    []string     `json:"evidence"` // probe IDs or provenance locators
	Attribution *Attribution `json:"attribution,omitempty"`
	DetectedAt  time.Time    `json:"detected_at"`
}

// Attribution binds a change to who/what caused it.
type Attribution struct {
	Kind       string    `json:"kind"` // commit|pr|deploy|idp_audit|unattributed
	Ref        string    `json:"ref"`
	Actor      string    `json:"actor,omitempty"`
	Team       string    `json:"team,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
}
