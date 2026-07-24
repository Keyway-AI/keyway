package discovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/discovery/envoy"
	"github.com/nometria/keyway/internal/discovery/istio"
	"github.com/nometria/keyway/internal/discovery/k8s"
	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const refDir = "../../testdata/discovery/reference"

func fixedClock() func() time.Time {
	return func() time.Time { return time.Unix(1_700_000_000, 0) }
}

func TestStableID(t *testing.T) {
	assert.Equal(t, "k8s://local/prod/sa-1",
		discovery.StableID(discovery.IDParts{Namespace: "prod", ServiceAccount: "sa-1", ServiceName: "svc"}))
	assert.Equal(t, "k8s://c1/prod/svc",
		discovery.StableID(discovery.IDParts{Cluster: "c1", Namespace: "prod", ServiceName: "svc"}))
	assert.Equal(t, "route://gw/r1",
		discovery.StableID(discovery.IDParts{Gateway: "gw", Route: "r1"}))
	assert.Equal(t, "url://api.example.com/v1",
		discovery.StableID(discovery.IDParts{Host: "HTTPS://API.example.com/", PathPrefix: "/v1"}))
}

func TestIstioDiscovery(t *testing.T) {
	d := istio.New().WithClock(fixedClock())
	cs, err := d.Discover(context.Background(), discovery.Scope{ConfigPaths: []string{refDir + "/istio.yaml"}})
	require.NoError(t, err)
	require.Len(t, cs, 3)

	byName := index(cs)
	pay := byName["payments-api"]
	require.NotNil(t, pay)
	assert.Equal(t, "k8s://local/prod/payments-api", pay.StableID)
	assert.Equal(t, []string{"https://kc/realms/main"}, pay.Expects.Issuers)
	assert.Equal(t, []string{"payments-api"}, pay.Expects.Audiences)
	assert.Equal(t, "team-payments", pay.OwnerTeam)
	assert.InDelta(t, 1.0, pay.Confidence["overall"], 0.001)
	require.NotEmpty(t, pay.Provenance["expects.issuers"])
	assert.Equal(t, refDir+"/istio.yaml", pay.Provenance["expects.issuers"][0].Locator)
}

func TestEnvoyDiscovery(t *testing.T) {
	d := envoy.New().WithClock(fixedClock())
	cs, err := d.Discover(context.Background(), discovery.Scope{ConfigPaths: []string{refDir + "/envoy.yaml"}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	c := cs[0]
	assert.Equal(t, model.ConsumerGatewayRoute, c.Kind)
	assert.Equal(t, []string{"https://kc/realms/main"}, c.Expects.Issuers)
	require.NotNil(t, c.JWKSBehavior.CacheTTLSec)
	assert.Equal(t, 600, *c.JWKSBehavior.CacheTTLSec, "cache_duration read from remote_jwks")
}

func TestK8sDiscovery(t *testing.T) {
	d := k8s.New().WithClock(fixedClock())
	cs, err := d.Discover(context.Background(), discovery.Scope{ConfigPaths: []string{refDir + "/k8s.yaml"}})
	require.NoError(t, err)
	require.Len(t, cs, 3)

	byName := index(cs)
	// payments-api: env hints + matching Service -> endpoints, probeable.
	pay := byName["payments-api"]
	require.NotNil(t, pay)
	assert.True(t, pay.Probeable)
	require.NotEmpty(t, pay.Endpoints)
	assert.Equal(t, "http://payments-api.prod.svc.cluster.local:8080", pay.Endpoints[0].URL)
	assert.Contains(t, pay.Expects.Audiences, "payments-api")

	// inventory-worker: SA token projection -> SA-based StableID, high confidence.
	inv := byName["inventory-worker"]
	require.NotNil(t, inv)
	assert.Equal(t, "k8s://local/prod/inventory-sa", inv.StableID)
	assert.InDelta(t, 1.0, inv.Confidence["overall"], 0.001)
	assert.Contains(t, inv.Expects.Audiences, "inventory-worker")
}

// TestAggregateReferenceStack is the AC-3 check: discovery over the reference
// stack finds the consumers with no user-authored model file. Istio and K8s
// records for the same service merge by StableID into one enriched consumer.
func TestAggregateReferenceStack(t *testing.T) {
	scope := discovery.Scope{ConfigPaths: []string{refDir}}
	cs, err := discovery.Run(context.Background(), scope,
		istio.New().WithClock(fixedClock()),
		k8s.New().WithClock(fixedClock()),
		envoy.New().WithClock(fixedClock()),
	)
	require.NoError(t, err)

	byID := map[string]model.Consumer{}
	for _, c := range cs {
		byID[c.StableID] = c
	}

	// Expected distinct consumers across the stack.
	expected := []string{
		"k8s://local/prod/payments-api",
		"k8s://local/prod/orders-api",
		"k8s://local/data/legacy-reporting",
		"k8s://local/prod/inventory-sa",
		"route://envoy/edge-gateway",
	}
	found := 0
	for _, id := range expected {
		if _, ok := byID[id]; ok {
			found++
		}
	}
	rate := float64(found) / float64(len(expected))
	assert.GreaterOrEqual(t, rate, 0.85, "AC-3: discover >=85%% of consumers (found %d/%d)", found, len(expected))

	// payments-api must be enriched by BOTH Istio (issuer/audience) and K8s (endpoints).
	pay := byID["k8s://local/prod/payments-api"]
	require.NotNil(t, pay)
	assert.Contains(t, pay.Expects.Issuers, "https://kc/realms/main", "issuer from Istio")
	assert.NotEmpty(t, pay.Endpoints, "endpoints from K8s")
	assert.True(t, pay.Probeable)
}

func index(cs []model.Consumer) map[string]*model.Consumer {
	m := map[string]*model.Consumer{}
	for i := range cs {
		m[cs[i].Name] = &cs[i]
	}
	return m
}

func TestIstioRequiredClaimsFromAuthzPolicy(t *testing.T) {
	d := istio.New().WithClock(fixedClock())
	cs, err := d.Discover(context.Background(), discovery.Scope{ConfigPaths: []string{"../../testdata/discovery/authz/stack.yaml"}})
	require.NoError(t, err)
	require.Len(t, cs, 1)
	c := cs[0]
	assert.ElementsMatch(t, []string{"dept", "scope"}, c.Expects.RequiredClaims,
		"required claims derived from AuthorizationPolicy when-conditions")
	assert.InDelta(t, 1.0, c.Confidence["expects.required_claims"], 0.001)
	require.NotEmpty(t, c.Provenance["expects.required_claims"])
}

func TestAliasFoldMergesOIDCIntoMesh(t *testing.T) {
	mesh := model.Consumer{
		StableID: "k8s://local/prod/payments-api", Name: "payments-api", Kind: model.ConsumerService,
		Endpoints: []model.Endpoint{{URL: "http://payments-api.prod:8080"}},
		Expects:   model.Expectations{Issuers: []string{"https://kc/realms/main"}, Audiences: []string{"payments-api"}},
		Probeable: true,
	}
	oidc := model.Consumer{
		StableID: "oidc://main/payments-api", Name: "payments-api", Kind: model.ConsumerService,
		Expects: model.Expectations{Issuers: []string{"https://kc/realms/main"}, Audiences: []string{"payments-api"}},
	}
	// A genuinely distinct OIDC client (different audience) must NOT fold.
	standalone := model.Consumer{
		StableID: "oidc://main/reporting", Name: "reporting",
		Expects: model.Expectations{Issuers: []string{"https://kc/realms/main"}, Audiences: []string{"reporting"}},
	}

	out := discovery.Merge([]model.Consumer{mesh, oidc, standalone})
	ids := map[string]bool{}
	for _, c := range out {
		ids[c.StableID] = true
	}
	assert.NotContains(t, ids, "oidc://main/payments-api", "shared issuer+audience folds into the mesh consumer")
	assert.Contains(t, ids, "k8s://local/prod/payments-api")
	assert.Contains(t, ids, "oidc://main/reporting", "distinct audience is not folded")
	assert.Len(t, out, 2)
}
