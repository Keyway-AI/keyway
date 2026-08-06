package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Keyway-AI/keyway/internal/coordination"
)

// The idempotency middleware replays the cached response when a POST repeats an
// Idempotency-Key for the SAME method, path, and body (PRD §12). Storage is the
// coordination.IdempotencyStore seam — in-memory for a single daemon, Postgres to
// share replays across replicas. The per-key in-flight lock below is a same-node
// concern (coalescing concurrent duplicates, KI-29) kept separate from storage.

// keyMutex serializes concurrent requests that share an idempotency key on this
// node, with a refcount so each lock is removed once no request references it.
type keyMutex struct {
	mu    sync.Mutex
	locks map[string]*keyLock
}

type keyLock struct {
	mu   sync.Mutex
	refs int
}

func newKeyMutex() *keyMutex { return &keyMutex{locks: map[string]*keyLock{}} }

// lock acquires the per-key lock, so only one request per idempotency key runs at
// a time on this node. Concurrent duplicates block here and (on acquiring) find
// the cached result to replay instead of re-executing.
func (k *keyMutex) lock(key string) {
	k.mu.Lock()
	kl := k.locks[key]
	if kl == nil {
		kl = &keyLock{}
		k.locks[key] = kl
	}
	kl.refs++
	k.mu.Unlock()
	kl.mu.Lock()
}

func (k *keyMutex) unlock(key string) {
	k.mu.Lock()
	kl := k.locks[key] // this goroutine holds a ref, so it's still present
	k.mu.Unlock()
	kl.mu.Unlock()
	k.mu.Lock()
	kl.refs--
	if kl.refs == 0 {
		delete(k.locks, key)
	}
	k.mu.Unlock()
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

		ctx := r.Context()
		key := idemKey(rawKey, r.Method, r.URL.Path, body)
		replay := func(rec coordination.Record) {
			if rec.ContentType != "" {
				w.Header().Set("Content-Type", rec.ContentType)
			}
			w.Header().Set("Idempotent-Replay", "true")
			w.WriteHeader(rec.Status)
			_, _ = w.Write(rec.Body)
		}

		if rec, ok, _ := s.idem.Get(ctx, key); ok {
			replay(rec)
			return
		}

		// Serialize concurrent requests that share the key on this node so only
		// the first executes; the rest replay its cached result (KI-29).
		s.inflight.lock(key)
		defer s.inflight.unlock(key)
		if rec, ok, _ := s.idem.Get(ctx, key); ok {
			replay(rec)
			return
		}

		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.status < 500 {
			_ = s.idem.Put(ctx, key, coordination.Record{
				Status:      rec.status,
				Body:        rec.body.Bytes(),
				ContentType: rec.Header().Get("Content-Type"),
			})
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

// defaultIdempotencyTTL is the in-memory cache lifetime used when the server is
// constructed without an explicit store.
const defaultIdempotencyTTL = 24 * time.Hour
