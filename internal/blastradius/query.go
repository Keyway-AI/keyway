// Package blastradius answers "if I make this change, who breaks?" and derives
// a safe grace period (PRD §10).
package blastradius

import (
	"time"

	"github.com/architsharma/keyway/internal/model"
)

// Proposal kinds.
const (
	KindRotateKey     = "rotate_key"
	KindRemoveClaim   = "remove_claim"
	KindChangeIssuer  = "change_issuer"
	KindDropAlgorithm = "drop_algorithm"
	KindRetireKey     = "retire_key"
)

// Verdicts.
const (
	VerdictWillBreak = "will_break"
	VerdictReady     = "ready"
	VerdictUnknown   = "unknown"
)

// ChangeProposal is a proposed change to evaluate (PRD §10.1).
type ChangeProposal struct {
	Kind         string
	IssuerID     string
	KID          string // rotate_key, retire_key
	ClaimName    string // remove_claim
	NewIssuerURL string // change_issuer
	Algorithm    string // drop_algorithm
}

// AffectedConsumer is one consumer's verdict under a proposal.
type AffectedConsumer struct {
	Consumer   model.Consumer
	Verdict    string // will_break | ready | unknown
	Reason     string
	Evidence   []string
	Confidence float64
}

// BlastRadiusResult is the full answer (PRD §10.1).
type BlastRadiusResult struct {
	Proposal               ChangeProposal
	Affected               []AffectedConsumer
	Unknown                []model.Consumer
	RecommendedGracePeriod time.Duration
	GraceBasis             string // StableID of the bounding consumer
	GeneratedAt            time.Time
}

// Resolve computes the blast radius of a proposal against a contract version and
// its probe history.
//
// TODO(M7): implement the four resolution algorithms (PRD §10.2) and the grace
// period calculation (PRD §10.3, see graceperiod.go). Must return in <10s on a
// 50-consumer graph (AC-9).
func Resolve(v model.ContractVersion, p ChangeProposal, history map[string][]model.ProbeResult) (BlastRadiusResult, error) {
	_ = v
	_ = history
	return BlastRadiusResult{Proposal: p}, model.ErrUnsupported
}
