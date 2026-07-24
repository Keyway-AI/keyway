package discovery

import (
	"context"

	"github.com/nometria/keyway/internal/model"
)

// Run executes every discoverer over the scope and merges the results by
// StableID, so a consumer seen by both Istio (issuer/audience) and Kubernetes
// (endpoints, owner) becomes one enriched record. Errors from individual
// discoverers are returned joined so a single bad source does not lose the rest.
func Run(ctx context.Context, scope Scope, discoverers ...Discoverer) ([]model.Consumer, error) {
	var all [][]model.Consumer
	var firstErr error
	for _, d := range discoverers {
		cs, err := d.Discover(ctx, scope)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if len(cs) > 0 {
			all = append(all, cs)
		}
	}
	return Merge(all...), firstErr
}

// Merge combines consumer sets by StableID, unioning expectations, endpoints,
// provenance and confidence.
func Merge(sets ...[]model.Consumer) []model.Consumer {
	order := []string{}
	byID := map[string]model.Consumer{}
	for _, set := range sets {
		for _, c := range set {
			existing, ok := byID[c.StableID]
			if !ok {
				order = append(order, c.StableID)
				byID[c.StableID] = c
				continue
			}
			byID[c.StableID] = mergeConsumers(existing, c)
		}
	}
	out := make([]model.Consumer, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func mergeConsumers(a, b model.Consumer) model.Consumer {
	a.Name = firstNonEmpty(a.Name, b.Name)
	a.Namespace = firstNonEmpty(a.Namespace, b.Namespace)
	a.OwnerTeam = firstNonEmpty(a.OwnerTeam, b.OwnerTeam)
	if a.Kind == "" {
		a.Kind = b.Kind
	}

	a.Expects.Issuers = unionStrings(a.Expects.Issuers, b.Expects.Issuers)
	a.Expects.Audiences = unionStrings(a.Expects.Audiences, b.Expects.Audiences)
	a.Expects.Algorithms = unionStrings(a.Expects.Algorithms, b.Expects.Algorithms)
	a.Expects.RequiredClaims = unionStrings(a.Expects.RequiredClaims, b.Expects.RequiredClaims)
	if b.Expects.ClockSkewSec > a.Expects.ClockSkewSec {
		a.Expects.ClockSkewSec = b.Expects.ClockSkewSec
	}

	a.Endpoints = unionEndpoints(a.Endpoints, b.Endpoints)
	a.JWKSBehavior = mergeJWKS(a.JWKSBehavior, b.JWKSBehavior)

	if a.Library == nil {
		a.Library = b.Library
	}
	a.Provenance = mergeProvenance(a.Provenance, b.Provenance)
	a.Confidence = mergeConfidence(a.Confidence, b.Confidence)
	a.Probeable = a.Probeable || b.Probeable
	return a
}

func mergeJWKS(a, b model.JWKSBehavior) model.JWKSBehavior {
	if a.CacheTTLSec == nil {
		a.CacheTTLSec = b.CacheTTLSec
	}
	if a.RefreshIntervalSec == nil {
		a.RefreshIntervalSec = b.RefreshIntervalSec
	}
	if a.RefreshesOnUnknownKID == nil {
		a.RefreshesOnUnknownKID = b.RefreshesOnUnknownKID
	}
	if a.Source == "" {
		a.Source = b.Source
	}
	return a
}

func mergeProvenance(a, b map[string][]model.ProvenanceRecord) map[string][]model.ProvenanceRecord {
	if a == nil {
		a = map[string][]model.ProvenanceRecord{}
	}
	for k, recs := range b {
		a[k] = append(a[k], recs...)
	}
	return a
}

func mergeConfidence(a, b map[string]float64) map[string]float64 {
	if a == nil {
		a = map[string]float64{}
	}
	for k, v := range b {
		if v > a[k] {
			a[k] = v
		}
	}
	return a
}

func unionStrings(a, b []string) []string {
	out := append([]string{}, a...)
	for _, v := range b {
		if !contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func unionEndpoints(a, b []model.Endpoint) []model.Endpoint {
	seen := map[string]struct{}{}
	var out []model.Endpoint
	for _, e := range append(append([]model.Endpoint{}, a...), b...) {
		if _, ok := seen[e.URL]; ok {
			continue
		}
		seen[e.URL] = struct{}{}
		out = append(out, e)
	}
	return out
}

func firstNonEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
