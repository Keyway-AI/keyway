package api

import (
	"sync"

	"github.com/Keyway-AI/keyway/internal/model"
)

// runIndex keeps the results of recent probe runs by run_id so they can be
// fetched via GET /v1/probes/runs/{run_id}. Bounded FIFO; the durable per-
// consumer history still lives in Postgres (KI-04).
type runIndex struct {
	mu    sync.Mutex
	max   int
	order []string
	byID  map[string][]model.ProbeResult
}

func newRunIndex(max int) *runIndex {
	return &runIndex{max: max, byID: make(map[string][]model.ProbeResult)}
}

func (r *runIndex) put(id string, results []model.ProbeResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[id]; !ok {
		r.order = append(r.order, id)
	}
	r.byID[id] = results
	for len(r.order) > r.max {
		oldest := r.order[0]
		r.order = r.order[1:]
		delete(r.byID, oldest)
	}
}

func (r *runIndex) get(id string) ([]model.ProbeResult, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.byID[id]
	return res, ok
}
