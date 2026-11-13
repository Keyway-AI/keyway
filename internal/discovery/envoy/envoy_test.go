package envoy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/discovery"
)

// TestClockSkewParsed guards the P5 fix: the jwt_authn provider's
// clock_skew_seconds must be captured so an explicitly wide skew can be measured.
// Before this, ClockSkewSec was never populated and P5 was vacuously 0%.
func TestClockSkewParsed(t *testing.T) {
	dir := t.TempDir()
	cfg := `http_filters:
- name: envoy.filters.http.jwt_authn
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
    providers:
      main:
        issuer: https://issuer.example.com
        audiences: [api]
        clock_skew_seconds: 600
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "envoy.yaml"), []byte(cfg), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 600, cs[0].Expects.ClockSkewSec,
		"explicit clock_skew_seconds must be captured so P5 can be measured")
}

// TestClockSkewUnsetIsZero ensures a provider that omits clock_skew_seconds
// records 0, not Envoy's 60s runtime default: the measurement flags only an
// explicit wide skew, never a provider relying on the default.
func TestClockSkewUnsetIsZero(t *testing.T) {
	dir := t.TempDir()
	cfg := `http_filters:
- name: envoy.filters.http.jwt_authn
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
    providers:
      main:
        issuer: https://issuer.example.com
        audiences: [api]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "envoy.yaml"), []byte(cfg), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, 0, cs[0].Expects.ClockSkewSec,
		"an unset clock skew is recorded as 0 (not the 60s default), so it never flags P5")
}

// TestRequiresAnyGroupsIssuers guards the P4 fix: two providers OR'd together by a
// requires_any form ONE route that trusts both issuers, so they must collapse into
// a single consumer carrying both issuers (not two single-issuer consumers, which
// would both miss P4-multi-issuer-trust and double-count the route).
func TestRequiresAnyGroupsIssuers(t *testing.T) {
	dir := t.TempDir()
	cfg := `http_filters:
- name: envoy.filters.http.jwt_authn
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
    providers:
      oidc:
        issuer: https://auth.example.health/
        audiences: [ehr]
      smart:
        issuer: https://ehr.example.health/
        audiences: [ehr]
    rules:
    - match: { prefix: / }
      requires:
        requires_any:
          requirements:
          - provider_name: oidc
          - provider_name: smart
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "envoy.yaml"), []byte(cfg), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1, "a requires_any over two providers is one route, so one consumer")
	assert.ElementsMatch(t, []string{"https://auth.example.health/", "https://ehr.example.health/"},
		cs[0].Expects.Issuers, "both OR'd issuers must appear so the route flags multi-issuer trust")
}

// TestDistinctProvidersStaySeparate ensures providers that are NOT OR'd together
// keep their own consumers (no spurious merging).
func TestDistinctProvidersStaySeparate(t *testing.T) {
	dir := t.TempDir()
	cfg := `http_filters:
- name: envoy.filters.http.jwt_authn
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
    providers:
      a:
        issuer: https://a.example.com/
      b:
        issuer: https://b.example.com/
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "envoy.yaml"), []byte(cfg), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 2, "providers with no requires_any disjunction remain independent consumers")
}
