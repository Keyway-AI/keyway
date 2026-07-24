// Package istio discovers consumers from Istio RequestAuthentication resources
// (confidence 1.0 — declarative and unambiguous). It reads manifests from the
// configured paths; the in-cluster dynamic-client path is a future addition.
package istio

import (
	"context"
	"strings"
	"time"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/model"
	"gopkg.in/yaml.v3"
)

// Discoverer reads Istio security CRDs.
type Discoverer struct {
	now func() time.Time
}

// New constructs an Istio discoverer.
func New() *Discoverer { return &Discoverer{now: time.Now} }

// WithClock injects a clock (tests).
func (d *Discoverer) WithClock(now func() time.Time) *Discoverer { d.now = now; return d }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "istio" }

// requestAuthentication is the subset of the CRD we read.
type requestAuthentication struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string            `yaml:"name"`
		Namespace string            `yaml:"namespace"`
		Labels    map[string]string `yaml:"labels"`
	} `yaml:"metadata"`
	Spec struct {
		Selector struct {
			MatchLabels map[string]string `yaml:"matchLabels"`
		} `yaml:"selector"`
		JWTRules []struct {
			Issuer               string   `yaml:"issuer"`
			Audiences            []string `yaml:"audiences"`
			JWKSURI              string   `yaml:"jwksUri"`
			ForwardOriginalToken bool     `yaml:"forwardOriginalToken"`
			FromHeaders          []struct {
				Name   string `yaml:"name"`
				Prefix string `yaml:"prefix"`
			} `yaml:"fromHeaders"`
		} `yaml:"jwtRules"`
	} `yaml:"spec"`
}

// authorizationPolicy is the subset of the Istio AuthorizationPolicy CRD we read
// to derive required claims from `when` conditions on request.auth.claims[...].
type authorizationPolicy struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Selector struct {
			MatchLabels map[string]string `yaml:"matchLabels"`
		} `yaml:"selector"`
		Rules []struct {
			When []struct {
				Key    string   `yaml:"key"`
				Values []string `yaml:"values"`
			} `yaml:"when"`
		} `yaml:"rules"`
	} `yaml:"spec"`
}

// Discover parses RequestAuthentication resources under scope.ConfigPaths and
// enriches them with required claims from AuthorizationPolicy resources that
// select the same workload.
func (d *Discoverer) Discover(_ context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	if len(scope.ConfigPaths) == 0 {
		return nil, nil
	}
	var consumers []model.Consumer
	claimsByWorkload := map[string][]string{} // "ns/service" -> required claims
	claimSource := map[string]string{}        // "ns/service" -> AuthorizationPolicy locator

	err := discovery.WalkYAML(scope.ConfigPaths, func(path string, doc []byte) error {
		var probe struct {
			Kind string `yaml:"kind"`
		}
		if yaml.Unmarshal(doc, &probe) != nil {
			return nil // tolerate non-Istio documents
		}
		switch probe.Kind {
		case "RequestAuthentication":
			var ra requestAuthentication
			if yaml.Unmarshal(doc, &ra) != nil || len(ra.Spec.JWTRules) == 0 {
				return nil
			}
			if len(scope.Namespaces) > 0 && !contains(scope.Namespaces, ra.Metadata.Namespace) {
				return nil
			}
			consumers = append(consumers, d.toConsumer(ra, path, scope))
		case "AuthorizationPolicy":
			var ap authorizationPolicy
			if yaml.Unmarshal(doc, &ap) != nil {
				return nil
			}
			if len(scope.Namespaces) > 0 && !contains(scope.Namespaces, ap.Metadata.Namespace) {
				return nil
			}
			if claims := requiredClaims(ap); len(claims) > 0 {
				key := workloadKey(ap.Metadata.Namespace, apServiceName(ap))
				claimsByWorkload[key] = unionAll(claimsByWorkload[key], claims)
				claimSource[key] = path
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Merge required claims into the matching consumers.
	for i := range consumers {
		key := workloadKey(consumers[i].Namespace, consumers[i].Name)
		claims := claimsByWorkload[key]
		if len(claims) == 0 {
			continue
		}
		consumers[i].Expects.RequiredClaims = unionAll(consumers[i].Expects.RequiredClaims, claims)
		consumers[i].Confidence["expects.required_claims"] = 1.0
		prov := model.ProvenanceRecord{
			Source: "istio:AuthorizationPolicy", Locator: claimSource[key], ObservedAt: d.now(), Confidence: 1.0,
		}
		consumers[i].Provenance["expects.required_claims"] = []model.ProvenanceRecord{prov}
	}
	return consumers, nil
}

// requiredClaims extracts claim names required by an AuthorizationPolicy's
// when-conditions of the form request.auth.claims[<name>].
func requiredClaims(ap authorizationPolicy) []string {
	var out []string
	for _, rule := range ap.Spec.Rules {
		for _, w := range rule.When {
			if name, ok := claimKey(w.Key); ok {
				out = appendUnique(out, name)
			}
		}
	}
	return out
}

// claimKey returns the claim name from "request.auth.claims[<name>]".
func claimKey(key string) (string, bool) {
	const prefix = "request.auth.claims["
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, "]") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(key, prefix), "]")
	if name == "" {
		return "", false
	}
	return name, true
}

func apServiceName(ap authorizationPolicy) string {
	for _, k := range []string{"app", "app.kubernetes.io/name"} {
		if v := ap.Spec.Selector.MatchLabels[k]; v != "" {
			return v
		}
	}
	return ap.Metadata.Name
}

func workloadKey(ns, service string) string {
	if ns == "" {
		ns = "default"
	}
	return ns + "/" + service
}

func unionAll(a, b []string) []string {
	out := append([]string{}, a...)
	for _, v := range b {
		out = appendUnique(out, v)
	}
	return out
}

func (d *Discoverer) toConsumer(ra requestAuthentication, path string, scope discovery.Scope) model.Consumer {
	name := serviceName(ra)
	ns := ra.Metadata.Namespace
	if ns == "" {
		ns = "default"
	}
	stableID := discovery.StableID(discovery.IDParts{
		Cluster:     scope.KubeContext,
		Namespace:   ns,
		ServiceName: name,
	})

	var issuers, audiences []string
	for _, r := range ra.Spec.JWTRules {
		if r.Issuer != "" {
			issuers = appendUnique(issuers, r.Issuer)
		}
		for _, a := range r.Audiences {
			audiences = appendUnique(audiences, a)
		}
	}

	source := "istio:RequestAuthentication/" + ra.Metadata.Name
	prov := model.ProvenanceRecord{Source: source, Locator: path, ObservedAt: d.now(), Confidence: 1.0}
	provMap := map[string][]model.ProvenanceRecord{
		"expects.issuers":   {prov},
		"expects.audiences": {prov},
	}
	conf := map[string]float64{
		"overall":           1.0,
		"expects.issuers":   1.0,
		"expects.audiences": 1.0,
	}

	return model.Consumer{
		ID:        stableID,
		StableID:  stableID,
		Kind:      model.ConsumerService,
		Name:      name,
		Namespace: ns,
		OwnerTeam: ownerTeam(ra.Metadata.Labels),
		Expects: model.Expectations{
			Issuers:      issuers,
			Audiences:    audiences,
			ClockSkewSec: 0,
		},
		JWKSBehavior: model.JWKSBehavior{Source: model.SrcConfig},
		Provenance:   provMap,
		Confidence:   conf,
		Probeable:    true,
	}
}

func serviceName(ra requestAuthentication) string {
	for _, key := range []string{"app", "app.kubernetes.io/name"} {
		if v := ra.Spec.Selector.MatchLabels[key]; v != "" {
			return v
		}
	}
	return ra.Metadata.Name
}

func ownerTeam(labels map[string]string) string {
	for _, key := range []string{"team", "owner", "app.kubernetes.io/part-of"} {
		if v := labels[key]; v != "" {
			return v
		}
	}
	return ""
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func appendUnique(ss []string, v string) []string {
	if contains(ss, v) {
		return ss
	}
	return append(ss, v)
}
