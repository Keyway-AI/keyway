// Package attribution binds contract changes to their cause: a git commit/PR, a
// Kubernetes deploy, or a Keycloak admin event (PRD §16 OPEN-5).
package attribution

import (
	"context"

	"github.com/architsharma/keyway/internal/model"
)

// Attributor resolves the cause of a change event.
type Attributor interface {
	Attribute(ctx context.Context, ev model.ChangeEvent) (*model.Attribution, error)
}

// Unattributed is the fallback used when no source can claim a change. v1 covers
// git and Keycloak admin events; everything else is unattributed (PRD OPEN-5).
func Unattributed() *model.Attribution {
	return &model.Attribution{Kind: "unattributed", Confidence: 0}
}

// TODO(M9): git attributor (blame the changed Istio/Envoy config path to a
// commit/PR), K8s deploy attributor, Keycloak admin-events attributor.
