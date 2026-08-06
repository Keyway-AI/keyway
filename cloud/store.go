package cloud

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a requested record does not exist (or is not
// visible to the caller — callers scope by owner before loading).
var ErrNotFound = errors.New("not found")

// Store is the persistence seam for the cloud layer. The in-memory implementation
// backs local/dev; a Postgres implementation (same interface) is the drop-in for
// hosting — every method is already tenant-scoped by OwnerID/ProjectID so no
// query leaks across tenants.
type Store interface {
	// Accounts.
	UpsertUser(ctx context.Context, u User) error
	GetUser(ctx context.Context, id string) (User, error)

	// Projects (tenant boundary: always filter by owner).
	CreateProject(ctx context.Context, p Project) error
	ListProjects(ctx context.Context, ownerID string) ([]Project, error)
	GetProject(ctx context.Context, id string) (Project, error)
	DeleteProject(ctx context.Context, id string) error

	// Analyses.
	SaveAnalysis(ctx context.Context, a Analysis) error
	ListAnalyses(ctx context.Context, projectID string, limit int) ([]Analysis, error) // newest first
	GetAnalysis(ctx context.Context, id string) (Analysis, error)
	LatestAnalysis(ctx context.Context, projectID string) (Analysis, error)

	Close() error
}
