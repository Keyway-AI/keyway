package discovery

import (
	"context"
	"strings"

	"github.com/Keyway-AI/keyway/internal/model"
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

// Merge combines consumer sets by identity, unioning expectations, endpoints,
// provenance and confidence. Two consumers merge when their identity sets —
// {StableID} ∪ Aliases — intersect, so the same workload discovered by two
// sources under different primary keys (Istio by service name, Kubernetes by
// service account) becomes one record (KI-28). The first-seen consumer's
// StableID is canonical, so ordering discoverers service-name-first yields the
// more readable identity.
func Merge(sets ...[]model.Consumer) []model.Consumer {
	order := []string{}                 // canonical StableIDs, first-seen order
	byID := map[string]model.Consumer{} // canonical StableID -> merged consumer
	canon := map[string]string{}        // any identity key -> canonical StableID

	for _, set := range sets {
		for _, c := range set {
			target := ""
			for _, k := range identityKeys(c) {
				if cn, ok := canon[k]; ok {
					target = cn
					break
				}
			}
			if target == "" {
				order = append(order, c.StableID)
				byID[c.StableID] = c
				for _, k := range identityKeys(c) {
					canon[k] = c.StableID
				}
				continue
			}
			byID[target] = mergeConsumers(byID[target], c)
			for _, k := range identityKeys(byID[target]) {
				if _, ok := canon[k]; !ok {
					canon[k] = target
				}
			}
		}
	}
	out := make([]model.Consumer, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return aliasFold(out)
}

// identityKeys is the set of stable identities a consumer can be correlated by.
func identityKeys(c model.Consumer) []string {
	keys := make([]string, 0, 1+len(c.Aliases))
	keys = append(keys, c.StableID)
	for _, a := range c.Aliases {
		if a != "" && a != c.StableID && !contains(keys, a) {
			keys = append(keys, a)
		}
	}
	return keys
}

// aliasFold merges an OIDC client-registry consumer (oidc://…) into a
// mesh-discovered consumer (k8s://… / route://…) when they share at least one
// issuer AND one audience — i.e. they are the same logical service seen two
// ways. The mesh consumer wins (it has endpoints and is probeable); the OIDC
// view's data is folded in (KI-24). The match requires BOTH issuer and audience
// to avoid over-merging distinct services that happen to share one.
func aliasFold(cs []model.Consumer) []model.Consumer {
	type pair struct{ iss, aud string }
	meshByPair := map[pair]int{}
	for i, c := range cs {
		if isOIDCStableID(c.StableID) {
			continue
		}
		for _, is := range c.Expects.Issuers {
			for _, au := range c.Expects.Audiences {
				meshByPair[pair{is, au}] = i
			}
		}
	}

	dropped := make([]bool, len(cs))
	for i := range cs {
		c := cs[i]
		if !isOIDCStableID(c.StableID) {
			continue
		}
		for _, is := range c.Expects.Issuers {
			for _, au := range c.Expects.Audiences {
				if j, ok := meshByPair[pair{is, au}]; ok && j != i {
					cs[j] = mergeConsumers(cs[j], c)
					dropped[i] = true
					break
				}
			}
			if dropped[i] {
				break
			}
		}
	}

	out := cs[:0:0]
	for i, c := range cs {
		if !dropped[i] {
			out = append(out, c)
		}
	}
	return out
}

func isOIDCStableID(id string) bool { return strings.HasPrefix(id, "oidc://") }

func mergeConsumers(a, b model.Consumer) model.Consumer {
	a.Name = firstNonEmpty(a.Name, b.Name)
	a.Namespace = firstNonEmpty(a.Namespace, b.Namespace)
	a.OwnerTeam = firstNonEmpty(a.OwnerTeam, b.OwnerTeam)
	if a.Kind == "" {
		a.Kind = b.Kind
	}

	// Keep every alternate identity of the merged-in consumer so the canonical
	// record can still be correlated by any of them.
	a.Aliases = unionStrings(a.Aliases, b.Aliases)
	if b.StableID != a.StableID {
		a.Aliases = unionStrings(a.Aliases, []string{b.StableID})
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
