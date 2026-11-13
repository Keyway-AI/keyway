package istio

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/discovery"
)

// TestScalarAudiencesTolerated guards the robustness fix found by the 60-repo
// independent benchmark: real manifests sometimes write `audiences` as a bare
// string instead of a list. A plain []string field would fail to unmarshal and
// drop the ENTIRE RequestAuthentication (issuer included); we must tolerate both.
func TestScalarAudiencesTolerated(t *testing.T) {
	dir := t.TempDir()
	manifest := `apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata:
  name: api
  namespace: prod
spec:
  selector:
    matchLabels:
      app: api
  jwtRules:
  - issuer: "https://issuer.example.com"
    audiences: "api.example.com"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ra.yaml"), []byte(manifest), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1, "the RequestAuthentication must not be dropped by the scalar audiences")
	assert.Equal(t, []string{"https://issuer.example.com"}, cs[0].Expects.Issuers)
	assert.Equal(t, []string{"api.example.com"}, cs[0].Expects.Audiences, "scalar audience captured as a one-element list")
}

// TestNamespaceWideAndNestedClaims guards KI-30: a selector-less
// AuthorizationPolicy applies its claim to every workload in the namespace, and a
// nested claim key contributes its top-level segment.
func TestNamespaceWideAndNestedClaims(t *testing.T) {
	dir := t.TempDir()
	manifest := `apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata: { name: api, namespace: prod }
spec:
  selector: { matchLabels: { app: api } }
  jwtRules:
  - issuer: "https://kc/realms/main"
    audiences: ["api"]
---
# Namespace-wide (no selector) — applies to every workload in prod.
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata: { name: require-groups, namespace: prod }
spec:
  rules:
  - when:
    - key: request.auth.claims[realm_access][roles]
      values: ["admin"]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stack.yaml"), []byte(manifest), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Contains(t, cs[0].Expects.RequiredClaims, "realm_access",
		"a selector-less policy's nested claim must apply to the workload (top-level segment)")
}

// TestAudienceBoundViaAuthorizationPolicy guards the P1 fix: an audience can be
// bound in an AuthorizationPolicy when-condition on request.auth.audiences rather
// than in the RequestAuthentication jwtRule (the edge-RA / per-service-policy
// gateway pattern). Without attributing it, the consumer looks audience-unbound
// and is falsely counted as P1-unbound-audience.
func TestAudienceBoundViaAuthorizationPolicy(t *testing.T) {
	dir := t.TempDir()
	manifest := `apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata: { name: api, namespace: prod }
spec:
  selector: { matchLabels: { app: api } }
  jwtRules:
  - issuer: "https://issuer.example.com"
---
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata: { name: require-aud, namespace: prod }
spec:
  selector: { matchLabels: { app: api } }
  rules:
  - when:
    - key: request.auth.audiences
      values: ["api.example.com"]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stack.yaml"), []byte(manifest), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Equal(t, []string{"api.example.com"}, cs[0].Expects.Audiences,
		"audience bound via an AuthorizationPolicy when-condition must attach to the consumer")
	require.NotEmpty(t, cs[0].Provenance["expects.audiences"],
		"the AP-sourced audience must carry provenance")
}

// TestAudienceNegationNotBound ensures a notValues condition (which excludes an
// audience) is never read as binding one — that would under-report P1.
func TestAudienceNegationNotBound(t *testing.T) {
	dir := t.TempDir()
	manifest := `apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata: { name: api, namespace: prod }
spec:
  selector: { matchLabels: { app: api } }
  jwtRules:
  - issuer: "https://issuer.example.com"
---
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata: { name: deny-aud, namespace: prod }
spec:
  selector: { matchLabels: { app: api } }
  rules:
  - when:
    - key: request.auth.audiences
      notValues: ["evil.example.com"]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stack.yaml"), []byte(manifest), 0o644))

	cs, err := New().Discover(context.Background(), discovery.Scope{ConfigPaths: []string{dir}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.Empty(t, cs[0].Expects.Audiences,
		"a notValues (negation) condition must not be read as an audience binding")
}
