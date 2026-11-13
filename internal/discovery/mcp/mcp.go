// Package mcp discovers consumers from Model Context Protocol (MCP) server
// authorization configuration. The MCP authorization specification requires an
// MCP server to behave as an OAuth 2.0 protected resource and to publish OAuth
// 2.0 Protected Resource Metadata (RFC 9728): a document naming the resource
// identifier the access token must be bound to and the authorization server(s)
// that issue those tokens. That document is exactly a declarative authorization
// contract — `resource` is the audience the token must carry (RFC 8707), and
// `authorization_servers` are the trusted issuers — so each one becomes a
// Consumer.
//
// This is the agent-authorization analog of the Envoy/Istio gateway sources:
// where a mesh declares `issuer` + `audiences` per route, an MCP server declares
// `authorization_servers` + `resource` per protected resource. Discovering it
// lets the audience-binding checks (the SoK's headline MCP threat, token
// passthrough to the wrong resource) run against real MCP deployments.
package mcp

import (
	"context"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/model"
)

// Discoverer parses MCP protected-resource metadata and MCP server config files.
type Discoverer struct {
	now func() time.Time
}

// New constructs an MCP discoverer.
func New() *Discoverer { return &Discoverer{now: time.Now} }

// WithClock injects a clock (tests).
func (d *Discoverer) WithClock(now func() time.Time) *Discoverer { d.now = now; return d }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "mcp" }

// Discover finds MCP protected-resource metadata under scope.ConfigPaths. Each
// distinct resource with at least one authorization server becomes a Consumer.
func (d *Discoverer) Discover(_ context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	if len(scope.ConfigPaths) == 0 {
		return nil, nil
	}
	// Dedup by resource across documents: the same protected-resource metadata is
	// often present both as a served .well-known file and inside a server config.
	seen := map[string]bool{}
	var consumers []model.Consumer
	err := discovery.WalkYAML(scope.ConfigPaths, func(path string, doc []byte) error {
		var root any
		if err := yaml.Unmarshal(doc, &root); err != nil {
			return nil // one unparseable doc never aborts the walk
		}
		metas := findResourceMetadata(root)
		// Deterministic order across runs (findResourceMetadata keys by resource).
		resources := make([]string, 0, len(metas))
		for r := range metas {
			resources = append(resources, r)
		}
		sort.Strings(resources)
		for _, r := range resources {
			if seen[r] {
				continue
			}
			if c, ok := d.toConsumer(metas[r], path); ok {
				seen[r] = true
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

// resourceMeta is a normalised MCP protected-resource authorization contract.
type resourceMeta struct {
	resource  string   // the canonical resource identifier (RFC 9728 "resource")
	issuers   []string // authorization_servers (+ singular issuer/authorization_server)
	audiences []string // resource + any explicit audience(s)
	scopes    []string // scopes_supported / scopes
	jwksURI   string   // jwks_uri, when the server advertises one directly
}

// findResourceMetadata recursively locates MCP protected-resource metadata in a
// decoded YAML/JSON tree, keyed by resource identifier. It accepts two shapes:
//
//  1. RFC 9728 Protected Resource Metadata — an object with a string `resource`
//     and a non-empty `authorization_servers` array (the signature the MCP auth
//     spec mandates). This is what a server serves at
//     `/.well-known/oauth-protected-resource`.
//  2. A nested server-config authorization block — an object under an
//     `authorization`/`auth`/`mcp` key carrying a single `issuer` (or
//     `authorization_server`) plus a `resource` or `audience`.
//
// Requiring either the RFC 9728 pair or an explicit issuer+resource/audience
// pair keeps false positives off generic config maps.
func findResourceMetadata(node any) map[string]resourceMeta {
	out := map[string]resourceMeta{}
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if m, ok := asResourceMeta(v); ok {
				if _, exists := out[m.resource]; !exists {
					out[m.resource] = m // first definition wins, deterministically
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

// asResourceMeta tests one map node for the two accepted shapes and normalises it.
func asResourceMeta(m map[string]any) (resourceMeta, bool) {
	resource := firstString(m, "resource", "resource_identifier", "resource_uri")
	issuers := mergeIssuers(m)
	audiences := strSlice(m, "audience", "audiences", "aud")
	scopes := uniqueStrings(append(strSlice(m, "scopes_supported", "scopes"), strSlice(m, "scope")...))
	jwks := firstString(m, "jwks_uri", "jwksUri")

	// The resource identifier is itself the audience the token must be bound to
	// (RFC 8707 resource indicator → aud), so fold it into audiences.
	if resource != "" {
		audiences = appendUnique(audiences, resource)
	}

	// Shape acceptance: need at least one issuer, and either a resource (RFC 9728)
	// or an explicit audience (a server-config auth block).
	if len(issuers) == 0 {
		return resourceMeta{}, false
	}
	if resource == "" && len(audiences) == 0 {
		return resourceMeta{}, false
	}
	// Key by resource when present; otherwise by the first audience so distinct
	// auth blocks without a resource identifier still dedup sensibly.
	key := resource
	if key == "" {
		key = audiences[0]
	}
	return resourceMeta{
		resource:  key,
		issuers:   issuers,
		audiences: uniqueStrings(audiences),
		scopes:    scopes,
		jwksURI:   strings.TrimSpace(jwks),
	}, true
}

// mergeIssuers collects trusted issuers from the array form
// (`authorization_servers`) and the singular forms (`issuer`,
// `authorization_server`).
func mergeIssuers(m map[string]any) []string {
	issuers := strSlice(m, "authorization_servers", "authorization_server_urls", "issuers")
	for _, k := range []string{"issuer", "authorization_server"} {
		if s, ok := m[k].(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				issuers = appendUnique(issuers, s)
			}
		}
	}
	return uniqueStrings(issuers)
}

func (d *Discoverer) toConsumer(m resourceMeta, path string) (model.Consumer, bool) {
	if len(m.issuers) == 0 {
		return model.Consumer{}, false
	}
	host := hostOf(m.resource)
	stableID := discovery.StableID(discovery.IDParts{Host: host})

	name := host
	if name == "" {
		name = m.resource
	}

	source := "mcp:protected-resource/" + m.resource
	prov := model.ProvenanceRecord{Source: source, Locator: path, ObservedAt: d.now(), Confidence: 0.95}

	jwks := model.JWKSBehavior{Source: model.SrcConfig}
	provMap := map[string][]model.ProvenanceRecord{
		"expects.issuers":   {prov},
		"expects.audiences": {prov},
	}
	if m.jwksURI != "" {
		jwks.JWKSURI = m.jwksURI
		provMap["jwks_behavior.jwks_uri"] = []model.ProvenanceRecord{prov}
	}

	// Probeable only when the resource is an addressable http(s) endpoint.
	probeable := strings.HasPrefix(m.resource, "https://") || strings.HasPrefix(m.resource, "http://")
	var endpoints []model.Endpoint
	if probeable {
		endpoints = []model.Endpoint{{URL: strings.TrimRight(m.resource, "/"), Method: "GET"}}
	}

	return model.Consumer{
		ID:        stableID,
		StableID:  stableID,
		Kind:      model.ConsumerService,
		Name:      name,
		Endpoints: endpoints,
		Expects: model.Expectations{
			Issuers:   sortStrings(m.issuers),
			Audiences: sortStrings(m.audiences),
		},
		JWKSBehavior: jwks,
		Provenance:   provMap,
		Confidence: map[string]float64{
			"overall":           0.95,
			"expects.issuers":   0.95,
			"expects.audiences": 0.95,
		},
		Probeable: probeable,
	}, true
}

// --- helpers -----------------------------------------------------------------

// hostOf returns the lowercase host of a resource URL, or the trimmed value when
// it is not a URL (e.g. a bare resource name used as an audience).
func hostOf(resource string) string {
	r := strings.TrimSpace(resource)
	r = strings.TrimPrefix(r, "https://")
	r = strings.TrimPrefix(r, "http://")
	if i := strings.IndexAny(r, "/?#"); i >= 0 {
		r = r[:i]
	}
	return strings.ToLower(r)
}

// firstString returns the first key present as a non-empty trimmed string.
func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return ""
}

// strSlice reads the given keys as string slices (or a single string) and unions
// their non-empty, trimmed values.
func strSlice(m map[string]any, keys ...string) []string {
	var out []string
	for _, k := range keys {
		switch v := m[k].(type) {
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

func uniqueStrings(ss []string) []string {
	var out []string
	for _, s := range ss {
		out = appendUnique(out, s)
	}
	return out
}

func sortStrings(ss []string) []string {
	out := append([]string{}, ss...)
	sort.Strings(out)
	return out
}
