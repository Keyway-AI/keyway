// Package postgres implements cloud.Store on PostgreSQL via pgx, the drop-in
// persistence backend for hosting Keyway Cloud (the in-memory store is dev-only
// and loses data on restart). Every method is tenant-scoped exactly as the
// interface requires: project reads filter by owner, analysis reads filter by
// project, and a project delete cascades to its analyses.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Keyway-AI/keyway/cloud"
)

// Aliases for the cloud model types so method bodies read cleanly; these are
// identical types, so *Store still satisfies cloud.Store.
type (
	User     = cloud.User
	Project  = cloud.Project
	Analysis = cloud.Analysis
)

// Store is the PostgreSQL-backed implementation of cloud.Store.
type Store struct {
	pool *pgxpool.Pool
}

var _ cloud.Store = (*Store)(nil)

// Open connects to Postgres and verifies connectivity. Run MigrateUp first.
func Open(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, errors.New("cloud/postgres: empty DSN")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("cloud/postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("cloud/postgres: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the connection pool.
func (s *Store) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

// --- accounts ----------------------------------------------------------------

func (s *Store) UpsertUser(ctx context.Context, u User) error {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}
	// created_at is set only on insert so it is preserved across repeated logins.
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cloud_users (id, login, name, avatar_url, email, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			login = EXCLUDED.login,
			name = EXCLUDED.name,
			avatar_url = EXCLUDED.avatar_url,
			email = EXCLUDED.email`,
		u.ID, u.Login, u.Name, u.AvatarURL, u.Email, u.CreatedAt)
	if err != nil {
		return fmt.Errorf("cloud/postgres: upsert user: %w", err)
	}
	return nil
}

func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, login, name, avatar_url, email, created_at
		FROM cloud_users WHERE id = $1`, id).
		Scan(&u.ID, &u.Login, &u.Name, &u.AvatarURL, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, cloud.ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("cloud/postgres: get user: %w", err)
	}
	return u, nil
}

// --- projects ----------------------------------------------------------------

func (s *Store) CreateProject(ctx context.Context, p Project) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	source, err := json.Marshal(p.Source)
	if err != nil {
		return fmt.Errorf("cloud/postgres: marshal source: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO cloud_projects (id, owner_id, name, source, created_at, latest_analysis_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.OwnerID, p.Name, source, p.CreatedAt, p.LatestAnalysisID)
	if err != nil {
		return fmt.Errorf("cloud/postgres: create project: %w", err)
	}
	return nil
}

func (s *Store) ListProjects(ctx context.Context, ownerID string) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, owner_id, name, source, created_at, latest_analysis_id
		FROM cloud_projects WHERE owner_id = $1 ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("cloud/postgres: list projects: %w", err)
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		p, serr := scanProject(rows)
		if serr != nil {
			return nil, serr
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProject(ctx context.Context, id string) (Project, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, owner_id, name, source, created_at, latest_analysis_id
		FROM cloud_projects WHERE id = $1`, id)
	p, err := scanProject(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, cloud.ErrNotFound
	}
	if err != nil {
		return Project{}, fmt.Errorf("cloud/postgres: get project: %w", err)
	}
	return p, nil
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	// Analyses cascade via the FK ON DELETE CASCADE.
	tag, err := s.pool.Exec(ctx, `DELETE FROM cloud_projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("cloud/postgres: delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return cloud.ErrNotFound
	}
	return nil
}

// scannable is satisfied by both pgx.Row and pgx.Rows.
type scannable interface{ Scan(dest ...any) error }

func scanProject(row scannable) (Project, error) {
	var (
		p      Project
		source []byte
	)
	if err := row.Scan(&p.ID, &p.OwnerID, &p.Name, &source, &p.CreatedAt, &p.LatestAnalysisID); err != nil {
		return Project{}, err
	}
	if len(source) > 0 {
		if err := json.Unmarshal(source, &p.Source); err != nil {
			return Project{}, fmt.Errorf("cloud/postgres: unmarshal source: %w", err)
		}
	}
	return p, nil
}

// --- analyses ----------------------------------------------------------------

func (s *Store) SaveAnalysis(ctx context.Context, a Analysis) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("cloud/postgres: marshal analysis: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cloud/postgres: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful Commit
	_, err = tx.Exec(ctx, `
		INSERT INTO cloud_analyses
			(id, project_id, created_at, trigger_kind, trigger_ref, hash, is_baseline,
			 consumer_count, issuer_count, change_count, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			created_at = EXCLUDED.created_at, trigger_kind = EXCLUDED.trigger_kind,
			trigger_ref = EXCLUDED.trigger_ref, hash = EXCLUDED.hash,
			is_baseline = EXCLUDED.is_baseline, consumer_count = EXCLUDED.consumer_count,
			issuer_count = EXCLUDED.issuer_count, change_count = EXCLUDED.change_count,
			data = EXCLUDED.data`,
		a.ID, a.ProjectID, a.CreatedAt, defaultStr(a.TriggerKind, "manual"), a.TriggerRef,
		a.Hash, a.IsBaseline, a.ConsumerCount, a.IssuerCount, a.ChangeCount, data)
	if err != nil {
		return fmt.Errorf("cloud/postgres: save analysis: %w", err)
	}
	// Point the project at this analysis as its latest (matches the memory store).
	if _, err := tx.Exec(ctx,
		`UPDATE cloud_projects SET latest_analysis_id = $1 WHERE id = $2`, a.ID, a.ProjectID); err != nil {
		return fmt.Errorf("cloud/postgres: update latest analysis: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("cloud/postgres: commit: %w", err)
	}
	return nil
}

func (s *Store) ListAnalyses(ctx context.Context, projectID string, limit int) ([]Analysis, error) {
	q := `SELECT data FROM cloud_analyses WHERE project_id = $1 ORDER BY created_at DESC`
	args := []any{projectID}
	if limit > 0 {
		q += ` LIMIT $2`
		args = append(args, limit)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("cloud/postgres: list analyses: %w", err)
	}
	defer rows.Close()
	var out []Analysis
	for rows.Next() {
		a, serr := scanAnalysis(rows)
		if serr != nil {
			return nil, serr
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAnalysis(ctx context.Context, id string) (Analysis, error) {
	a, err := scanAnalysis(s.pool.QueryRow(ctx, `SELECT data FROM cloud_analyses WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Analysis{}, cloud.ErrNotFound
	}
	if err != nil {
		return Analysis{}, fmt.Errorf("cloud/postgres: get analysis: %w", err)
	}
	return a, nil
}

func (s *Store) LatestAnalysis(ctx context.Context, projectID string) (Analysis, error) {
	a, err := scanAnalysis(s.pool.QueryRow(ctx,
		`SELECT data FROM cloud_analyses WHERE project_id = $1 ORDER BY created_at DESC LIMIT 1`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Analysis{}, cloud.ErrNotFound
	}
	if err != nil {
		return Analysis{}, fmt.Errorf("cloud/postgres: latest analysis: %w", err)
	}
	return a, nil
}

func scanAnalysis(row scannable) (Analysis, error) {
	var data []byte
	if err := row.Scan(&data); err != nil {
		return Analysis{}, err
	}
	var a Analysis
	if err := json.Unmarshal(data, &a); err != nil {
		return Analysis{}, fmt.Errorf("cloud/postgres: unmarshal analysis: %w", err)
	}
	return a, nil
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
