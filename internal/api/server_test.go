package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/app"
	"github.com/Keyway-AI/keyway/internal/issuerregistry"
	"github.com/Keyway-AI/keyway/internal/model"
)

// memStore is a minimal in-memory store.Store for handler tests.
type memStore struct {
	versions map[string]model.ContractVersion
	latest   *model.ContractVersion
}

func newMemStore() *memStore { return &memStore{versions: map[string]model.ContractVersion{}} }

func (m *memStore) SaveContractVersion(_ context.Context, v model.ContractVersion) error {
	m.versions[v.ID] = v
	vv := v
	m.latest = &vv
	return nil
}
func (m *memStore) GetContractVersion(_ context.Context, id string) (model.ContractVersion, error) {
	if v, ok := m.versions[id]; ok {
		return v, nil
	}
	return model.ContractVersion{}, model.ErrNotFound
}
func (m *memStore) LatestVersion(context.Context) (model.ContractVersion, error) {
	if m.latest == nil {
		return model.ContractVersion{}, model.ErrNotFound
	}
	return *m.latest, nil
}
func (m *memStore) BaselineVersion(context.Context) (model.ContractVersion, error) {
	return model.ContractVersion{}, model.ErrNotFound
}
func (m *memStore) SaveChangeEvents(context.Context, []model.ChangeEvent) error { return nil }
func (m *memStore) ListChangeEvents(context.Context, time.Time) ([]model.ChangeEvent, error) {
	return nil, nil
}
func (m *memStore) SaveProbeResults(context.Context, []model.ProbeResult) error { return nil }
func (m *memStore) ProbeHistory(context.Context, string, int) ([]model.ProbeResult, error) {
	return nil, nil
}

func testServer(t *testing.T) *Server {
	t.Helper()
	reg, err := issuerregistry.NewRegistry([]issuerregistry.Spec{
		{Name: "default", Type: model.IssuerGenericOIDC, URL: "https://issuer.test"},
	})
	require.NoError(t, err)
	return NewServer(Config{Token: "secret"}, app.Deps{Store: newMemStore(), Issuers: reg})
}

func do(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, strings.NewReader(string(b)))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestHealthNoAuth(t *testing.T) {
	h := testServer(t).Routes()
	w := do(t, h, "GET", "/v1/health", "", nil)
	assert.Equal(t, 200, w.Code)
}

func TestAuthRequired(t *testing.T) {
	h := testServer(t).Routes()
	assert.Equal(t, 401, do(t, h, "GET", "/v1/coverage", "", nil).Code)
	assert.Equal(t, 401, do(t, h, "GET", "/v1/coverage", "wrong", nil).Code)
	assert.Equal(t, 200, do(t, h, "GET", "/v1/coverage", "secret", nil).Code)
}

func TestCanaryFlowOverAPI(t *testing.T) {
	h := testServer(t).Routes()

	// Announce.
	w := do(t, h, "POST", "/v1/canary/announce", "secret", map[string]any{"issuer_id": "default", "alg": "RS256"})
	require.Equal(t, 200, w.Code)
	var key model.Key
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &key))
	assert.Equal(t, model.KeyAnnounced, key.Status)

	// Status shows the announced kid.
	w = do(t, h, "GET", "/v1/canary/status?issuer_id=default", "secret", nil)
	require.Equal(t, 200, w.Code)
	var status map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &status))
	assert.Equal(t, key.KID, status["announced_kid"])

	// Promote.
	w = do(t, h, "POST", "/v1/canary/promote", "secret", map[string]any{"issuer_id": "default", "kid": key.KID})
	assert.Equal(t, 200, w.Code)
}

func TestSnapshotAndConsumers(t *testing.T) {
	h := testServer(t).Routes()
	// No discoverers -> empty baseline snapshot.
	w := do(t, h, "POST", "/v1/snapshots", "secret", nil)
	require.Equal(t, 200, w.Code)
	var snap map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &snap))
	assert.True(t, snap["is_baseline"].(bool))

	w = do(t, h, "GET", "/v1/consumers", "secret", nil)
	assert.Equal(t, 200, w.Code)
}

func TestUIServed(t *testing.T) {
	h := testServer(t).Routes()
	w := do(t, h, "GET", "/", "", nil)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "Keyway")
	// The embedded bundle must be the real Vite build (references a hashed asset),
	// not the placeholder — guards KI-12.
	assert.Contains(t, w.Body.String(), "/assets/", "expected the real web bundle to be embedded (run `make web-build`)")

	// A client-side route falls back to index.html (SPA history mode).
	spa := do(t, h, "GET", "/consumers", "", nil)
	assert.Equal(t, 200, spa.Code)
	assert.Contains(t, spa.Body.String(), "/assets/")
}

// doKey is like do but sets an Idempotency-Key header.
func doKey(t *testing.T, h http.Handler, method, path, token, key string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, strings.NewReader(string(b)))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// TestIdempotencyReplay verifies a repeated Idempotency-Key replays the original
// response instead of re-executing — canary announce mints a new key each call,
// so the replayed body must be byte-identical (same kid).
func TestIdempotencyReplay(t *testing.T) {
	h := testServer(t).Routes()
	body := map[string]any{"issuer_id": "default", "alg": "RS256"}

	first := doKey(t, h, "POST", "/v1/canary/announce", "secret", "key-abc", body)
	require.Equal(t, 200, first.Code)
	assert.Empty(t, first.Header().Get("Idempotent-Replay"))

	replay := doKey(t, h, "POST", "/v1/canary/announce", "secret", "key-abc", body)
	require.Equal(t, 200, replay.Code)
	assert.Equal(t, "true", replay.Header().Get("Idempotent-Replay"))
	assert.Equal(t, first.Body.String(), replay.Body.String(), "replay must be byte-identical")

	// A different key executes again -> a different kid.
	other := doKey(t, h, "POST", "/v1/canary/announce", "secret", "key-xyz", body)
	require.Equal(t, 200, other.Code)
	assert.NotEqual(t, first.Body.String(), other.Body.String())
}

// TestIdempotencyNoKeyPassthrough verifies requests without the header are not cached.
func TestIdempotencyNoKeyPassthrough(t *testing.T) {
	h := testServer(t).Routes()
	body := map[string]any{"issuer_id": "default", "alg": "RS256"}
	a := do(t, h, "POST", "/v1/canary/announce", "secret", body)
	b := do(t, h, "POST", "/v1/canary/announce", "secret", body)
	assert.NotEqual(t, a.Body.String(), b.Body.String(), "no key -> each request executes")
}

// TestIdempotencyBoundToBody verifies the cache key includes the request body,
// so the same Idempotency-Key with a DIFFERENT body executes fresh rather than
// replaying an unrelated response (SEC-05). Without body binding, the second
// call would wrongly return the first body.
func TestIdempotencyBoundToBody(t *testing.T) {
	h := testServer(t).Routes()

	first := doKey(t, h, "POST", "/v1/canary/announce", "secret", "same-key",
		map[string]any{"issuer_id": "default", "alg": "RS256"})
	require.Equal(t, 200, first.Code)

	// Same key, different body: must NOT replay the first response.
	second := doKey(t, h, "POST", "/v1/canary/announce", "secret", "same-key",
		map[string]any{"issuer_id": "default", "alg": "RS384"})
	require.Equal(t, 200, second.Code)
	assert.Empty(t, second.Header().Get("Idempotent-Replay"),
		"different body under the same key must not be served from cache")
	assert.NotEqual(t, first.Body.String(), second.Body.String())
}

// TestConsumerProbesEndpoint verifies the per-consumer probe-history endpoint
// (powers the detail drawer) returns 200 with a results array.
func TestConsumerProbesEndpoint(t *testing.T) {
	h := testServer(t).Routes()
	w := do(t, h, "GET", "/v1/consumers/k8s%3A%2F%2Flocal%2Fprod%2Fapi-a/probes", "secret", nil)
	require.Equal(t, 200, w.Code)
	var body struct {
		ConsumerID string              `json:"consumer_id"`
		Results    []model.ProbeResult `json:"results"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotNil(t, body.Results, "results must be a (possibly empty) array, never null")
}

// TestIdempotencyConcurrentCoalescing fires many concurrent requests with the
// SAME Idempotency-Key and body: exactly one must execute (the canary announce
// mints a new key each real call), the rest replay it, and every response body
// must be identical (KI-29).
func TestIdempotencyConcurrentCoalescing(t *testing.T) {
	h := testServer(t).Routes()
	body := map[string]any{"issuer_id": "default", "alg": "RS256"}

	const n = 24
	var wg sync.WaitGroup
	start := make(chan struct{})
	bodies := make([]string, n)
	replays := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // release all at once to maximize overlap
			w := doKey(t, h, "POST", "/v1/canary/announce", "secret", "same-key", body)
			bodies[i] = w.Body.String()
			replays[i] = w.Header().Get("Idempotent-Replay")
		}(i)
	}
	close(start)
	wg.Wait()

	executed := 0
	for i := 0; i < n; i++ {
		assert.Equal(t, bodies[0], bodies[i], "every concurrent response must be identical")
		if replays[i] != "true" {
			executed++
		}
	}
	assert.Equal(t, 1, executed, "exactly one request executes; the rest replay")
}

func TestRunIndex(t *testing.T) {
	idx := newRunIndex(2)
	idx.put("a", []model.ProbeResult{{ProbeID: "valid_token"}})
	idx.put("b", nil)
	idx.put("c", nil) // evicts "a" (FIFO, max 2)

	_, ok := idx.get("a")
	assert.False(t, ok, "oldest evicted")
	got, ok := idx.get("b")
	assert.True(t, ok)
	_ = got
	res, ok := idx.get("a")
	assert.False(t, ok)
	assert.Nil(t, res)
}

func TestGetProbeRunNotFound(t *testing.T) {
	h := testServer(t).Routes()
	w := do(t, h, "GET", "/v1/probes/runs/does-not-exist", "secret", nil)
	assert.Equal(t, 404, w.Code)
}
