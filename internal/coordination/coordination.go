// Package coordination provides the cross-process seams a multi-replica Keyway
// deployment needs: a shared idempotency store (so a retried write replays the
// same result on any replica) and a leader gate (so exactly one replica runs the
// scheduler). Each seam has an in-memory single-node implementation (the default,
// and all a single daemon needs) and a Postgres implementation that shares state
// across replicas. The interfaces are the point: turning on HA is choosing a
// different adapter, not a rewrite (architecture review W5/#6).
package coordination

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Record is a cached idempotent response.
type Record struct {
	Status      int
	Body        []byte
	ContentType string
}

// IdempotencyStore persists the response to a completed write so a retry with the
// same idempotency key replays it instead of re-executing. A Postgres-backed
// implementation shares this across replicas; the in-memory one is per-process.
type IdempotencyStore interface {
	// Get returns the stored record for key, or ok=false if absent/expired.
	Get(ctx context.Context, key string) (Record, bool, error)
	// Put stores a record for key with the store's configured TTL.
	Put(ctx context.Context, key string, rec Record) error
}

// Leader gates work that must run on exactly one replica. IsLeader reports (and,
// for durable backends, lazily acquires) leadership; it is cheap to call each
// scheduler tick. Close releases any held lock.
type Leader interface {
	IsLeader(ctx context.Context) bool
	Close() error
}

// Coordinator bundles the coordination seams and owns whatever resources back
// them (e.g. a Postgres pool). Close releases them.
type Coordinator struct {
	idem   IdempotencyStore
	leader Leader
	pool   *pgxpool.Pool // non-nil only for the Postgres coordinator
	closes []func()
}

func (c *Coordinator) Idempotency() IdempotencyStore { return c.idem }
func (c *Coordinator) Leader() Leader                { return c.leader }

// Pool returns the operational Postgres pool backing the coordinator, or nil for
// the in-memory (single-node) coordinator. It lets other operational stores that
// share the same database (e.g. the Postgres keystore) reuse one pool.
func (c *Coordinator) Pool() *pgxpool.Pool { return c.pool }

// Close releases the leader lock and any owned pool.
func (c *Coordinator) Close() {
	if c.leader != nil {
		_ = c.leader.Close()
	}
	for i := len(c.closes) - 1; i >= 0; i-- {
		c.closes[i]()
	}
}

// NewMemory returns a single-node coordinator: an in-memory idempotency store and
// a leader that is always the leader. This is the default and is all a single
// daemon needs.
func NewMemory(idemTTL time.Duration) *Coordinator {
	return &Coordinator{
		idem:   NewMemoryIdempotency(idemTTL),
		leader: localLeader{},
	}
}

// --- in-memory idempotency --------------------------------------------------

// memoryIdempotency is a TTL-bounded, hard-capped, per-process idempotency store.
type memoryIdempotency struct {
	mu    sync.Mutex
	ttl   time.Duration
	max   int
	order []string
	m     map[string]memEntry
}

type memEntry struct {
	rec     Record
	expires time.Time
}

// NewMemoryIdempotency returns an in-memory idempotency store with the given TTL
// and a hard FIFO cap (4096 entries) so a burst of distinct keys cannot exhaust
// memory.
func NewMemoryIdempotency(ttl time.Duration) IdempotencyStore {
	return &memoryIdempotency{ttl: ttl, max: 4096, m: make(map[string]memEntry)}
}

func (s *memoryIdempotency) Get(_ context.Context, key string) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[key]
	if !ok {
		return Record{}, false, nil
	}
	if time.Now().After(e.expires) {
		delete(s.m, key)
		return Record{}, false, nil
	}
	return e.rec, true, nil
}

func (s *memoryIdempotency) Put(_ context.Context, key string, rec Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.m[key]; !exists {
		s.order = append(s.order, key)
	}
	s.m[key] = memEntry{rec: rec, expires: time.Now().Add(s.ttl)}
	for len(s.order) > s.max {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.m, oldest)
	}
	return nil
}

// --- local (always-on) leader -----------------------------------------------

type localLeader struct{}

func (localLeader) IsLeader(context.Context) bool { return true }
func (localLeader) Close() error                  { return nil }

// NewLocalLeader returns a leader that always holds leadership (single node).
func NewLocalLeader() Leader { return localLeader{} }
