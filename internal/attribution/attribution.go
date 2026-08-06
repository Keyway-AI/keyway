// Package attribution binds contract changes to their cause: a git commit/PR, a
// Kubernetes deploy, or a Keycloak admin event (PRD §16 OPEN-5).
package attribution

import (
	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/ports"
)

// Attributor resolves the cause of a change event. It is an alias of the shared
// ports.Attributor so producers here and the contract consumer share one type.
type Attributor = ports.Attributor

// Unattributed is the fallback used when no source can claim a change. v1 covers
// git commits, K8s deploy annotations, and Keycloak admin events; everything
// else is unattributed (PRD OPEN-5).
func Unattributed() *model.Attribution {
	return &model.Attribution{Kind: "unattributed", Confidence: 0}
}
