// Package envoy discovers consumers from Envoy jwt_authn provider configuration.
// cache_duration is a direct, high-confidence read of JWKS behavior, which is
// why this adapter is prioritized for that signal (PRD §7.2).
package envoy

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/model"
)

// Discoverer parses Envoy static config or admin /config_dump.
type Discoverer struct {
	now func() time.Time
}

// New constructs an Envoy discoverer.
func New() *Discoverer { return &Discoverer{now: time.Now} }

// WithClock injects a clock (tests).
func (d *Discoverer) WithClock(now func() time.Time) *Discoverer { d.now = now; return d }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "envoy" }

// Discover finds jwt_authn providers under scope.ConfigPaths. Each provider with
// an issuer becomes a gateway_route consumer.
func (d *Discoverer) Discover(_ context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	if len(scope.ConfigPaths) == 0 {
		return nil, nil
	}
	var consumers []model.Consumer
	err := discovery.WalkYAML(scope.ConfigPaths, func(path string, doc []byte) error {
		var root any
		if err := yaml.Unmarshal(doc, &root); err != nil {
			return nil
		}
		provs := findProviders(root)
		// Iterate in sorted order so discovery output is deterministic across runs
		// (findProviders returns a map). (KI-31)
		names := make([]string, 0, len(provs))
		for n := range provs {
			names = append(names, n)
		}
		sort.Strings(names)
		byName := make(map[string]model.Consumer, len(names))
		for _, name := range names {
			if c, ok := d.toConsumer(name, provs[name], path, scope); ok {
				byName[name] = c
			}
		}
		// A route that accepts several providers via requires_any is ONE validation
		// point trusting multiple issuers, not several independent single-issuer
		// consumers. Merge each such group into one consumer so the multiple issuers
		// are visible (P4-multi-issuer-trust) and the route is counted once, not N
		// times. Providers not OR'd with others keep their own consumer. (KI-34)
		merged := map[string]bool{}
		for _, group := range findRequiresAnyGroups(root) {
			var base string
			for _, n := range group { // group is returned sorted; first present wins
				if _, ok := byName[n]; ok && !merged[n] {
					base = n
					break
				}
			}
			if base == "" {
				continue
			}
			bc := byName[base]
			for _, n := range group {
				if n == base || merged[n] {
					continue
				}
				if other, ok := byName[n]; ok {
					bc = mergeProviders(bc, other)
					merged[n] = true
				}
			}
			byName[base] = bc
		}
		for _, name := range names {
			if !merged[name] {
				consumers = append(consumers, byName[name])
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

// provider is a normalised jwt_authn provider.
type provider struct {
	issuer        string
	audiences     []string
	cacheDuration string
	jwksURI       string
	clockSkewSec  int
}

// findProviders recursively locates a "providers" map under any jwt_authn
// typed_config. Each value with an "issuer" is treated as a provider.
func findProviders(node any) map[string]provider {
	out := map[string]provider{}
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if provs, ok := v["providers"].(map[string]any); ok {
				for name, p := range provs {
					if _, exists := out[name]; exists {
						continue // keep the first definition, deterministically (KI-31)
					}
					if pm, ok := p.(map[string]any); ok {
						if iss, ok := pm["issuer"].(string); ok && strings.TrimSpace(iss) != "" {
							out[name] = provider{
								issuer:        strings.TrimSpace(iss), // KI-26: whitespace is never part of an issuer
								audiences:     toStrings(pm["audiences"]),
								cacheDuration: cacheDurationOf(pm),
								jwksURI:       remoteJWKSURI(pm),
								clockSkewSec:  clockSkewOf(pm),
							}
						}
					}
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(node)
	return out
}

// findRequiresAnyGroups returns, for each jwt_authn requires_any that OR's two or
// more providers, the sorted list of provider_names in that disjunction. These
// providers serve a single route together (any one token satisfies it), so they
// describe one consumer trusting several issuers. requirement_map indirection and
// requires_all (which narrows, not widens, trust) are intentionally not grouped.
func findRequiresAnyGroups(node any) [][]string {
	var groups [][]string
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if ra, ok := v["requires_any"].(map[string]any); ok {
				if reqs, ok := ra["requirements"].([]any); ok {
					var names []string
					for _, r := range reqs {
						if rm, ok := r.(map[string]any); ok {
							if pn, ok := rm["provider_name"].(string); ok {
								if pn = strings.TrimSpace(pn); pn != "" {
									names = appendUniqueStr(names, pn)
								}
							}
						}
					}
					if len(names) >= 2 {
						sort.Strings(names)
						groups = append(groups, names)
					}
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(node)
	return groups
}

// mergeProviders folds b into a: it unions issuers and audiences, keeps the wider
// clock skew, and concatenates provenance. a's identity (StableID, name) is kept,
// so the merged route is the sorted-first provider of the requires_any group.
func mergeProviders(a, b model.Consumer) model.Consumer {
	a.Expects.Issuers = sortedUnion(a.Expects.Issuers, b.Expects.Issuers)
	a.Expects.Audiences = sortedUnion(a.Expects.Audiences, b.Expects.Audiences)
	if b.Expects.ClockSkewSec > a.Expects.ClockSkewSec {
		a.Expects.ClockSkewSec = b.Expects.ClockSkewSec
	}
	for k, recs := range b.Provenance {
		a.Provenance[k] = append(a.Provenance[k], recs...)
	}
	return a
}

func appendUniqueStr(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}

func sortedUnion(a, b []string) []string {
	out := append([]string{}, a...)
	for _, v := range b {
		out = appendUniqueStr(out, v)
	}
	sort.Strings(out)
	return out
}

func (d *Discoverer) toConsumer(name string, p provider, path string, scope discovery.Scope) (model.Consumer, bool) {
	if p.issuer == "" {
		return model.Consumer{}, false
	}
	stableID := discovery.StableID(discovery.IDParts{Gateway: "envoy", Route: name})

	jwks := model.JWKSBehavior{JWKSURI: p.jwksURI, Source: model.SrcConfig} // rotation endpoint (KI-32)
	if sec, ok := parseDuration(p.cacheDuration); ok {
		jwks.CacheTTLSec = &sec
	}

	source := "envoy:jwt_authn/" + name
	prov := model.ProvenanceRecord{Source: source, Locator: path, ObservedAt: d.now(), Confidence: 1.0}
	return model.Consumer{
		ID:        stableID,
		StableID:  stableID,
		Kind:      model.ConsumerGatewayRoute,
		Name:      name,
		Namespace: scope.KubeContext,
		Expects: model.Expectations{
			Issuers:      []string{p.issuer},
			Audiences:    p.audiences,
			ClockSkewSec: p.clockSkewSec,
		},
		JWKSBehavior: jwks,
		Provenance: map[string][]model.ProvenanceRecord{
			"expects.issuers":         {prov},
			"expects.audiences":       {prov},
			"jwks_behavior.cache_ttl": {prov},
		},
		Confidence: map[string]float64{
			"overall":           1.0,
			"expects.issuers":   1.0,
			"expects.audiences": 1.0,
		},
		Probeable: true,
	}, true
}

// remoteJWKSURI extracts the JWKS endpoint from a provider's remote_jwks config.
func remoteJWKSURI(pm map[string]any) string {
	rj, ok := pm["remote_jwks"].(map[string]any)
	if !ok {
		return ""
	}
	hu, ok := rj["http_uri"].(map[string]any)
	if !ok {
		return ""
	}
	if uri, ok := hu["uri"].(string); ok {
		return uri
	}
	return ""
}

// clockSkewOf reads the provider's explicit clock-skew tolerance. Envoy's
// jwt_authn applies a 60s default when clock_skew_seconds is absent, but we
// record only what the config declares (0 = unset): the measurement flags an
// explicit wide skew, never an unset provider relying on the default. Accepts
// the proto snake_case and the camelCase some manifests use.
func clockSkewOf(pm map[string]any) int {
	for _, k := range []string{"clock_skew_seconds", "clockSkewSeconds"} {
		if v, ok := pm[k]; ok {
			switch n := v.(type) {
			case int:
				return n
			case int64:
				return int(n)
			case float64:
				return int(n)
			case string:
				if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

func cacheDurationOf(pm map[string]any) string {
	rj, ok := pm["remote_jwks"].(map[string]any)
	if !ok {
		return ""
	}
	switch cd := rj["cache_duration"].(type) {
	case string:
		return cd
	case map[string]any:
		// Protobuf Duration form: {seconds: 600}
		if s, ok := cd["seconds"]; ok {
			return toString(s) + "s"
		}
	}
	return ""
}

// parseDuration parses Envoy-style durations like "600s" or "600.5s".
func parseDuration(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if d, err := time.ParseDuration(s); err == nil {
		return int(d.Seconds()), true
	}
	if n, err := strconv.Atoi(strings.TrimSuffix(s, "s")); err == nil {
		return n, true
	}
	return 0, false
}

func toStrings(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			if s = strings.TrimSpace(s); s != "" { // KI-26
				out = append(out, s)
			}
		}
	}
	return out
}

func toString(v any) string {
	switch n := v.(type) {
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case float64:
		return strconv.FormatInt(int64(n), 10)
	case string:
		return n
	default:
		return ""
	}
}
