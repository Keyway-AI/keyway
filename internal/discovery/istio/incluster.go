package istio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/model"
)

// InClusterDiscoverer reads live Istio security CRDs from the Kubernetes API via
// a dynamic client (KI-02), instead of static manifests. It reuses the exact
// same mapping as the manifest source, so a consumer derived in-cluster is
// identical to one derived from the equivalent YAML.
type InClusterDiscoverer struct {
	client   dynamic.Interface
	versions []string // API versions to try, in order
	inner    *Discoverer
	// clusterRef labels provenance locators (e.g. the kube-context name).
	clusterRef string
}

// NewInCluster constructs an in-cluster Istio discoverer over the given dynamic
// client. clusterRef (typically the kube-context) is only used to label
// provenance; pass "" for an in-cluster (service-account) client.
func NewInCluster(client dynamic.Interface, clusterRef string) *InClusterDiscoverer {
	return &InClusterDiscoverer{
		client:     client,
		versions:   []string{"v1", "v1beta1"},
		inner:      New(),
		clusterRef: clusterRef,
	}
}

// WithClock injects a clock (tests).
func (d *InClusterDiscoverer) WithClock(now func() time.Time) *InClusterDiscoverer {
	d.inner.WithClock(now)
	return d
}

var _ discovery.Discoverer = (*InClusterDiscoverer)(nil)

// Name identifies this source in provenance records.
func (d *InClusterDiscoverer) Name() string { return "istio" }

var (
	gvrRequestAuth = "requestauthentications"
	gvrAuthPolicy  = "authorizationpolicies"
)

// Discover lists RequestAuthentication and AuthorizationPolicy objects from the
// cluster and maps them to consumers.
func (d *InClusterDiscoverer) Discover(ctx context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	raItems, err := d.list(ctx, gvrRequestAuth)
	if err != nil {
		return nil, fmt.Errorf("istio(in-cluster): list requestauthentications: %w", err)
	}
	apItems, err := d.list(ctx, gvrAuthPolicy)
	if err != nil {
		return nil, fmt.Errorf("istio(in-cluster): list authorizationpolicies: %w", err)
	}

	var ras []raWithLoc
	for _, u := range raItems {
		var ra requestAuthentication
		if err := decodeUnstructured(u, &ra); err != nil || len(ra.Spec.JWTRules) == 0 {
			continue
		}
		ras = append(ras, raWithLoc{ra, d.locator(ra.Metadata.Namespace, ra.Metadata.Name, "RequestAuthentication")})
	}
	var aps []apWithLoc
	for _, u := range apItems {
		var ap authorizationPolicy
		if err := decodeUnstructured(u, &ap); err != nil {
			continue
		}
		aps = append(aps, apWithLoc{ap, d.locator(ap.Metadata.Namespace, ap.Metadata.Name, "AuthorizationPolicy")})
	}
	return d.inner.assemble(scope, ras, aps), nil
}

// list returns the items of the first API version that responds without error,
// so a cluster serving these CRDs under v1beta1 still works.
func (d *InClusterDiscoverer) list(ctx context.Context, resource string) ([]unstructured.Unstructured, error) {
	var lastErr error
	for _, v := range d.versions {
		gvr := schema.GroupVersionResource{Group: "security.istio.io", Version: v, Resource: resource}
		ul, err := d.client.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err == nil {
			return ul.Items, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// locator builds a stable provenance reference for an in-cluster object.
func (d *InClusterDiscoverer) locator(ns, name, kind string) string {
	cluster := d.clusterRef
	if cluster == "" {
		cluster = "in-cluster"
	}
	return fmt.Sprintf("cluster://%s/%s/%s/%s", cluster, ns, kind, name)
}

// decodeUnstructured converts a dynamic-client object into one of the CRD structs
// by round-tripping through JSON. The CRD structs carry camelCase yaml tags that
// match the Kubernetes JSON field names, and JSON is valid YAML, so yaml.Unmarshal
// reads the JSON directly.
func decodeUnstructured(u unstructured.Unstructured, out any) error {
	b, err := json.Marshal(u.Object)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, out)
}
