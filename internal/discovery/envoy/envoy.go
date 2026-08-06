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
		for _, name := range names {
			if c, ok := d.toConsumer(name, provs[name], path, scope); ok {
				consumers = append(consumers, c)
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
			Issuers:   []string{p.issuer},
			Audiences: p.audiences,
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
