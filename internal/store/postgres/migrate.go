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

// migrator builds a migrate.Migrate over the embedded SQL and the given DSN.
// golang-migrate's pgx/v5 driver registers the "pgx5" scheme, so a standard
// postgres:// URL is rewritten accordingly.
func migrator(dsn string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("postgres: open embedded migrations: %w", err)
	}
	url := dsn
	for _, scheme := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(url, scheme) {
			url = "pgx5://" + strings.TrimPrefix(url, scheme)
			break
		}
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return nil, fmt.Errorf("postgres: init migrator: %w", err)
	}
	return m, nil
}

// MigrateUp applies all pending migrations. ErrNoChange is treated as success.
func MigrateUp(dsn string) error {
	m, err := migrator(dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back the most recent migration.
func MigrateDown(dsn string) error {
	m, err := migrator(dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: migrate down: %w", err)
	}
	return nil
}
