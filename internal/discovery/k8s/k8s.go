// Package k8s discovers consumers from Kubernetes Services, their workloads, and
// projected service-account token volumes.
package k8s

import (
	"context"

	"github.com/architsharma/keyway/internal/discovery"
	"github.com/architsharma/keyway/internal/model"
)

// ownerLabelPriority is the label lookup order for OwnerTeam (PRD §7.3).
var ownerLabelPriority = []string{"team", "owner", "app.kubernetes.io/part-of"}

// Discoverer enumerates Services and backing Deployments/StatefulSets.
type Discoverer struct{}

// New constructs a Kubernetes discoverer.
func New() *Discoverer { return &Discoverer{} }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "k8s" }

// Discover detects SA token projections (audience read directly), env-var hints
// (confidence 0.5), owner labels, and endpoints. TODO(M2).
func (d *Discoverer) Discover(ctx context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	_ = ctx
	_ = scope
	_ = ownerLabelPriority
	return nil, nil
}
