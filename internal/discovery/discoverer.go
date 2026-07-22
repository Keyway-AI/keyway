// Package discovery derives the consumer inventory automatically from cluster
// and config sources. No source ever requires the user to author a model file.
package discovery

import (
	"context"

	"github.com/architsharma/keyway/internal/model"
)

// Scope bounds a discovery run.
type Scope struct {
	KubeContext string
	Namespaces  []string
	ConfigPaths []string
	IssuerURLs  []string
}

// Discoverer is one source of consumers (Istio, Envoy, K8s, OIDC client registry).
type Discoverer interface {
	// Name is a stable identifier used in provenance records, e.g. "istio".
	Name() string
	// Discover returns the consumers this source can see within scope.
	Discover(ctx context.Context, scope Scope) ([]model.Consumer, error)
}
