package oidcclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/model"
)

// mockKeycloak serves the two admin endpoints the discoverer needs.
func mockKeycloak(t *testing.T) *httptest.Server {
	t.Helper()
	clientsJSON := `[
	  {"clientId":"payments-api","protocol":"openid-connect","bearerOnly":true,"enabled":true,
	   "protocolMappers":[{"protocolMapper":"oidc-audience-mapper","config":{"included.client.audience":"payments-api"}}]},
	  {"clientId":"web-spa","protocol":"openid-connect","publicClient":true,"enabled":true},
	  {"clientId":"reporting","protocol":"openid-connect","enabled":true,
	   "protocolMappers":[{"protocolMapper":"oidc-audience-mapper","config":{"included.custom.audience":"reports"}}]},
	  {"clientId":"account","protocol":"openid-connect","enabled":true},
	  {"clientId":"legacy-saml","protocol":"saml","enabled":true},
	  {"clientId":"disabled-svc","protocol":"openid-connect","bearerOnly":true,"enabled":false}
	]`
	mux := http.NewServeMux()
	mux.HandleFunc("/realms/master/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("grant_type") != "password" || r.FormValue("username") != "admin" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"admin-token"}`))
	})
	mux.HandleFunc("/admin/realms/bench/clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer admin-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(clientsJSON))
	})
	return httptest.NewServer(mux)
}

func TestOIDCClientDiscovery(t *testing.T) {
	srv := mockKeycloak(t)
	defer srv.Close()

	d, err := New(Options{
		RealmURL:      srv.URL + "/realms/bench",
		AdminUser:     "admin",
		AdminPassword: "admin",
	})
	require.NoError(t, err)

	consumers, err := d.Discover(context.Background(), discovery.Scope{})
	require.NoError(t, err)

	byID := map[string]model.Consumer{}
	for _, c := range consumers {
		byID[c.StableID] = c
	}

	// Keycloak plumbing (account), SAML, and disabled clients are excluded.
	assert.Len(t, consumers, 3)
	assert.NotContains(t, byID, "oidc://bench/account")
	assert.NotContains(t, byID, "oidc://bench/legacy-saml")
	assert.NotContains(t, byID, "oidc://bench/disabled-svc")

	// Confidential bearer client → probeable service with its audience.
	pay := byID["oidc://bench/payments-api"]
	require.NotEmpty(t, pay.StableID)
	assert.Equal(t, model.ConsumerService, pay.Kind)
	assert.True(t, pay.Probeable)
	assert.Contains(t, pay.Expects.Audiences, "payments-api")
	assert.Equal(t, []string{srv.URL + "/realms/bench"}, pay.Expects.Issuers)
	assert.InDelta(t, 0.9, pay.Confidence["overall"], 0.001)

	// Custom-audience mapper is read.
	rep := byID["oidc://bench/reporting"]
	assert.Contains(t, rep.Expects.Audiences, "reports")

	// Public SPA client → not probeable, kind=client (OPEN-1).
	spa := byID["oidc://bench/web-spa"]
	assert.Equal(t, model.ConsumerClient, spa.Kind)
	assert.False(t, spa.Probeable)
}

func TestSplitRealmURL(t *testing.T) {
	base, realm, err := splitRealmURL("https://kc.example.com/realms/main")
	require.NoError(t, err)
	assert.Equal(t, "https://kc.example.com", base)
	assert.Equal(t, "main", realm)

	_, _, err = splitRealmURL("https://kc.example.com/not-a-realm")
	assert.Error(t, err)
}

func TestAuthFailure(t *testing.T) {
	srv := mockKeycloak(t)
	defer srv.Close()
	d, _ := New(Options{RealmURL: srv.URL + "/realms/bench", AdminUser: "wrong", AdminPassword: "x"})
	_, err := d.Discover(context.Background(), discovery.Scope{})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "admin auth"))
}
