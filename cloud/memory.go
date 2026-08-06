package cloud

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore is an in-process Store for local/dev and tests. Data is lost on
// restart — hosting should swap in the Postgres store (same interface).
type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]User
	projects map[string]Project
	analyses map[string]Analysis
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:    map[string]User{},
		projects: map[string]Project{},
		analyses: map[string]Analysis{},
	}
}

func (s *MemoryStore) UpsertUser(_ context.Context, u User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Preserve CreatedAt across repeated logins.
	if existing, ok := s.users[u.ID]; ok {
		u.CreatedAt = existing.CreatedAt
	}
	s.users[u.ID] = u
	return nil
}

func (s *MemoryStore) GetUser(_ context.Context, id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) CreateProject(_ context.Context, p Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ID] = p
	return nil
}

func (s *MemoryStore) ListProjects(_ context.Context, ownerID string) ([]Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Project
	for _, p := range s.projects {
		if p.OwnerID == ownerID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) GetProject(_ context.Context, id string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	return p, nil
}

func (s *MemoryStore) DeleteProject(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[id]; !ok {
		return ErrNotFound
	}
	delete(s.projects, id)
	for aid, a := range s.analyses {
		if a.ProjectID == id {
			delete(s.analyses, aid)
		}
	}
	return nil
}

func (s *MemoryStore) SaveAnalysis(_ context.Context, a Analysis) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.analyses[a.ID] = a
	if p, ok := s.projects[a.ProjectID]; ok {
		p.LatestAnalysisID = a.ID
		s.projects[a.ProjectID] = p
	}
	return nil
}

func (s *MemoryStore) ListAnalyses(_ context.Context, projectID string, limit int) ([]Analysis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Analysis
	for _, a := range s.analyses {
		if a.ProjectID == projectID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) GetAnalysis(_ context.Context, id string) (Analysis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.analyses[id]
	if !ok {
		return Analysis{}, ErrNotFound
	}
	return a, nil
}

func (s *MemoryStore) LatestAnalysis(ctx context.Context, projectID string) (Analysis, error) {
	list, err := s.ListAnalyses(ctx, projectID, 1)
	if err != nil {
		return Analysis{}, err
	}
	if len(list) == 0 {
		return Analysis{}, ErrNotFound
	}
	return list[0], nil
}

func (s *MemoryStore) Close() error { return nil }
