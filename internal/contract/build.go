package contract

import (
	"time"

	"github.com/google/uuid"
	"github.com/nometria/keyway/internal/model"
)

// BuildInput is the raw material for assembling a contract version.
type BuildInput struct {
	Issuers     []model.Issuer
	Consumers   []model.Consumer
	Edges       []model.Edge
	TriggerKind string // scheduled|deploy|commit|manual
	TriggerRef  string
	Now         time.Time
}

// Build assembles a ContractVersion from discovery output and computes its
// canonical hash. It assigns a fresh volatile ID and CreatedAt but leaves the
// baseline decision to Snapshot (version.go).
//
// TODO(M4): richer edge derivation (currently edges are passed through). The
// canonical hash and assembly are complete.
func Build(in BuildInput) model.ContractVersion {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	v := model.ContractVersion{
		ID:          uuid.NewString(),
		CreatedAt:   now,
		Issuers:     in.Issuers,
		Consumers:   in.Consumers,
		Edges:       in.Edges,
		TriggerKind: in.TriggerKind,
		TriggerRef:  in.TriggerRef,
	}
	v.Hash = Hash(v)
	return v
}
