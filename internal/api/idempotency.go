package api

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

// idemStore caches responses to write requests keyed by a client-supplied
// Idempotency-Key, so a retried write replays the original result instead of
// re-executing (PRD §12). It is an in-memory, TTL-bounded cache — sufficient for
// a single daemon; a multi-replica deployment would back this with Postgres
// (tracked in KNOWN_ISSUES.md KI-05).
type idemStore struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]idemEntry
}

type idemEntry struct {
	status      int
	body        []byte
	contentType string
	expires     time.Time
}

func newIdemStore(ttl time.Duration) *idemStore {
	return &idemStore{ttl: ttl, m: make(map[string]idemEntry)}
}

func (s *idemStore) get(key string, now time.Time) (idemEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[key]
	if !ok {
		return idemEntry{}, false
	}
	if now.After(e.expires) {
		delete(s.m, key)
		return idemEntry{}, false
	}
	return e, true
}

func (s *idemStore) put(key string, e idemEntry, now time.Time) {
	e.expires = now.Add(s.ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	// Opportunistic sweep so the map cannot grow without bound.
	if len(s.m) > 4096 {
		for k, v := range s.m {
			if now.After(v.expires) {
				delete(s.m, k)
			}
		}
	}
	s.m[key] = e
}

// recorder captures a handler's response so it can be cached and replayed.
type recorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// idempotency replays a cached response when a POST repeats an Idempotency-Key.
// Requests without the header, and non-POST methods, pass straight through.
// Only deterministic responses (< 500) are cached, so a transient 5xx can be
// retried for real.
func (s *Server) idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}
		now := time.Now()
		if e, ok := s.idem.get(key, now); ok {
			if e.contentType != "" {
				w.Header().Set("Content-Type", e.contentType)
			}
			w.Header().Set("Idempotent-Replay", "true")
			w.WriteHeader(e.status)
			_, _ = w.Write(e.body)
			return
		}
		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.status < 500 {
			s.idem.put(key, idemEntry{
				status:      rec.status,
				body:        rec.body.Bytes(),
				contentType: rec.Header().Get("Content-Type"),
			}, now)
		}
	})
}
