package istio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/nometria/keyway/internal/discovery"
)

func gvr(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "security.istio.io", Version: "v1", Resource: resource}
}

func raObj(ns, name, issuer string, audiences ...string) *unstructured.Unstructured {
	auds := make([]any, len(audiences))
	for i, a := range audiences {
		auds[i] = a
	}
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "security.istio.io/v1",
		"kind":       "RequestAuthentication",
		"metadata":   map[string]any{"name": name, "namespace": ns, "labels": map[string]any{"app": name}},
		"spec": map[string]any{
			"selector": map[string]any{"matchLabels": map[string]any{"app": name}},
			"jwtRules": []any{map[string]any{"issuer": issuer, "audiences": auds}},
		},
	}}
	return u
}

func apObj(ns, name, app, claimKey string, values ...string) *unstructured.Unstructured {
	vals := make([]any, len(values))
	for i, v := range values {
		vals[i] = v
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "security.istio.io/v1",
		"kind":       "AuthorizationPolicy",
		"metadata":   map[string]any{"name": name, "namespace": ns},
		"spec": map[string]any{
			"selector": map[string]any{"matchLabels": map[string]any{"app": app}},
			"rules": []any{map[string]any{
				"when": []any{map[string]any{"key": claimKey, "values": vals}},
			}},
		},
	}}
}

func newFakeClient(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	listKinds := map[schema.GroupVersionResource]string{
		gvr("requestauthentications"): "RequestAuthenticationList",
		gvr("authorizationpolicies"):  "AuthorizationPolicyList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, objs...)
}

// TestInClusterDiscover verifies that live CRDs read via the dynamic client map
// to the same consumer shape as the manifest source, including merged required
// claims from an AuthorizationPolicy (KI-02).
func TestInClusterDiscover(t *testing.T) {
	client := newFakeClient(
		raObj("prod", "api-a", "https://kc/realms/main", "api-a", "api-b"),
		apObj("prod", "api-a-require-dept", "api-a", "request.auth.claims[dept]", "finance"),
	)
	d := NewInCluster(client, "staging-cluster")

	consumers, err := d.Discover(context.Background(), discovery.Scope{KubeContext: "staging-cluster"})
	require.NoError(t, err)
	require.Len(t, consumers, 1)

	c := consumers[0]
	assert.Equal(t, "api-a", c.Name)
	assert.Equal(t, "prod", c.Namespace)
	assert.ElementsMatch(t, []string{"api-a", "api-b"}, c.Expects.Audiences)
	assert.Equal(t, []string{"https://kc/realms/main"}, c.Expects.Issuers)
	assert.Contains(t, c.Expects.RequiredClaims, "dept", "AuthorizationPolicy claim must merge onto the workload")
	assert.Contains(t, c.Provenance["expects.issuers"][0].Locator, "cluster://staging-cluster/prod/")
}

// TestInClusterNamespaceScope verifies namespace filtering is honored.
func TestInClusterNamespaceScope(t *testing.T) {
	client := newFakeClient(
		raObj("prod", "api-a", "https://kc/realms/main", "api-a"),
		raObj("dev", "api-b", "https://kc/realms/main", "api-b"),
	)
	d := NewInCluster(client, "")

	consumers, err := d.Discover(context.Background(), discovery.Scope{Namespaces: []string{"prod"}})
	require.NoError(t, err)
	require.Len(t, consumers, 1)
	assert.Equal(t, "api-a", consumers[0].Name)
}

// TestInClusterEmpty verifies an empty cluster yields no consumers and no error.
func TestInClusterEmpty(t *testing.T) {
	d := NewInCluster(newFakeClient(), "")
	consumers, err := d.Discover(context.Background(), discovery.Scope{})
	require.NoError(t, err)
	assert.Empty(t, consumers)
}
