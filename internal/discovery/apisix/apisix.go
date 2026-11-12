// Package apisix discovers consumers from Apache APISIX declarative
// configuration (the standalone apisix.yaml, or exported admin-API route
// objects). A route whose `plugins` map carries a JWT-validating plugin is a
// token-validating component:
//
//   - `openid-connect` declares the trusted issuer (via its `discovery` URL) and
//     an audience (conventionally `client_id`), mapping to a high-confidence
//     consumer;
//   - `jwt-auth` validates the signature and expiry but keys issuer/audience off
//     per-consumer credentials, so it yields a lower-confidence consumer.
//
// APISIX attaches plugins per route/service as a map keyed by plugin name, which
// is the one structural difference from the Kong source.
package apisix

import (
	"context"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/model"
)

// Discoverer parses APISIX declarative config under scope.ConfigPaths.
type Discoverer struct {
	now func() time.Time
}

// New constructs an APISIX discoverer.
func New() *Discoverer { return &Discoverer{now: time.Now} }

// WithClock injects a clock (tests).
func (d *Discoverer) WithClock(now func() time.Time) *Discoverer { d.now = now; return d }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "apisix" }

type apisixRoute struct {
	Name    string                    `yaml:"name"`
	ID      string                    `yaml:"id"`
	URI     string                    `yaml:"uri"`
	Uris    []string                  `yaml:"uris"`
	Plugins map[string]map[string]any `yaml:"plugins"`
}

type apisixConfig struct {
	Routes   []apisixRoute `yaml:"routes"`
	Services []apisixRoute `yaml:"services"` // services carry plugins the same way
}

// Discover finds JWT-validating routes/services in APISIX config. A document
// qualifies only if it has a `routes` or `services` list whose entries carry a
// recognised JWT plugin, so unrelated manifests never false-match.
func (d *Discoverer) Discover(_ context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	if len(scope.ConfigPaths) == 0 {
		return nil, nil
	}
	var consumers []model.Consumer
	err := discovery.WalkYAML(scope.ConfigPaths, func(path string, doc []byte) error {
		var cfg apisixConfig
		if err := yaml.Unmarshal(doc, &cfg); err != nil {
			return nil
		}
		var out []model.Consumer
		for _, r := range append(cfg.Routes, cfg.Services...) {
			if c, ok := d.toConsumer(r, path); ok {
				out = append(out, c)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].StableID < out[j].StableID })
		consumers = append(consumers, out...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

func (d *Discoverer) toConsumer(r apisixRoute, path string) (model.Consumer, bool) {
	var (
		issuers    []string
		audiences  []string
		jwksURI    string
		confidence float64
		pluginTag  string
	)
	if oidc, ok := r.Plugins["openid-connect"]; ok {
		if iss := oidcIssuer(oidc); iss != "" {
			issuers = appendUnique(issuers, iss)
		}
		// client_id is the conventional audience an OIDC resource binds to.
		if cid := configString(oidc, "client_id"); cid != "" {
			audiences = appendUnique(audiences, cid)
		}
		audiences = unionStrings(audiences, configStrings(oidc, "audience", "audiences"))
		if u := configString(oidc, "jwks_uri"); u != "" {
			jwksURI = u
		}
		confidence = 0.95
		pluginTag = "openid-connect"
	}
	if _, ok := r.Plugins["jwt-auth"]; ok && pluginTag == "" {
		confidence = 0.6 // signature + default expiry check; issuer/audience is per-credential
		pluginTag = "jwt-auth"
	}
	if pluginTag == "" {
		return model.Consumer{}, false
	}

	name := routeName(r)
	stableID := discovery.StableID(discovery.IDParts{Gateway: "apisix", Route: name})
	source := "apisix:" + pluginTag + "/" + name
	prov := model.ProvenanceRecord{Source: source, Locator: path, ObservedAt: d.now(), Confidence: confidence}

	provMap := map[string][]model.ProvenanceRecord{"expects.issuers": {prov}, "expects.audiences": {prov}}
	jwks := model.JWKSBehavior{Source: model.SrcConfig}
	if jwksURI != "" {
		jwks.JWKSURI = jwksURI
		provMap["jwks_behavior.jwks_uri"] = []model.ProvenanceRecord{prov}
	}

	return model.Consumer{
		ID:       stableID,
		StableID: stableID,
		Kind:     model.ConsumerGatewayRoute,
		Name:     name,
		Expects: model.Expectations{
			Issuers:   sortStrings(issuers),
			Audiences: sortStrings(audiences),
		},
		JWKSBehavior: jwks,
		Provenance:   provMap,
		Confidence:   map[string]float64{"overall": confidence},
		Probeable:    true,
	}, true
}

func routeName(r apisixRoute) string {
	switch {
	case r.Name != "":
		return r.Name
	case r.ID != "":
		return r.ID
	case r.URI != "":
		return strings.ReplaceAll(strings.TrimPrefix(r.URI, "/"), "/", "_")
	case len(r.Uris) > 0:
		return strings.ReplaceAll(strings.TrimPrefix(r.Uris[0], "/"), "/", "_")
	default:
		return "route"
	}
}

// oidcIssuer derives the issuer from the openid-connect plugin's `discovery` URL
// (or explicit `issuer`), stripping the well-known suffix.
func oidcIssuer(cfg map[string]any) string {
	iss := configString(cfg, "issuer")
	if iss == "" {
		iss = configString(cfg, "discovery")
	}
	iss = strings.TrimSuffix(iss, "/.well-known/openid-configuration")
	iss = strings.TrimSuffix(iss, "/.well-known/oauth-authorization-server")
	return strings.TrimSpace(iss)
}

// --- small helpers -----------------------------------------------------------

func configString(cfg map[string]any, key string) string {
	if cfg == nil {
		return ""
	}
	if s, ok := cfg[key].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func configStrings(cfg map[string]any, keys ...string) []string {
	var out []string
	if cfg == nil {
		return out
	}
	for _, k := range keys {
		switch v := cfg[k].(type) {
		case []any:
			for _, e := range v {
				if s, ok := e.(string); ok {
					if s = strings.TrimSpace(s); s != "" {
						out = appendUnique(out, s)
					}
				}
			}
		case string:
			if s := strings.TrimSpace(v); s != "" {
				out = appendUnique(out, s)
			}
		}
	}
	return out
}

func appendUnique(ss []string, v string) []string {
	for _, s := range ss {
		if s == v {
			return ss
		}
	}
	return append(ss, v)
}

func unionStrings(a, b []string) []string {
	out := append([]string{}, a...)
	for _, v := range b {
		out = appendUnique(out, v)
	}
	return out
}

func sortStrings(ss []string) []string {
	out := append([]string{}, ss...)
	sort.Strings(out)
	return out
}
