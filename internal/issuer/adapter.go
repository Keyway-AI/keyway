// Package issuer defines the issuer adapter interface and provides
// per-issuer-type implementations (keycloak, k8ssa, generic).
package issuer

import (
	"context"

	"github.com/architsharma/keyway/internal/model"
)

// Adapter abstracts an issuer Keyway can describe, mint tokens against, and
// (where it controls the private key) run the canary key lifecycle on.
type Adapter interface {
	// Describe returns the current state of the issuer, including its JWKS keys
	// and observed claim schema.
	Describe(ctx context.Context) (model.Issuer, error)

	// MintToken produces a signed JWT with the given claims using the named key.
	// Returns model.ErrNoPrivateKey if ControlsPrivateKey is false.
	MintToken(ctx context.Context, kid string, claims map[string]any) (string, error)

	// AnnounceKey publishes a new key in JWKS WITHOUT using it for signing.
	// This is the canary operation. Returns model.ErrUnsupported where not possible.
	AnnounceKey(ctx context.Context, alg string) (model.Key, error)

	// PromoteKey moves a key from announced -> active (begins signing with it).
	PromoteKey(ctx context.Context, kid string) error

	// RetireKey removes a key from active signing use.
	RetireKey(ctx context.Context, kid string) error
}
