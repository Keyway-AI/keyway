// Package postgres implements store.Store on PostgreSQL via pgx (PRD §2).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/store"
)

// Store is the PostgreSQL-backed implementation of store.Store.
type Store struct {
	dsn  string
	pool *pgxpool.Pool
}

// Open connects to Postgres and verifies connectivity.
func Open(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, errors.New("postgres: empty DSN")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &Store{dsn: dsn, pool: pool}, nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// DSN returns the configured connection string.
func (s *Store) DSN() string { return s.dsn }

var _ store.Store = (*Store)(nil)

// --- contract versions ------------------------------------------------------

func (s *Store) SaveContractVersion(ctx context.Context, v model.ContractVersion) error {
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("postgres: marshal version: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO contract_versions (id, hash, created_at, is_baseline, trigger_kind, trigger_ref, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		v.ID, v.Hash, v.CreatedAt, v.IsBaseline, defaultStr(v.TriggerKind, "manual"), nullable(v.TriggerRef), string(data))
	if err != nil {
		return fmt.Errorf("postgres: save version: %w", err)
	}
	return nil
}

func (s *Store) GetContractVersion(ctx context.Context, id string) (model.ContractVersion, error) {
	return s.scanVersion(ctx, `SELECT data FROM contract_versions WHERE id = $1`, id)
}

func (s *Store) LatestVersion(ctx context.Context) (model.ContractVersion, error) {
	return s.scanVersion(ctx, `SELECT data FROM contract_versions ORDER BY created_at DESC LIMIT 1`)
}

func (s *Store) BaselineVersion(ctx context.Context) (model.ContractVersion, error) {
	return s.scanVersion(ctx, `SELECT data FROM contract_versions WHERE is_baseline ORDER BY created_at ASC LIMIT 1`)
}

func (s *Store) scanVersion(ctx context.Context, query string, args ...any) (model.ContractVersion, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, query, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ContractVersion{}, model.ErrNotFound
	}
	if err != nil {
		return model.ContractVersion{}, fmt.Errorf("postgres: query version: %w", err)
	}
	var v model.ContractVersion
	if err := json.Unmarshal(data, &v); err != nil {
		return model.ContractVersion{}, fmt.Errorf("postgres: unmarshal version: %w", err)
	}
	return v, nil
}

// --- change events ----------------------------------------------------------

func (s *Store) SaveChangeEvents(ctx context.Context, events []model.ChangeEvent) error {
	if len(events) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, e := range events {
		oldV := mustJSON(e.OldValue)
		newV := mustJSON(e.NewValue)
		evidence := mustJSON(e.Evidence)
		var attribution any
		if e.Attribution != nil {
			attribution = mustJSON(e.Attribution)
		}
		detected := e.DetectedAt
		if detected.IsZero() {
			detected = time.Now().UTC()
		}
		batch.Queue(`
			INSERT INTO change_events
			  (id, from_version, to_version, consumer_id, field, old_value, new_value,
			   class, severity, confidence, evidence, attribution, detected_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			e.ID, e.FromVersion, e.ToVersion, e.ConsumerID, e.Field, oldV, newV,
			string(e.Class), string(e.Severity), e.Confidence, evidence, attribution, detected)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range events {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("postgres: save change events: %w", err)
		}
	}
	return nil
}

func (s *Store) ListChangeEvents(ctx context.Context, since time.Time) ([]model.ChangeEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, from_version, to_version, consumer_id, field, old_value, new_value,
		       class, severity, confidence, evidence, attribution, detected_at
		FROM change_events
		WHERE detected_at >= $1
		ORDER BY detected_at DESC`, since)
	if err != nil {
		return nil, fmt.Errorf("postgres: list change events: %w", err)
	}
	defer rows.Close()

	var out []model.ChangeEvent
	for rows.Next() {
		var (
			e                            model.ChangeEvent
			class, severity              string
			oldV, newV, evidence, attrib []byte
		)
		if err := rows.Scan(&e.ID, &e.FromVersion, &e.ToVersion, &e.ConsumerID, &e.Field,
			&oldV, &newV, &class, &severity, &e.Confidence, &evidence, &attrib, &e.DetectedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan change event: %w", err)
		}
		e.Class = model.ChangeClass(class)
		e.Severity = model.Severity(severity)
		_ = json.Unmarshal(oldV, &e.OldValue)
		_ = json.Unmarshal(newV, &e.NewValue)
		_ = json.Unmarshal(evidence, &e.Evidence)
		if len(attrib) > 0 {
			var a model.Attribution
			if json.Unmarshal(attrib, &a) == nil {
				e.Attribution = &a
			}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// --- probe results ----------------------------------------------------------

func (s *Store) SaveProbeResults(ctx context.Context, results []model.ProbeResult) error {
	if len(results) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, r := range results {
		runAt := r.RunAt
		if runAt.IsZero() {
			runAt = time.Now().UTC()
		}
		batch.Queue(`
			INSERT INTO probe_results
			  (id, probe_id, consumer_id, endpoint_url, status_code, latency_ms, passed, raw_response, run_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			r.ID, r.ProbeID, r.ConsumerID, r.EndpointURL, r.StatusCode, r.LatencyMs, r.Passed, truncate(r.RawResponse, 512), runAt)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range results {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("postgres: save probe results: %w", err)
		}
	}
	return nil
}

func (s *Store) ProbeHistory(ctx context.Context, consumerID string, limit int) ([]model.ProbeResult, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, probe_id, consumer_id, endpoint_url, status_code, latency_ms, passed, raw_response, run_at
		FROM probe_results
		WHERE consumer_id = $1
		ORDER BY run_at DESC
		LIMIT $2`, consumerID, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: probe history: %w", err)
	}
	defer rows.Close()

	var out []model.ProbeResult
	for rows.Next() {
		var r model.ProbeResult
		if err := rows.Scan(&r.ID, &r.ProbeID, &r.ConsumerID, &r.EndpointURL, &r.StatusCode,
			&r.LatencyMs, &r.Passed, &r.RawResponse, &r.RunAt); err != nil {
			return nil, fmt.Errorf("postgres: scan probe result: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- helpers ----------------------------------------------------------------

func mustJSON(v any) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
