// Package mutations generates benchmark scenarios: known contract changes (true
// positives) and no-op changes (false positives). Roughly half of each keeps the
// FPR meaningful (PRD §13.2).
package mutations

import (
	"fmt"

	"github.com/nometria/keyway/internal/model"
)

// Expected is the ground truth for a scenario.
type Expected struct {
	Detected bool
	Class    model.ChangeClass
	Severity model.Severity
	Consumer string
}

// Scenario is a before/after consumer-graph pair with its expected diff outcome.
type Scenario struct {
	ID       string
	Name     string
	Before   []model.Consumer
	After    []model.Consumer
	Expected Expected
}

func intp(i int) *int    { return &i }
func boolp(b bool) *bool { return &b }

// baseConsumer returns a representative consumer.
func baseConsumer(id string) model.Consumer {
	return model.Consumer{
		ID: id, StableID: id, Name: id, Kind: model.ConsumerService,
		Expects: model.Expectations{
			Issuers:      []string{"https://kc/realms/main"},
			Audiences:    []string{id},
			Algorithms:   []string{"RS256"},
			ClockSkewSec: 60,
		},
		JWKSBehavior: model.JWKSBehavior{CacheTTLSec: intp(300), RefreshesOnUnknownKID: boolp(true), Source: model.SrcConfig},
		Confidence:   map[string]float64{"overall": 1.0},
		Probeable:    true,
	}
}

func clone(c model.Consumer) model.Consumer {
	c.Expects.Issuers = append([]string{}, c.Expects.Issuers...)
	c.Expects.Audiences = append([]string{}, c.Expects.Audiences...)
	c.Expects.Algorithms = append([]string{}, c.Expects.Algorithms...)
	c.Expects.RequiredClaims = append([]string{}, c.Expects.RequiredClaims...)
	if c.JWKSBehavior.CacheTTLSec != nil {
		c.JWKSBehavior.CacheTTLSec = intp(*c.JWKSBehavior.CacheTTLSec)
	}
	if c.JWKSBehavior.RefreshesOnUnknownKID != nil {
		c.JWKSBehavior.RefreshesOnUnknownKID = boolp(*c.JWKSBehavior.RefreshesOnUnknownKID)
	}
	return c
}

// truePositives enumerates one scenario per classification-table row (PRD §9.2).
func truePositives(id string) []Scenario {
	c := baseConsumer(id)
	mk := func(name string, mutate func(*model.Consumer), class model.ChangeClass, sev model.Severity) Scenario {
		after := clone(c)
		mutate(&after)
		return Scenario{
			ID: id, Name: name, Before: []model.Consumer{c}, After: []model.Consumer{after},
			Expected: Expected{Detected: true, Class: class, Severity: sev, Consumer: id},
		}
	}
	return []Scenario{
		mk("audience-added", func(a *model.Consumer) { a.Expects.Audiences = append(a.Expects.Audiences, "extra") }, model.ChangeWidened, model.SeverityMedium),
		mk("audience-removed", func(a *model.Consumer) { a.Expects.Audiences = []string{} }, model.ChangeNarrowed, model.SeverityLow),
		mk("issuer-added", func(a *model.Consumer) { a.Expects.Issuers = append(a.Expects.Issuers, "https://other") }, model.ChangeWidened, model.SeverityHigh),
		mk("alg-none-added", func(a *model.Consumer) { a.Expects.Algorithms = append(a.Expects.Algorithms, "none") }, model.ChangeWidened, model.SeverityCritical),
		mk("required-claim-added", func(a *model.Consumer) { a.Expects.RequiredClaims = []string{"dept"} }, model.ChangeNarrowed, model.SeverityLow),
		mk("clock-skew-increased", func(a *model.Consumer) { a.Expects.ClockSkewSec = 300 }, model.ChangeWidened, model.SeverityMedium),
		mk("refresh-off", func(a *model.Consumer) { a.JWKSBehavior.RefreshesOnUnknownKID = boolp(false) }, model.ChangeNarrowed, model.SeverityHigh),
		mk("cache-ttl-increased", func(a *model.Consumer) { a.JWKSBehavior.CacheTTLSec = intp(3600) }, model.ChangeNarrowed, model.SeverityMedium),
	}
}

// noOps enumerate changes that must NOT affect the contract (PRD §13.2). Each
// mutates something the hash/diff deliberately ignores.
func noOps(id string) []Scenario {
	c := baseConsumer(id)
	mk := func(name string, mutate func(*model.Consumer)) Scenario {
		after := clone(c)
		mutate(&after)
		return Scenario{
			ID: id, Name: name, Before: []model.Consumer{c}, After: []model.Consumer{after},
			Expected: Expected{Detected: false, Consumer: id},
		}
	}
	// Each mutates a field the diff deliberately does not walk (only expects.*
	// and jwks_behavior.{refresh,cache} produce events), so all yield zero events.
	return []Scenario{
		mk("owner-relabel", func(a *model.Consumer) { a.OwnerTeam = "renamed-team" }),
		mk("confidence-wobble", func(a *model.Consumer) { a.Confidence = map[string]float64{"overall": 0.95} }),
		mk("endpoint-added", func(a *model.Consumer) {
			a.Endpoints = append(a.Endpoints, model.Endpoint{URL: "http://x", Method: "GET"})
		}),
		mk("provenance-refreshed", func(a *model.Consumer) {
			a.Provenance = map[string][]model.ProvenanceRecord{"expects.audiences": {{Source: "istio", Locator: "f.yaml"}}}
		}),
		mk("library-detected", func(a *model.Consumer) {
			a.Library = &model.LibraryInfo{Name: "MicahParks/keyfunc", Version: "v2.0.0"}
		}),
		mk("jwks-source-refined", func(a *model.Consumer) { a.JWKSBehavior.Source = model.SrcProbed }),
		mk("refresh-interval-learned", func(a *model.Consumer) { a.JWKSBehavior.RefreshIntervalSec = intp(3600) }),
		mk("probeable-toggled", func(a *model.Consumer) { a.Probeable = !a.Probeable }),
	}
}

// Generate produces a corpus of `rounds` copies of every TP and no-op scenario,
// yielding roughly 50/50 true/false positives (PRD §13.2).
func Generate(rounds int) []Scenario {
	if rounds < 1 {
		rounds = 1
	}
	var out []Scenario
	for r := 0; r < rounds; r++ {
		id := fmt.Sprintf("svc-%04d", r)
		for _, s := range truePositives(id) {
			s.ID = fmt.Sprintf("tp-%s-%s", id, s.Name)
			out = append(out, s)
		}
		for _, s := range noOps(id) {
			s.ID = fmt.Sprintf("np-%s-%s", id, s.Name)
			out = append(out, s)
		}
	}
	return out
}
