// Package store defines the persistence interface for Keyway and provides
// implementations (see store/postgres).
package store

import (
	"context"
	"time"

	"github.com/nometria/keyway/internal/model"
)

// Store is the persistence boundary. All implementations must be safe for
// concurrent use.
type Store interface {
	SaveContractVersion(ctx context.Context, v model.ContractVersion) error
	GetContractVersion(ctx context.Context, id string) (model.ContractVersion, error)
	LatestVersion(ctx context.Context) (model.ContractVersion, error)
	BaselineVersion(ctx context.Context) (model.ContractVersion, error)

	SaveChangeEvents(ctx context.Context, events []model.ChangeEvent) error
	ListChangeEvents(ctx context.Context, since time.Time) ([]model.ChangeEvent, error)

	SaveProbeResults(ctx context.Context, results []model.ProbeResult) error
	ProbeHistory(ctx context.Context, consumerID string, limit int) ([]model.ProbeResult, error)
}
