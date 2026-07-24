package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nometria/keyway/internal/issuerregistry"
	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	return NewServer(Config{Token: "secret"}, Deps{Store: newMemStore(), Issuers: reg})
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
}
