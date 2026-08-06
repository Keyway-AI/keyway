// Package cloud is the multi-tenant hosted layer ("Keyway Cloud") on top of the
// open-source engine. It adds accounts, projects, and persisted analysis history
// around the same discovery/contract/diff/threat logic the CLI and self-hosted
// server use — so the static, config-driven half of Keyway (auth-contract
// discovery, drift, threat coverage) can run as a SaaS on repos users connect or
// upload. The live half (probing, canary, blast radius) stays self-hosted by
// design; the cloud never handles a customer's signing keys or staging traffic.
package cloud

import (
	"time"

	"github.com/Keyway-AI/keyway/internal/model"
)

// User is an authenticated account (currently always backed by a GitHub login).
type User struct {
	ID        string    `json:"id"` // stable internal id, e.g. "gh:42" (provider:providerID)
	Login     string    `json:"login"`
	Name      string    `json:"name,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// SourceKind is where a project's auth config comes from.
type SourceKind string

const (
	SourceUpload SourceKind = "upload" // config files uploaded through the UI/API
	SourceGitHub SourceKind = "github" // fetched from a connected GitHub repository
)

// Source describes where a project reads its auth configuration from.
type Source struct {
	Kind SourceKind `json:"kind"`
	Repo string     `json:"repo,omitempty"` // "owner/name" for github sources
	Ref  string     `json:"ref,omitempty"`  // branch or tag (default "main")
	Path string     `json:"path,omitempty"` // optional subdirectory within the repo
}

// Project is one tracked auth surface (a repo or an uploaded config set) owned by
// a user. It is the multi-tenant boundary: every project belongs to exactly one
// OwnerID, and all reads/writes are scoped to the requesting user's projects.
type Project struct {
	ID               string    `json:"id"`
	OwnerID          string    `json:"owner_id"`
	Name             string    `json:"name"`
	Source           Source    `json:"source"`
	CreatedAt        time.Time `json:"created_at"`
	LatestAnalysisID string    `json:"latest_analysis_id,omitempty"`
}

// Analysis is a single run over a project's config: the derived contract plus the
// drift (change events) versus the project's previous analysis. Stored so a
// project accrues history over time.
type Analysis struct {
	ID          string                `json:"id"`
	ProjectID   string                `json:"project_id"`
	CreatedAt   time.Time             `json:"created_at"`
	TriggerKind string                `json:"trigger_kind"` // upload | sync | manual
	TriggerRef  string                `json:"trigger_ref,omitempty"`
	Hash        string                `json:"hash"`
	IsBaseline  bool                  `json:"is_baseline"`
	Version     model.ContractVersion `json:"version"`
	Changes     []model.ChangeEvent   `json:"changes"`
	// Denormalized summary for cheap list rendering.
	ConsumerCount int `json:"consumer_count"`
	IssuerCount   int `json:"issuer_count"`
	ChangeCount   int `json:"change_count"`
}

// AnalysisSummary is the compact form used in list/history views.
type AnalysisSummary struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TriggerKind   string    `json:"trigger_kind"`
	TriggerRef    string    `json:"trigger_ref,omitempty"`
	Hash          string    `json:"hash"`
	IsBaseline    bool      `json:"is_baseline"`
	ConsumerCount int       `json:"consumer_count"`
	IssuerCount   int       `json:"issuer_count"`
	ChangeCount   int       `json:"change_count"`
}

// Summary projects an Analysis to its compact form.
func (a Analysis) Summary() AnalysisSummary {
	return AnalysisSummary{
		ID: a.ID, CreatedAt: a.CreatedAt, TriggerKind: a.TriggerKind, TriggerRef: a.TriggerRef,
		Hash: a.Hash, IsBaseline: a.IsBaseline,
		ConsumerCount: a.ConsumerCount, IssuerCount: a.IssuerCount, ChangeCount: a.ChangeCount,
	}
}
