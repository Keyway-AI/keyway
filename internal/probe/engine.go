package probe

import (
	"context"
	"strings"
	"time"

	"github.com/architsharma/keyway/internal/model"
)

// EngineConfig controls probe execution (PRD §6.3). Defaults come from
// DefaultEngineConfig.
type EngineConfig struct {
	MaxConcurrentPerConsumer int           // default 2
	MaxConcurrentGlobal      int           // default 20
	RequestTimeout           time.Duration // default 10s
	InterProbeDelay          time.Duration // default 200ms
	AbortOnConsecutive5xx    int           // default 3
	DryRun                   bool

	// Allowlist of host substrings the engine may target. Empty = deny all
	// unless AllowProduction is set (the --i-know-this-is-production override).
	Allowlist       []string
	AllowProduction bool
}

// DefaultEngineConfig returns the PRD §6.3 defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxConcurrentPerConsumer: 2,
		MaxConcurrentGlobal:      20,
		RequestTimeout:           10 * time.Second,
		InterProbeDelay:          200 * time.Millisecond,
		AbortOnConsecutive5xx:    3,
	}
}

// HostAllowed reports whether the engine may target a host (staging guard).
// Default deny: an empty allowlist without the production override blocks all.
func (c EngineConfig) HostAllowed(host string) bool {
	if c.AllowProduction {
		return true
	}
	for _, allowed := range c.Allowlist {
		if allowed != "" && strings.Contains(host, allowed) {
			return true
		}
	}
	return false
}

// Engine runs probes against consumers.
type Engine struct {
	cfg    EngineConfig
	probes []Probe
}

// NewEngine constructs an engine with the default probe set.
func NewEngine(cfg EngineConfig) *Engine {
	return &Engine{cfg: cfg, probes: Definitions()}
}

// Run executes the applicable probes against the given consumers.
//
// TODO(M3): implement concurrency (per-consumer and global caps), the staging
// guard (refuse hosts failing HostAllowed), the kill switch poll, the
// consecutive-5xx abort, and the baseline-5xx "unverified, not a finding" rule.
func (e *Engine) Run(ctx context.Context, iss model.Issuer, consumers []model.Consumer) ([]model.ProbeResult, error) {
	_ = ctx
	_ = iss
	_ = consumers
	return nil, model.ErrUnsupported // replaced with real execution in M3
}
