package apisix

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/model"
)

func run(t *testing.T, name, body string) []model.Consumer {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	return cs
}

// TestOpenIDConnectRoute maps the openid-connect plugin: issuer from the
// discovery URL, audience from client_id.
func TestOpenIDConnectRoute(t *testing.T) {
	cfg := `routes:
- name: orders
  uri: /orders/*
  plugins:
    openid-connect:
      discovery: https://auth.example.com/.well-known/openid-configuration
      client_id: orders-api
      jwks_uri: https://auth.example.com/jwks
`
	cs := run(t, "apisix.yaml", cfg)
	require.Len(t, cs, 1)
	c := cs[0]
	assert.Equal(t, []string{"https://auth.example.com"}, c.Expects.Issuers,
		"issuer comes from the discovery URL with the well-known suffix stripped")
	assert.Equal(t, []string{"orders-api"}, c.Expects.Audiences, "client_id is the conventional audience")
	assert.Equal(t, "https://auth.example.com/jwks", c.JWKSBehavior.JWKSURI)
	assert.InDelta(t, 0.95, c.Confidence["overall"], 0.001)
	assert.Equal(t, "route://apisix/orders", c.StableID)
}

// TestJWTAuthPlugin captures a jwt-auth route as a lower-confidence validator.
func TestJWTAuthPlugin(t *testing.T) {
	cfg := `routes:
- name: legacy
  uri: /legacy
  plugins:
    jwt-auth: {}
`
	cs := run(t, "apisix.yaml", cfg)
	require.Len(t, cs, 1)
	assert.Empty(t, cs[0].Expects.Issuers)
	assert.InDelta(t, 0.6, cs[0].Confidence["overall"], 0.001)
}

// TestExplicitAudienceUnioned keeps both client_id and an explicit audience.
func TestExplicitAudienceUnioned(t *testing.T) {
	cfg := `routes:
- name: r
  uri: /r
  plugins:
    openid-connect:
      discovery: https://auth.example.com
      client_id: r-client
      audience: [r-api]
`
	cs := run(t, "apisix.yaml", cfg)
	require.Len(t, cs, 1)
	assert.ElementsMatch(t, []string{"r-client", "r-api"}, cs[0].Expects.Audiences)
}

// TestNoJWTPluginIgnored ensures a route with only non-auth plugins is skipped.
func TestNoJWTPluginIgnored(t *testing.T) {
	cfg := `routes:
- name: r
  uri: /r
  plugins:
    limit-count:
      count: 100
`
	cs := run(t, "apisix.yaml", cfg)
	assert.Empty(t, cs)
}

// TestRouteNameFallsBackToURI derives a name from the URI when none is set.
func TestRouteNameFallsBackToURI(t *testing.T) {
	cfg := `routes:
- uri: /api/v1/*
  plugins:
    jwt-auth: {}
`
	cs := run(t, "apisix.yaml", cfg)
	require.Len(t, cs, 1)
	assert.Equal(t, "route://apisix/api_v1_*", cs[0].StableID)
}
