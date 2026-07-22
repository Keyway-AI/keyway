// Package postgres implements store.Store on PostgreSQL via pgx (PRD §2).
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/architsharma/keyway/internal/model"
	"github.com/architsharma/keyway/internal/store"
)

// Store is the PostgreSQL-backed implementation of store.Store.
type Store struct {
	dsn string
	// TODO(M1): pool *pgxpool.Pool
}

// Open connects to Postgres. TODO(M1): establish a pgxpool and ping.
func Open(ctx context.Context, dsn string) (*Store, error) {
	_ = ctx
	if dsn == "" {
		return nil, errors.New("postgres: empty DSN")
	}
	return &Store{dsn: dsn}, nil
}

// DSN returns the configured connection string.
func (s *Store) DSN() string { return s.dsn }

// Close releases the connection pool.
func (s *Store) Close() {}

var _ store.Store = (*Store)(nil)

// The methods below are wired to SQL in M1 (see migrations/ and PROGRESS.md).

func (s *Store) SaveContractVersion(context.Context, model.ContractVersion) error {
	return model.ErrUnsupported
}
func (s *Store) GetContractVersion(context.Context, string) (model.ContractVersion, error) {
	return model.ContractVersion{}, model.ErrNotFound
}
func (s *Store) LatestVersion(context.Context) (model.ContractVersion, error) {
	return model.ContractVersion{}, model.ErrNotFound
}
func (s *Store) BaselineVersion(context.Context) (model.ContractVersion, error) {
	return model.ContractVersion{}, model.ErrNotFound
}
func (s *Store) SaveChangeEvents(context.Context, []model.ChangeEvent) error {
	return model.ErrUnsupported
}
func (s *Store) ListChangeEvents(context.Context, time.Time) ([]model.ChangeEvent, error) {
	return nil, model.ErrUnsupported
}
func (s *Store) SaveProbeResults(context.Context, []model.ProbeResult) error {
	return model.ErrUnsupported
}
func (s *Store) ProbeHistory(context.Context, string, int) ([]model.ProbeResult, error) {
	return nil, model.ErrUnsupported
}
