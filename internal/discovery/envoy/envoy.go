// Package envoy discovers consumers from Envoy jwt_authn filter configuration.
// cache_duration is a direct, high-confidence read of JWKS behaviour.
package envoy

import (
	"context"

	"github.com/architsharma/keyway/internal/discovery"
	"github.com/architsharma/keyway/internal/model"
)

// Discoverer parses Envoy static config or admin /config_dump.
type Discoverer struct{}

// New constructs an Envoy discoverer.
func New() *Discoverer { return &Discoverer{} }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "envoy" }

// Discover extracts providers[].issuer/audiences/remote_jwks/cache_duration.
// TODO(M2): confidence 1.0 for config files, 0.9 for admin dump.
func (d *Discoverer) Discover(ctx context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	_ = ctx
	_ = scope
	return nil, nil
}
