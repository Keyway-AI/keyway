package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/discovery/istio"
	"github.com/nometria/keyway/internal/discovery/k8s"
)

// TestCrossSourceMergeByAlias is the KI-28 regression test: the SAME workload is
// described by an Istio RequestAuthentication (keyed by service name) and a
// Kubernetes Deployment with a projected SA token (keyed by service account).
// Before aliases they produced two un-merged consumers; now they merge into one.
func TestCrossSourceMergeByAlias(t *testing.T) {
	dir := t.TempDir()
	// Istio side: issuer/audience, selector app=inventory-worker.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "istio.yaml"), []byte(`
apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata: { name: inventory-worker, namespace: prod }
spec:
  selector: { matchLabels: { app: inventory-worker } }
  jwtRules:
  - issuer: "https://kc/realms/main"
    audiences: ["inventory-worker"]
`), 0o644))
	// Kubernetes side: a Deployment whose SA differs from the app name, with a
	// projected service-account token (so the primary id is SA-based).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "k8s.yaml"), []byte(`
apiVersion: apps/v1
kind: Deployment
metadata: { name: inventory-worker, namespace: prod, labels: { team: platform } }
spec:
  template:
    metadata: { labels: { app: inventory-worker } }
    spec:
      serviceAccountName: inventory-sa
      volumes:
      - name: token
        projected:
          sources:
          - serviceAccountToken: { audience: "inventory-worker", path: token }
`), 0o644))

	scope := discovery.Scope{ConfigPaths: []string{dir}}
	// Istio first, so the readable service-name identity is canonical.
	cs, err := discovery.Run(context.Background(), scope, istio.New(), k8s.New())
	require.NoError(t, err)

	require.Len(t, cs, 1, "the two sources must merge into a single consumer (KI-28)")
	c := cs[0]
	assert.Equal(t, "k8s://local/prod/inventory-worker", c.StableID, "canonical id is the readable service name")
	assert.Contains(t, c.Expects.Issuers, "https://kc/realms/main", "issuer from the Istio side")
	assert.Equal(t, "platform", c.OwnerTeam, "owner from the Kubernetes side")
	assert.Contains(t, c.Aliases, "k8s://local/prod/inventory-sa", "the SA identity is retained as an alias")
}

// TestNoOverMergeDistinctWorkloads guards against the merge being too eager: two
// genuinely different workloads (different service names, no shared identity)
// must remain separate.
func TestNoOverMergeDistinctWorkloads(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "istio.yaml"), []byte(`
apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata: { name: alpha, namespace: prod }
spec:
  selector: { matchLabels: { app: alpha } }
  jwtRules:
  - issuer: "https://kc/realms/main"
    audiences: ["alpha"]
---
apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata: { name: beta, namespace: prod }
spec:
  selector: { matchLabels: { app: beta } }
  jwtRules:
  - issuer: "https://kc/realms/main"
    audiences: ["beta"]
`), 0o644))

	cs, err := discovery.Run(context.Background(), discovery.Scope{ConfigPaths: []string{dir}}, istio.New(), k8s.New())
	require.NoError(t, err)
	require.Len(t, cs, 2, "distinct workloads must not merge")
}
