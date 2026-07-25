// Package memory is an in-memory store.Store implementation. It backs tests and
// offline local development (`--db memory`) so neither needs a running Postgres,
// and it documents the store contract as a small, readable reference.
package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/store"
)

// Store keeps all contract versions, change events, and probe results in memory.
// Safe for concurrent use.
type Store struct {
	mu       sync.RWMutex
	versions map[string]model.ContractVersion
	order    []string // version IDs in save order (last = latest)
	baseline string
	events   []model.ChangeEvent
	probes   []model.ProbeResult
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{versions: map[string]model.ContractVersion{}}
}

var _ store.Store = (*Store)(nil)

func (s *Store) SaveContractVersion(_ context.Context, v model.ContractVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.versions[v.ID]; !exists {
		s.order = append(s.order, v.ID)
	}
	s.versions[v.ID] = v
	if v.IsBaseline && s.baseline == "" {
		s.baseline = v.ID
	}
	return nil
}

func (s *Store) GetContractVersion(_ context.Context, id string) (model.ContractVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.versions[id]; ok {
		return v, nil
	}
	return model.ContractVersion{}, model.ErrNotFound
}

func (s *Store) LatestVersion(context.Context) (model.ContractVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.order) == 0 {
		return model.ContractVersion{}, model.ErrNotFound
	}
	return s.versions[s.order[len(s.order)-1]], nil
}

func (s *Store) BaselineVersion(context.Context) (model.ContractVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.baseline == "" {
		return model.ContractVersion{}, model.ErrNotFound
	}
	return s.versions[s.baseline], nil
}

func (s *Store) SaveChangeEvents(_ context.Context, events []model.ChangeEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
	return nil
}

func (s *Store) ListChangeEvents(_ context.Context, since time.Time) ([]model.ChangeEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.ChangeEvent
	for _, e := range s.events {
		if since.IsZero() || !e.DetectedAt.Before(since) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *Store) SaveProbeResults(_ context.Context, results []model.ProbeResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.probes = append(s.probes, results...)
	return nil
}

func (s *Store) ProbeHistory(_ context.Context, consumerID string, limit int) ([]model.ProbeResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.ProbeResult
	for _, p := range s.probes {
		if p.ConsumerID == consumerID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RunAt.After(out[j].RunAt) }) // newest first
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Close is a no-op; present so the store factory can treat every backend uniformly.
func (s *Store) Close() error { return nil }
