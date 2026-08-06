// Package ports holds the small shared interfaces (hexagonal "ports") that would
// otherwise be duplicated across domain packages only to dodge an import cycle.
// It imports the model leaf and nothing else internal, so any package can depend
// on it without creating a cycle.
package ports

import (
	"context"

	"github.com/Keyway-AI/keyway/internal/model"
)

// Attributor binds a change event to its cause — a git commit/PR, a Kubernetes
// deploy, or an IdP admin action. It is the single definition of the port;
// contract.Attributor and attribution.Attributor are aliases of this type, so
// the producing package (attribution) and the consuming package (contract) share
// one interface without importing each other.
type Attributor interface {
	Attribute(ctx context.Context, ev model.ChangeEvent) (*model.Attribution, error)
}
