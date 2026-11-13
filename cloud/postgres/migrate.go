package postgres

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" scheme
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migrator builds a migrate.Migrate over the embedded cloud SQL and the given
// DSN. golang-migrate's pgx/v5 driver registers the "pgx5" scheme, so a standard
// postgres:// URL is rewritten accordingly (same convention as internal/store).
func migrator(dsn string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("cloud/postgres: open embedded migrations: %w", err)
	}
	url := dsn
	for _, scheme := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(url, scheme) {
			url = "pgx5://" + strings.TrimPrefix(url, scheme)
			break
		}
	}
	// Use a cloud-specific migrations table so pointing the cloud layer at the
	// same database as the self-hosted engine (which uses the default
	// schema_migrations) never clashes on migration state.
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	url += sep + "x-migrations-table=schema_migrations_cloud"
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return nil, fmt.Errorf("cloud/postgres: init migrator: %w", err)
	}
	return m, nil
}

// MigrateUp applies all pending cloud migrations. ErrNoChange is success.
func MigrateUp(dsn string) error {
	m, err := migrator(dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("cloud/postgres: migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back the most recent cloud migration.
func MigrateDown(dsn string) error {
	m, err := migrator(dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("cloud/postgres: migrate down: %w", err)
	}
	return nil
}
