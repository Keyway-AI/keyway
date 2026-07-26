package coordination

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// schedulerLockID is the fixed key for the scheduler-leadership advisory lock.
// Any int64 works as long as it is stable across replicas; this is derived from
// "keyway.scheduler" and hard-coded so all replicas contend for the same lock.
const schedulerLockID int64 = 0x6b657977_6179_5f31 // "keyw" "ay_1"

// Open builds a Coordinator for the DSN plus a cleanup that Close() invokes. A
// "memory" DSN yields the single-node in-memory coordinator; anything else is a
// Postgres DSN whose seams are shared across replicas.
func Open(ctx context.Context, dsn string, idemTTL time.Duration) (*Coordinator, error) {
	if dsn == "" || dsn == "memory" || strings.HasPrefix(dsn, "memory:") {
		return NewMemory(idemTTL), nil
	}
	// A small dedicated pool: the leader holds one connection for the lifetime of
	// its lock, and idempotency reads/writes are short. Cap low so coordination
	// never starves the main store's pool.
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("coordination: parse dsn: %w", err)
	}
	cfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("coordination: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("coordination: ping: %w", err)
	}
	return &Coordinator{
		idem:   &pgIdempotency{pool: pool, ttl: idemTTL},
		leader: &pgLeader{pool: pool, lockID: schedulerLockID},
		pool:   pool,
		closes: []func(){pool.Close},
	}, nil
}

// --- Postgres idempotency ---------------------------------------------------

type pgIdempotency struct {
	pool *pgxpool.Pool
	ttl  time.Duration
}

func (s *pgIdempotency) Get(ctx context.Context, key string) (Record, bool, error) {
	var rec Record
	err := s.pool.QueryRow(ctx,
		`SELECT status, body, content_type FROM keyway_idempotency
		 WHERE key = $1 AND expires_at > now()`, key,
	).Scan(&rec.Status, &rec.Body, &rec.ContentType)
	if err != nil {
		if isNoRows(err) {
			return Record{}, false, nil
		}
		return Record{}, false, fmt.Errorf("coordination: idem get: %w", err)
	}
	return rec, true, nil
}

func (s *pgIdempotency) Put(ctx context.Context, key string, rec Record) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO keyway_idempotency (key, status, body, content_type, expires_at)
		 VALUES ($1, $2, $3, $4, now() + $5::interval)
		 ON CONFLICT (key) DO UPDATE
		   SET status = EXCLUDED.status, body = EXCLUDED.body,
		       content_type = EXCLUDED.content_type, expires_at = EXCLUDED.expires_at`,
		key, rec.Status, rec.Body, rec.ContentType, s.ttl.String(),
	)
	if err != nil {
		return fmt.Errorf("coordination: idem put: %w", err)
	}
	return nil
}

// --- Postgres leader (session-level advisory lock) --------------------------

// pgLeader holds a session-scoped advisory lock on a dedicated connection while
// it is leader. If the process dies, the connection drops and Postgres releases
// the lock automatically, so a standby's next IsLeader acquires it.
type pgLeader struct {
	pool   *pgxpool.Pool
	lockID int64

	mu   sync.Mutex
	conn *pgxpool.Conn // non-nil only while leadership is held
}

func (l *pgLeader) IsLeader(ctx context.Context) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Already leader: confirm the held connection is still alive.
	if l.conn != nil {
		if err := l.conn.Ping(ctx); err == nil {
			return true
		}
		// Lost the connection (and thus the lock); drop it and re-contend below.
		l.conn.Release()
		l.conn = nil
	}

	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return false
	}
	var got bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, l.lockID).Scan(&got); err != nil || !got {
		conn.Release()
		return false
	}
	l.conn = conn // hold the connection (and the lock) until we lose or Close
	return true
}

func (l *pgLeader) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conn == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = l.conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, l.lockID)
	l.conn.Release()
	l.conn = nil
	return nil
}

// isNoRows reports whether err is pgx's "no rows" sentinel.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
