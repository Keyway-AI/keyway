// Package generic implements issuer.Adapter for a generic OIDC issuer whose
// signing keys Keyway controls locally. It is fully self-contained (it holds its
// own key material), which makes it the reference issuer for tests and the local
// canary demo.
package generic

import (
	"context"
	"time"

	"github.com/Keyway-AI/keyway/internal/issuer"
	"github.com/Keyway-AI/keyway/internal/issuer/localkeys"
	"github.com/Keyway-AI/keyway/internal/model"
)

// Adapter is a locally-keyed OIDC issuer.
type Adapter struct {
	name      string
	issuerURL string
	jwksURI   string
	keys      *localkeys.KeySet
	now       func() time.Time
}

// Options configures a generic issuer.
type Options struct {
	Name      string
	IssuerURL string
	JWKSURI   string
	// Now allows tests to inject a clock; defaults to time.Now.
	Now func() time.Time
	// ImportKeys, when non-empty, seeds the key set with these persisted keys
	// (so the issuer mints with a fixed, externally-shared key) instead of
	// generating a fresh one. Used by the empirical eval to share one key between
	// the JWKS server real verifiers fetch and the harness that mints tokens.
	ImportKeys []localkeys.PersistedKey
}

// New constructs a generic issuer with a freshly generated active RS256 key.
func New(opts Options) (*Adapter, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	jwks := opts.JWKSURI
	if jwks == "" {
		jwks = opts.IssuerURL + "/.well-known/jwks.json"
	}
	a := &Adapter{
		name:      opts.Name,
		issuerURL: opts.IssuerURL,
		jwksURI:   jwks,
		keys:      localkeys.NewKeySet(),
		now:       now,
	}
	if len(opts.ImportKeys) > 0 {
		if err := a.keys.Import(opts.ImportKeys); err != nil {
			return nil, err
		}
		return a, nil
	}
	if _, err := a.keys.Generate("RS256", model.KeyActive, now()); err != nil {
		return nil, err
	}
	return a, nil
}

var _ issuer.Adapter = (*Adapter)(nil)

// KeySet exposes the underlying key set (used by the probe engine to resolve
// active/announced/retired kids and by tests to serve a JWKS).
func (a *Adapter) KeySet() *localkeys.KeySet { return a.keys }

// Describe returns the issuer metadata including current JWKS keys.
func (a *Adapter) Describe(context.Context) (model.Issuer, error) {
	return model.Issuer{
		Name:               a.name,
		Type:               model.IssuerGenericOIDC,
		IssuerURL:          a.issuerURL,
		JWKSURI:            a.jwksURI,
		Keys:               a.keys.Keys(),
		ControlsPrivateKey: true,
	}, nil
}

// MintToken signs claims with the named key (or the active key when kid == "").
func (a *Adapter) MintToken(_ context.Context, kid string, claims map[string]any) (string, error) {
	return a.keys.Sign(kid, claims)
}

// AnnounceKey publishes a new canary key without using it to sign.
func (a *Adapter) AnnounceKey(_ context.Context, alg string) (model.Key, error) {
	return a.keys.Generate(alg, model.KeyAnnounced, a.now())
}

// PromoteKey moves an announced key into active signing use.
func (a *Adapter) PromoteKey(_ context.Context, kid string) error {
	return a.keys.Promote(kid, a.now())
}

// RetireKey marks a key retired.
func (a *Adapter) RetireKey(_ context.Context, kid string) error {
	return a.keys.Retire(kid, a.now())
}
