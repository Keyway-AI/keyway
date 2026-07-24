// Package istio discovers consumers from Istio RequestAuthentication resources
// (confidence 1.0 — declarative and unambiguous). It reads manifests from the
// configured paths; the in-cluster dynamic-client path is a future addition.
package istio

import (
	"context"
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

// Discover parses RequestAuthentication resources under scope.ConfigPaths.
func (d *Discoverer) Discover(_ context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	if len(scope.ConfigPaths) == 0 {
		return nil, nil
	}
	var consumers []model.Consumer
	err := discovery.WalkYAML(scope.ConfigPaths, func(path string, doc []byte) error {
		var ra requestAuthentication
		if err := yaml.Unmarshal(doc, &ra); err != nil {
			return nil // tolerate non-Istio documents
		}
		if ra.Kind != "RequestAuthentication" || len(ra.Spec.JWTRules) == 0 {
			return nil
		}
		if len(scope.Namespaces) > 0 && !contains(scope.Namespaces, ra.Metadata.Namespace) {
			return nil
		}
		consumers = append(consumers, d.toConsumer(ra, path, scope))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return consumers, nil
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
