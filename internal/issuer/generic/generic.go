// Package generic implements the issuer.Adapter for generic OIDC issuers with
// locally controlled keys.
package generic

import (
	"context"

	"github.com/architsharma/keyway/internal/issuer"
	"github.com/architsharma/keyway/internal/model"
)

// Adapter serves generic local-key OIDC issuer operations. TODO(M2/M6).
type Adapter struct{}

// New constructs a generic OIDC adapter.
func New() *Adapter { return &Adapter{} }

var _ issuer.Adapter = (*Adapter)(nil)

func (a *Adapter) Describe(context.Context) (model.Issuer, error) {
	return model.Issuer{}, model.ErrUnsupported
}
func (a *Adapter) MintToken(context.Context, string, map[string]any) (string, error) {
	return "", model.ErrUnsupported
}
func (a *Adapter) AnnounceKey(context.Context, string) (model.Key, error) {
	return model.Key{}, model.ErrUnsupported
}
func (a *Adapter) PromoteKey(context.Context, string) error { return model.ErrUnsupported }
func (a *Adapter) RetireKey(context.Context, string) error  { return model.ErrUnsupported }
