// Package keycloak implements issuer.Adapter for Keycloak realms.
//
// Describe reads the realm's published OIDC metadata and JWKS over HTTP (no
// admin credentials needed for read). Minting and the canary lifecycle use a
// Keyway-operated signing key: in staging, consumers are configured to trust
// Keyway's key alongside the realm's, which is what lets Keyway present valid,
// expired, wrong-issuer, canary, etc. tokens. This mirrors the PRD's "full mode"
// where Keyway controls a private key it can sign with (§1.3), without ever
// touching the realm's own private material.
package keycloak

import (
	"context"
	"time"

	"github.com/architsharma/keyway/internal/issuer"
	"github.com/architsharma/keyway/internal/issuer/localkeys"
	"github.com/architsharma/keyway/internal/issuer/oidc"
	"github.com/architsharma/keyway/internal/model"
)

// Adapter talks to a Keycloak realm.
type Adapter struct {
	name         string
	realmURL     string
	adminCredEnv string
	keys         *localkeys.KeySet
	now          func() time.Time
}

// Options configures a Keycloak adapter.
type Options struct {
	Name         string
	RealmURL     string
	AdminCredEnv string
	Now          func() time.Time
}

// New constructs a Keycloak adapter with a Keyway-operated active signing key.
func New(opts Options) (*Adapter, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	a := &Adapter{
		name:         opts.Name,
		realmURL:     opts.RealmURL,
		adminCredEnv: opts.AdminCredEnv,
		keys:         localkeys.NewKeySet(),
		now:          now,
	}
	if _, err := a.keys.Generate("RS256", model.KeyActive, now()); err != nil {
		return nil, err
	}
	return a, nil
}

var _ issuer.Adapter = (*Adapter)(nil)

// KeySet exposes the Keyway-operated key set.
func (a *Adapter) KeySet() *localkeys.KeySet { return a.keys }

// Describe reads the realm's OIDC discovery document and JWKS.
func (a *Adapter) Describe(ctx context.Context) (model.Issuer, error) {
	client := oidc.DefaultClient()
	doc, err := oidc.Discover(ctx, a.realmURL, client)
	if err != nil {
		return model.Issuer{}, err
	}
	iss := model.Issuer{
		Name:               a.name,
		Type:               model.IssuerKeycloak,
		IssuerURL:          doc.Issuer,
		JWKSURI:            doc.JWKSURI,
		DiscoveryDoc:       doc.Raw,
		ControlsPrivateKey: true,
	}
	if set, err := oidc.FetchJWKS(ctx, doc.JWKSURI, client); err == nil {
		iss.Keys = oidc.Keys(set, a.now())
	}
	// Merge in the Keyway-operated key(s) so probe planning can find one to sign with.
	iss.Keys = append(iss.Keys, a.keys.Keys()...)
	return iss, nil
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
