// Package oidcclient discovers consumers from an OIDC provider's client
// registry (e.g. Keycloak clients with audience/protocol mappers).
package oidcclient

import (
	"context"

	"github.com/architsharma/keyway/internal/discovery"
	"github.com/architsharma/keyway/internal/model"
)

// Discoverer reads registered clients from an OIDC provider.
type Discoverer struct{}

// New constructs an OIDC client-registry discoverer.
func New() *Discoverer { return &Discoverer{} }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "oidcclient" }

// Discover reads /admin/realms/{realm}/clients and maps token-validating
// clients to consumers. TODO(M2).
func (d *Discoverer) Discover(ctx context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	_ = ctx
	_ = scope
	return nil, nil
}
