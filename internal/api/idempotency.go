package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sync"
	"time"
)

// idemStore caches responses to write requests keyed by a client-supplied
// Idempotency-Key (bound to method+path+body), so a retried write replays the
// original result instead of re-executing (PRD §12). It is an in-memory,
// TTL-bounded, hard-capped cache — sufficient for a single daemon; a
// multi-replica deployment would back this with Postgres (KI-05).
type idemStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	max   int
	order []string
	m     map[string]idemEntry
	locks map[string]*keyLock // per-key locks for in-flight coalescing (KI-29)
}

// keyLock serializes concurrent requests that share an idempotency key, with a
// refcount so the lock is removed once no request references it.
type keyLock struct {
	mu   sync.Mutex
	refs int
}

type idemEntry struct {
	status      int
	body        []byte
	contentType string
	expires     time.Time
}

func newIdemStore(ttl time.Duration) *idemStore {
	return &idemStore{ttl: ttl, max: 4096, m: make(map[string]idemEntry), locks: map[string]*keyLock{}}
}

// lock acquires the per-key lock, so only one request per idempotency key runs at
// a time. Concurrent duplicates block here and (on acquiring) find the cached
// result to replay instead of re-executing (KI-29).
func (s *idemStore) lock(key string) {
	s.mu.Lock()
	kl := s.locks[key]
	if kl == nil {
		kl = &keyLock{}
		s.locks[key] = kl
	}
	kl.refs++
	s.mu.Unlock()
	kl.mu.Lock()
}

func (s *idemStore) unlock(key string) {
	s.mu.Lock()
	kl := s.locks[key] // this goroutine holds a ref, so it's still present
	s.mu.Unlock()
	kl.mu.Unlock()
	s.mu.Lock()
	kl.refs--
	if kl.refs == 0 {
		delete(s.locks, key)
	}
	s.mu.Unlock()
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
	if _, exists := s.m[key]; !exists {
		s.order = append(s.order, key)
	}
	s.m[key] = e
	// Hard FIFO cap: evict oldest entries when over capacity (not just expired
	// ones), so a burst of distinct keys cannot exhaust memory.
	for len(s.order) > s.max {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.m, oldest)
	}
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

// idempotency replays a cached response when a POST repeats an Idempotency-Key
// for the SAME method, path, and body. Requests without the header, and non-POST
// methods, pass straight through. Only deterministic responses (< 500) are
// cached, so a transient 5xx can be retried for real.
func (s *Server) idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		rawKey := r.Header.Get("Idempotency-Key")
		if rawKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Read the body to bind it into the cache key, then restore it for the
		// handler. (An upstream MaxBytesReader bounds the read.)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body too large")
			return
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))

		key := idemKey(rawKey, r.Method, r.URL.Path, body)
		replay := func(e idemEntry) {
			if e.contentType != "" {
				w.Header().Set("Content-Type", e.contentType)
			}
			w.Header().Set("Idempotent-Replay", "true")
			w.WriteHeader(e.status)
			_, _ = w.Write(e.body)
		}

		if e, ok := s.idem.get(key, time.Now()); ok {
			replay(e)
			return
		}

		// Serialize concurrent requests that share the key so only the first
		// executes; the rest replay its cached result (KI-29).
		s.idem.lock(key)
		defer s.idem.unlock(key)
		if e, ok := s.idem.get(key, time.Now()); ok {
			replay(e)
			return
		}

		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.status < 500 {
			s.idem.put(key, idemEntry{
				status:      rec.status,
				body:        rec.body.Bytes(),
				contentType: rec.Header().Get("Content-Type"),
			}, time.Now())
		}
	})
}

// idemKey binds the client key to the request identity so the same key on a
// different endpoint or body cannot replay an unrelated response.
func idemKey(rawKey, method, path string, body []byte) string {
	h := sha256.New()
	for _, part := range []string{rawKey, method, path} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
