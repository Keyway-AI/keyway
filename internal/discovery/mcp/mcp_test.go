package mcp

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

// mustDiscoverJSON writes one document to a temp dir and runs discovery over it.
func mustDiscoverJSON(t *testing.T, name, body string) []model.Consumer {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	return cs
}

// mustDiscoverYAML is mustDiscoverJSON's twin for YAML inputs (identical path).
func mustDiscoverYAML(t *testing.T, name, body string) []model.Consumer {
	t.Helper()
	return mustDiscoverJSON(t, name, body)
}

// TestProtectedResourceMetadata is the RFC 9728 shape the MCP auth spec mandates:
// `resource` is the audience the token must be bound to and `authorization_servers`
// are the trusted issuers. Both must land on the consumer.
func TestProtectedResourceMetadata(t *testing.T) {
	dir := t.TempDir()
	// Served verbatim at /.well-known/oauth-protected-resource; saved as .json.
	doc := `{
  "resource": "https://mcp.example.com",
  "authorization_servers": ["https://auth.example.com"],
  "scopes_supported": ["mcp:tools", "mcp:resources"],
  "bearer_methods_supported": ["header"],
  "jwks_uri": "https://auth.example.com/jwks.json"
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "oauth-protected-resource.json"), []byte(doc), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	c := cs[0]
	assert.Equal(t, []string{"https://auth.example.com"}, c.Expects.Issuers,
		"authorization_servers must become the trusted issuers")
	assert.Contains(t, c.Expects.Audiences, "https://mcp.example.com",
		"the resource identifier is the audience the token must be bound to (RFC 8707)")
	assert.Equal(t, "https://auth.example.com/jwks.json", c.JWKSBehavior.JWKSURI)
	assert.True(t, c.Probeable, "an http(s) resource is an addressable endpoint")
	assert.Equal(t, "url://mcp.example.com", c.StableID)
}

// TestNestedServerConfigAuthBlock accepts the other common shape: an MCP server
// config whose authorization block names a single issuer plus a resource.
func TestNestedServerConfigAuthBlock(t *testing.T) {
	cfg := `mcp:
  name: files-server
  authorization:
    issuer: https://login.example.com
    resource: https://files.mcp.example.com
    scopes: [files:read]
`
	cs := mustDiscoverYAML(t, "server.yaml", cfg)
	require.Len(t, cs, 1)
	assert.Equal(t, []string{"https://login.example.com"}, cs[0].Expects.Issuers)
	assert.Contains(t, cs[0].Expects.Audiences, "https://files.mcp.example.com")
}

// TestExplicitAudienceDistinctFromResource keeps an explicit audience that differs
// from the resource URL (some deployments bind to a logical API name).
func TestExplicitAudienceDistinctFromResource(t *testing.T) {
	doc := `{
  "resource": "https://mcp.example.com",
  "authorization_servers": ["https://auth.example.com"],
  "audience": "mcp-api"
}`
	cs := mustDiscoverJSON(t, "prm.json", doc)
	require.Len(t, cs, 1)
	assert.ElementsMatch(t, []string{"https://mcp.example.com", "mcp-api"}, cs[0].Expects.Audiences)
}

// TestDedupByResource collapses the same protected-resource metadata seen in two
// files (a served .well-known doc and the server config that embeds it).
func TestDedupByResource(t *testing.T) {
	dir := t.TempDir()
	prm := `{"resource":"https://mcp.example.com","authorization_servers":["https://auth.example.com"]}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte(prm), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.json"), []byte(prm), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1, "the same resource in two documents is one consumer")
}

// TestNonURLResourceNotProbeable marks a bare resource name (not an http endpoint)
// as discovered-but-not-probeable, mirroring how SPA clients are handled.
func TestNonURLResourceNotProbeable(t *testing.T) {
	doc := `{"resource":"urn:mcp:internal-tools","authorization_servers":["https://auth.example.com"]}`
	cs := mustDiscoverJSON(t, "prm.json", doc)
	require.Len(t, cs, 1)
	assert.False(t, cs[0].Probeable, "a non-http resource identifier is not a probeable endpoint")
}

// TestNoFalsePositives guards against matching generic config maps: a `resource`
// with no authorization server, and an issuer with no resource/audience, must
// both yield nothing.
func TestNoFalsePositives(t *testing.T) {
	cs := mustDiscoverJSON(t, "a.json", `{"resource":"https://x.example.com","scopes":["a"]}`)
	assert.Empty(t, cs, "a resource with no authorization_servers is not a contract")

	cs = mustDiscoverJSON(t, "b.json", `{"issuer":"https://auth.example.com"}`)
	assert.Empty(t, cs, "an issuer with no resource or audience is not a contract")
}
