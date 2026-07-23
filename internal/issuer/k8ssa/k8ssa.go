// Package k8ssa implements issuer.Adapter for Kubernetes service-account token
// issuers. Keyway operates a signing key that projected-token consumers are
// configured to trust in staging, enabling the full probe + canary suite.
package k8ssa

import (
	"context"
	"time"

	"github.com/architsharma/keyway/internal/issuer"
	"github.com/architsharma/keyway/internal/issuer/localkeys"
	"github.com/architsharma/keyway/internal/model"
)

// Adapter serves K8s service-account issuer operations.
type Adapter struct {
	name      string
	issuerURL string
	keys      *localkeys.KeySet
	now       func() time.Time
}

// Options configures a K8s SA adapter.
type Options struct {
	Name      string
	IssuerURL string
	Now       func() time.Time
}

// New constructs a K8s SA adapter with a Keyway-operated active key.
func New(opts Options) (*Adapter, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	a := &Adapter{name: opts.Name, issuerURL: opts.IssuerURL, keys: localkeys.NewKeySet(), now: now}
	if _, err := a.keys.Generate("RS256", model.KeyActive, now()); err != nil {
		return nil, err
	}
	return a, nil
}

var _ issuer.Adapter = (*Adapter)(nil)

// KeySet exposes the Keyway-operated key set.
func (a *Adapter) KeySet() *localkeys.KeySet { return a.keys }

// Describe returns the issuer metadata including current keys.
func (a *Adapter) Describe(context.Context) (model.Issuer, error) {
	return model.Issuer{
		Name:               a.name,
		Type:               model.IssuerK8sSA,
		IssuerURL:          a.issuerURL,
		Keys:               a.keys.Keys(),
		ControlsPrivateKey: true,
	}, nil
}

// MintToken signs claims with the Keyway-operated key.
func (a *Adapter) MintToken(_ context.Context, kid string, claims map[string]any) (string, error) {
	return a.keys.Sign(kid, claims)
}

// AnnounceKey publishes a Keyway-operated canary key.
func (a *Adapter) AnnounceKey(_ context.Context, alg string) (model.Key, error) {
	return a.keys.Generate(alg, model.KeyAnnounced, a.now())
}

// PromoteKey moves an announced key to active.
func (a *Adapter) PromoteKey(_ context.Context, kid string) error {
	return a.keys.Promote(kid, a.now())
}

// RetireKey marks a key retired.
func (a *Adapter) RetireKey(_ context.Context, kid string) error {
	return a.keys.Retire(kid, a.now())
}
