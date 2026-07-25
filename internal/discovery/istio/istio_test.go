package istio

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
