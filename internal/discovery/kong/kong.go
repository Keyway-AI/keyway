// Package kong discovers consumers from Kong Gateway declarative configuration
// (the `_format_version` kong.yml / decK format). A Kong route or service that
// carries a JWT-validating plugin is a token-validating component:
//
//   - the `openid-connect` plugin declares the trusted `issuer` and `audience`
//     directly, so it maps to a high-confidence consumer the same way an Envoy
//     jwt_authn provider does;
//   - the classic `jwt` plugin validates the signature and the claims named in
//     `claims_to_verify` (e.g. `exp`) but keys issuer/audience off per-consumer
//     credentials rather than plugin config, so it yields a lower-confidence
//     consumer that still records which claims are enforced.
//
// Plugins attach at global, service, or route scope; a route's effective plugin
// set is global ∪ service ∪ route, with the narrower scope winning on conflict.
package kong

import (
	"sort"
	"strings"
	"time"

	"context"

	"gopkg.in/yaml.v3"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/model"
)

// Discoverer parses Kong declarative config under scope.ConfigPaths.
type Discoverer struct {
	now func() time.Time
}

// New constructs a Kong discoverer.
func New() *Discoverer { return &Discoverer{now: time.Now} }

// WithClock injects a clock (tests).
func (d *Discoverer) WithClock(now func() time.Time) *Discoverer { d.now = now; return d }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "kong" }

type kongPlugin struct {
	Name   string         `yaml:"name"`
	Config map[string]any `yaml:"config"`
}

type kongRoute struct {
	Name    string       `yaml:"name"`
	Paths   []string     `yaml:"paths"`
	Hosts   []string     `yaml:"hosts"`
	Plugins []kongPlugin `yaml:"plugins"`
}

type kongService struct {
	Name    string       `yaml:"name"`
	Host    string       `yaml:"host"`
	URL     string       `yaml:"url"`
	Routes  []kongRoute  `yaml:"routes"`
	Plugins []kongPlugin `yaml:"plugins"`
}

type kongConfig struct {
	FormatVersion string        `yaml:"_format_version"`
	Plugins       []kongPlugin  `yaml:"plugins"`
	Services      []kongService `yaml:"services"`
	Routes        []kongRoute   `yaml:"routes"`
}

// Discover finds JWT-validating routes/services in Kong declarative config. Only
// documents that carry a `_format_version` key are treated as Kong config, so a
// repo of mixed manifests never false-matches.
func (d *Discoverer) Discover(_ context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	if len(scope.ConfigPaths) == 0 {
		return nil, nil
	}
	var consumers []model.Consumer
	err := discovery.WalkYAML(scope.ConfigPaths, func(path string, doc []byte) error {
		var cfg kongConfig
		if err := yaml.Unmarshal(doc, &cfg); err != nil {
			return nil
		}
		if strings.TrimSpace(cfg.FormatVersion) == "" {
			return nil // not a Kong declarative document
		}
		consumers = append(consumers, d.fromConfig(cfg, path)...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

func (d *Discoverer) fromConfig(cfg kongConfig, path string) []model.Consumer {
	var out []model.Consumer
	emit := func(name string, plugins []kongPlugin) {
		if c, ok := d.toConsumer(name, plugins, path); ok {
			out = append(out, c)
		}
	}
	for _, svc := range cfg.Services {
		svcName := svc.Name
		if svcName == "" {
			svcName = svc.Host
		}
		if len(svc.Routes) == 0 {
			// A service with no inline routes still validates tokens if a JWT plugin
			// is attached at service or global scope.
			emit(svcName, mergePlugins(cfg.Plugins, svc.Plugins))
			continue
		}
		for _, r := range svc.Routes {
			emit(routeName(r, svcName), mergePlugins(cfg.Plugins, svc.Plugins, r.Plugins))
		}
	}
	// Top-level routes (decK allows routes outside a service block).
	for _, r := range cfg.Routes {
		emit(routeName(r, "route"), mergePlugins(cfg.Plugins, r.Plugins))
	}
	// Deterministic ordering across runs.
	sort.Slice(out, func(i, j int) bool { return out[i].StableID < out[j].StableID })
	return out
}

// mergePlugins overlays plugin lists from widest to narrowest scope, keyed by
// plugin name so a route-level plugin overrides the same plugin set globally.
func mergePlugins(lists ...[]kongPlugin) []kongPlugin {
	byName := map[string]kongPlugin{}
	var order []string
	for _, list := range lists {
		for _, p := range list {
			if _, exists := byName[p.Name]; !exists {
				order = append(order, p.Name)
			}
			byName[p.Name] = p // later (narrower) scope wins
		}
	}
	out := make([]kongPlugin, 0, len(order))
	for _, n := range order {
		out = append(out, byName[n])
	}
	return out
}

func routeName(r kongRoute, fallback string) string {
	if r.Name != "" {
		return r.Name
	}
	if len(r.Paths) > 0 {
		return fallback + strings.ReplaceAll(r.Paths[0], "/", "_")
	}
	return fallback
}

// toConsumer builds a consumer from a route/service's effective plugin set, or
// reports false if none of the plugins validate JWTs.
func (d *Discoverer) toConsumer(name string, plugins []kongPlugin, path string) (model.Consumer, bool) {
	var (
		issuers    []string
		audiences  []string
		required   []string
		jwksURI    string
		confidence float64
		pluginTag  string
	)
	for _, p := range plugins {
		switch p.Name {
		case "openid-connect":
			if iss := oidcIssuer(p.Config); iss != "" {
				issuers = appendUnique(issuers, iss)
			}
			audiences = unionStrings(audiences, configStrings(p.Config, "audience", "audience_required", "audiences"))
			if u := configString(p.Config, "jwks_uri"); u != "" {
				jwksURI = u
			}
			confidence = 0.95
			pluginTag = "openid-connect"
		case "jwt":
			required = unionStrings(required, configStrings(p.Config, "claims_to_verify"))
			if confidence == 0 {
				confidence = 0.6 // signature + listed claims, but issuer/audience is per-credential
				pluginTag = "jwt"
			}
		}
	}
	if pluginTag == "" {
		return model.Consumer{}, false
	}
	stableID := discovery.StableID(discovery.IDParts{Gateway: "kong", Route: name})
	source := "kong:" + pluginTag + "/" + name
	prov := model.ProvenanceRecord{Source: source, Locator: path, ObservedAt: d.now(), Confidence: confidence}

	provMap := map[string][]model.ProvenanceRecord{"expects.issuers": {prov}, "expects.audiences": {prov}}
	jwks := model.JWKSBehavior{Source: model.SrcConfig}
	if jwksURI != "" {
		jwks.JWKSURI = jwksURI
		provMap["jwks_behavior.jwks_uri"] = []model.ProvenanceRecord{prov}
	}
	if len(required) > 0 {
		provMap["expects.required_claims"] = []model.ProvenanceRecord{prov}
	}

	return model.Consumer{
		ID:        stableID,
		StableID:  stableID,
		Kind:      model.ConsumerGatewayRoute,
		Name:      name,
		Endpoints: nil,
		Expects: model.Expectations{
			Issuers:        sortStrings(issuers),
			Audiences:      sortStrings(audiences),
			RequiredClaims: sortStrings(required),
		},
		JWKSBehavior: jwks,
		Provenance:   provMap,
		Confidence:   map[string]float64{"overall": confidence},
		Probeable:    true,
	}, true
}

// oidcIssuer extracts the issuer from an openid-connect plugin's `issuer`, which
// is usually the discovery URL; strip the well-known suffix to get the issuer.
func oidcIssuer(cfg map[string]any) string {
	iss := configString(cfg, "issuer")
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
