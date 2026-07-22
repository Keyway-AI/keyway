package diff

import (
	"github.com/architsharma/keyway/internal/model"
)

// Compute diffs two contract versions and returns the classified change events
// (PRD §9). Consumers are matched across versions by StableID; a consumer
// present in only one version yields a consumer_added / consumer_removed event.
//
// TODO(M4): walk expectations and jwks_behavior field-by-field, emit atomic
// changes, and run each through Classify. The classifier (classify.go) is
// already complete and tested; this walker is the remaining piece.
func Compute(from, to model.ContractVersion) []model.ChangeEvent {
	_ = from
	_ = to
	return nil
}
