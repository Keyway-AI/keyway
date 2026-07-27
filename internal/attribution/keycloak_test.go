package attribution

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nometria/keyway/internal/model"
)

// mockKeycloakAdmin serves the admin-cli token grant and an admin-events log.
func mockKeycloakAdmin(t *testing.T, events string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/master/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "admin-token"})
	})
	mux.HandleFunc("/admin/realms/main/admin-events", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer admin-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, events)
	})
	return httptest.NewServer(mux)
}

func TestKeycloakAdminAttributeIdPSide(t *testing.T) {
	ref := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	cause := ref.Add(-1 * time.Hour).UnixMilli()  // the real cause
	future := ref.Add(1 * time.Hour).UnixMilli()  // after detection -> ignored
	stale := ref.Add(-48 * time.Hour).UnixMilli() // outside the 24h window
	events := fmt.Sprintf(`[
		{"time":%d,"operationType":"UPDATE","resourceType":"COMPONENT","resourcePath":"components/rsa-key","authDetails":{"userId":"ops-alice"}},
		{"time":%d,"operationType":"CREATE","resourceType":"CLIENT","resourcePath":"clients/x","authDetails":{"userId":"later-bob"}},
		{"time":%d,"operationType":"DELETE","resourceType":"CLIENT","resourcePath":"clients/y","authDetails":{"userId":"old-carol"}}
	]`, cause, future, stale)

	srv := mockKeycloakAdmin(t, events)
	defer srv.Close()

	a, err := NewKeycloakAdmin(KeycloakOptions{
		RealmURL:      srv.URL + "/realms/main",
		AdminUser:     "admin",
		AdminPassword: "admin",
		Now:           func() time.Time { return ref },
	})
	require.NoError(t, err)

	// A JWKS-behavior change is IdP-side, so the attributor engages.
	ev := model.ChangeEvent{
		Field:      "jwks_behavior.refreshes_on_unknown_kid",
		DetectedAt: ref,
	}
	attr, err := a.Attribute(context.Background(), ev)
	require.NoError(t, err)
	assert.Equal(t, "idp_audit", attr.Kind)
	assert.Equal(t, "ops-alice", attr.Actor, "must pick the most recent event before detection, within window")
	assert.Contains(t, attr.Ref, "UPDATE")
	assert.InDelta(t, 0.7, attr.Confidence, 0.001)
}

func TestKeycloakAdminSkipsNonIdP(t *testing.T) {
	// A manifest-file change is not IdP-side; the attributor must decline (so git/
	// deploy attribution can own it) without even calling the admin API.
	a, err := NewKeycloakAdmin(KeycloakOptions{RealmURL: "https://kc.example.com/realms/main"})
	require.NoError(t, err)
	ev := model.ChangeEvent{Field: "expects.audiences", Evidence: []string{"deploy/istio.yaml"}}
	attr, err := a.Attribute(context.Background(), ev)
	require.NoError(t, err)
	assert.Equal(t, "unattributed", attr.Kind)
}

func TestKeycloakAdminRejectsHTTP(t *testing.T) {
	_, err := NewKeycloakAdmin(KeycloakOptions{RealmURL: "http://kc.example.com/realms/main"})
	require.Error(t, err, "admin credentials must not be sent over plain http")
}

func TestKeycloakAdminNoMatch(t *testing.T) {
	ref := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	// Only a future event exists -> nothing to attribute.
	events := fmt.Sprintf(`[{"time":%d,"operationType":"UPDATE","resourceType":"COMPONENT","resourcePath":"c/k","authDetails":{"userId":"x"}}]`,
		ref.Add(2*time.Hour).UnixMilli())
	srv := mockKeycloakAdmin(t, events)
	defer srv.Close()

	a, err := NewKeycloakAdmin(KeycloakOptions{RealmURL: srv.URL + "/realms/main", Now: func() time.Time { return ref }})
	require.NoError(t, err)
	attr, err := a.Attribute(context.Background(), model.ChangeEvent{Field: "jwks_behavior.cache_ttl_sec", DetectedAt: ref})
	require.NoError(t, err)
	assert.Equal(t, "unattributed", attr.Kind)
}
