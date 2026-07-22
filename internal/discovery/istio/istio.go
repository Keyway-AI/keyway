// Package istio discovers consumers from Istio RequestAuthentication and
// AuthorizationPolicy resources (confidence 1.0 — declarative and unambiguous).
package istio

import (
	"context"

	"github.com/architsharma/keyway/internal/discovery"
	"github.com/architsharma/keyway/internal/model"
)

// Discoverer reads Istio security CRDs via the dynamic client.
type Discoverer struct{}

// New constructs an Istio discoverer.
func New() *Discoverer { return &Discoverer{} }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "istio" }

// Discover reads RequestAuthentication.spec.jwtRules[] and maps selectors to
// workloads. TODO(M2): implement via k8s.io/client-go dynamic client
// (security.istio.io/v1).
func (d *Discoverer) Discover(ctx context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	_ = ctx
	_ = scope
	return nil, nil
}
