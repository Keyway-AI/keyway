package kong

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/model"

	"github.com/Keyway-AI/keyway/internal/discovery"
)

func run(t *testing.T, name, body string) []model.Consumer {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	return cs
}

// TestOpenIDConnectRoute maps the openid-connect plugin's issuer/audience to a
// high-confidence consumer, stripping the discovery-URL suffix from the issuer.
func TestOpenIDConnectRoute(t *testing.T) {
	cfg := `_format_version: "3.0"
services:
- name: orders
  url: http://orders.internal
  routes:
  - name: orders-api
    paths: [/orders]
    plugins:
    - name: openid-connect
      config:
        issuer: https://auth.example.com/.well-known/openid-configuration
        audience: [orders-api]
        jwks_uri: https://auth.example.com/jwks
`
	cs := run(t, "kong.yml", cfg)
	require.Len(t, cs, 1)
	c := cs[0]
	assert.Equal(t, []string{"https://auth.example.com"}, c.Expects.Issuers,
		"the /.well-known/openid-configuration suffix must be stripped to the issuer")
	assert.Equal(t, []string{"orders-api"}, c.Expects.Audiences)
	assert.Equal(t, "https://auth.example.com/jwks", c.JWKSBehavior.JWKSURI)
	assert.InDelta(t, 0.95, c.Confidence["overall"], 0.001)
	assert.Equal(t, "route://kong/orders-api", c.StableID)
}

// TestJWTPluginRecordsClaims captures the classic jwt plugin: no issuer/audience
// in config, but claims_to_verify tells us which claims are enforced.
func TestJWTPluginRecordsClaims(t *testing.T) {
	cfg := `_format_version: "3.0"
services:
- name: legacy
  routes:
  - name: legacy-api
    paths: [/legacy]
    plugins:
    - name: jwt
      config:
        claims_to_verify: [exp, nbf]
`
	cs := run(t, "kong.yml", cfg)
	require.Len(t, cs, 1)
	assert.ElementsMatch(t, []string{"exp", "nbf"}, cs[0].Expects.RequiredClaims)
	assert.Empty(t, cs[0].Expects.Issuers, "the classic jwt plugin keys issuer off credentials, not config")
	assert.InDelta(t, 0.6, cs[0].Confidence["overall"], 0.001)
}

// TestServiceScopePluginAppliesToRoutes checks a plugin attached at service scope
// is inherited by the service's routes.
func TestServiceScopePluginAppliesToRoutes(t *testing.T) {
	cfg := `_format_version: "3.0"
services:
- name: billing
  plugins:
  - name: openid-connect
    config:
      issuer: https://auth.example.com
      audience: billing
  routes:
  - name: billing-a
    paths: [/a]
  - name: billing-b
    paths: [/b]
`
	cs := run(t, "kong.yml", cfg)
	require.Len(t, cs, 2, "a service-scope JWT plugin applies to each of its routes")
	for _, c := range cs {
		assert.Equal(t, []string{"https://auth.example.com"}, c.Expects.Issuers)
	}
}

// TestRouteOverridesGlobal confirms the narrower (route) scope wins when the same
// plugin is set globally and on the route.
func TestRouteOverridesGlobal(t *testing.T) {
	cfg := `_format_version: "3.0"
plugins:
- name: openid-connect
  config:
    issuer: https://global.example.com
    audience: global
services:
- name: svc
  routes:
  - name: svc-route
    paths: [/x]
    plugins:
    - name: openid-connect
      config:
        issuer: https://route.example.com
        audience: route-aud
`
	cs := run(t, "kong.yml", cfg)
	require.Len(t, cs, 1)
	assert.Equal(t, []string{"https://route.example.com"}, cs[0].Expects.Issuers,
		"the route-scope plugin overrides the global one")
}

// TestNonKongDocumentIgnored ensures a document without _format_version is skipped.
func TestNonKongDocumentIgnored(t *testing.T) {
	cfg := `apiVersion: v1
kind: Service
metadata:
  name: not-kong
`
	cs := run(t, "svc.yaml", cfg)
	assert.Empty(t, cs)
}

// TestNoJWTPluginNoConsumer ensures a Kong route with only non-auth plugins is
// not reported as a token validator.
func TestNoJWTPluginNoConsumer(t *testing.T) {
	cfg := `_format_version: "3.0"
services:
- name: svc
  routes:
  - name: r
    paths: [/x]
    plugins:
    - name: rate-limiting
      config: { minute: 100 }
`
	cs := run(t, "kong.yml", cfg)
	assert.Empty(t, cs)
}
