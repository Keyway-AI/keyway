// Package keycloak implements the issuer.Adapter for Keycloak realms, where
// Keyway controls the private key and can run the full probe + canary suite.
package keycloak

import (
	"context"

	"github.com/architsharma/keyway/internal/issuer"
	"github.com/architsharma/keyway/internal/model"
)

// Adapter talks to a Keycloak realm's admin and OIDC endpoints.
type Adapter struct {
	realmURL     string
	adminCredEnv string
}

// Options configures a Keycloak adapter.
type Options struct {
	RealmURL     string
	AdminCredEnv string
}

// New constructs a Keycloak adapter.
func New(opts Options) *Adapter {
	return &Adapter{realmURL: opts.RealmURL, adminCredEnv: opts.AdminCredEnv}
}

// Compile-time assertion that Adapter satisfies issuer.Adapter.
var _ issuer.Adapter = (*Adapter)(nil)

// Describe returns the realm's issuer metadata and JWKS. TODO(M2): fetch the
// discovery doc and JWKS from a.realmURL, authenticating with a.adminCredEnv.
func (a *Adapter) Describe(ctx context.Context) (model.Issuer, error) {
	return model.Issuer{
		Type:               model.IssuerKeycloak,
		IssuerURL:          a.realmURL,
		ControlsPrivateKey: true,
	}, model.ErrUnsupported
}

// MintToken signs claims with the named realm key. TODO(M2): use a.adminCredEnv
// to obtain the realm signing key and produce a compact JWS.
func (a *Adapter) MintToken(ctx context.Context, kid string, claims map[string]any) (string, error) {
	_ = a.adminCredEnv
	return "", model.ErrUnsupported
}

// AnnounceKey publishes a canary key without using it to sign. TODO(M6).
func (a *Adapter) AnnounceKey(ctx context.Context, alg string) (model.Key, error) {
	return model.Key{}, model.ErrUnsupported
}

// PromoteKey moves an announced key to active. TODO(M6).
func (a *Adapter) PromoteKey(ctx context.Context, kid string) error {
	return model.ErrUnsupported
}

// RetireKey removes a key from active signing. TODO(M6).
func (a *Adapter) RetireKey(ctx context.Context, kid string) error {
	return model.ErrUnsupported
}
