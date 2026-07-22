// Package mutations injects known contract changes into a running scenario so
// the harness can measure detection (PRD §13). Roughly 50% of corpus scenarios
// are true mutations and 50% are no-ops (dependency bumps, comment edits,
// replica counts) — without the no-op half the FPR number is meaningless.
package mutations

// Mutation describes a single change to apply to a scenario.
type Mutation struct {
	Target    string `yaml:"target"`    // consumer StableID
	Field     string `yaml:"field"`     // dotted path, e.g. expects.audiences
	Operation string `yaml:"operation"` // add|remove|increase|decrease|set|noop
	Value     any    `yaml:"value"`
}

// IsNoOp reports whether this mutation is expected to produce zero change
// events (the false-positive half of the corpus).
func (m Mutation) IsNoOp() bool { return m.Operation == "noop" }

// TODO(M8): Apply rewrites the target scenario's Istio/Envoy/K8s manifests to
// realise the mutation, and Revert restores them.
