package attribution

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestK8sDeployAttribute(t *testing.T) {
	dir := t.TempDir()
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-a
  namespace: prod
  annotations:
    kubernetes.io/change-cause: "kubectl apply -f api-a.yaml (PR #482 by alice)"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api-a.yaml"), []byte(manifest), 0o644))

	a := NewK8sDeploy(dir)
	attr, err := a.Attribute(context.Background(), model.ChangeEvent{Evidence: []string{"api-a.yaml"}})
	require.NoError(t, err)
	assert.Equal(t, "deploy", attr.Kind)
	assert.Contains(t, attr.Ref, "PR #482")
	assert.InDelta(t, 0.8, attr.Confidence, 0.001)
}

func TestK8sDeployNoAnnotation(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plain.yaml"), []byte("kind: Deployment\nmetadata:\n  name: x\n"), 0o644))
	a := NewK8sDeploy(dir)
	attr, _ := a.Attribute(context.Background(), model.ChangeEvent{Evidence: []string{"plain.yaml"}})
	assert.Equal(t, "unattributed", attr.Kind)
}

// TestChainPrefersFirstMatch verifies the chain returns the first source that binds.
func TestChainPrefersFirstMatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "d.yaml"),
		[]byte("kind: Deployment\nmetadata:\n  annotations:\n    kubernetes.io/change-cause: \"deploy-42\"\n"), 0o644))

	// git attributor finds nothing (not a repo); deploy attributor binds.
	chain := NewChain(NewGit(dir), NewK8sDeploy(dir))
	attr, err := chain.Attribute(context.Background(), model.ChangeEvent{Evidence: []string{"d.yaml"}})
	require.NoError(t, err)
	assert.Equal(t, "deploy", attr.Kind)
	assert.Equal(t, "deploy-42", attr.Ref)
}
