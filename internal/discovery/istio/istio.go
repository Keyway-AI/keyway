// Package istio discovers consumers from Istio RequestAuthentication resources
// (confidence 1.0 — declarative and unambiguous). It reads from static manifests
// (Discoverer) or, via a client-go dynamic client, live cluster CRDs
// (InClusterDiscoverer, see incluster.go). Both share the same mapping.
package istio

import (
	"context"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/model"
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

// stringSlice tolerates a field written as either a YAML scalar or a sequence.
// Istio's `audiences` is schema-typed as a list, but real-world manifests in the
// wild sometimes write a bare string (`audiences: "api"`). A plain []string field
// fails to unmarshal that, which would drop the ENTIRE RequestAuthentication —
// losing the issuer too. Accepting both keeps discovery robust on messy input.
type stringSlice []string

func (s *stringSlice) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		if value.Value == "" {
			*s = nil
			return nil
		}
		*s = []string{value.Value}
		return nil
	}
	var arr []string
	if err := value.Decode(&arr); err != nil {
		return err
	}
	*s = arr
	return nil
}

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
			Issuer               string      `yaml:"issuer"`
			Audiences            stringSlice `yaml:"audiences"`
			JWKSURI              string      `yaml:"jwksUri"`
			ForwardOriginalToken bool        `yaml:"forwardOriginalToken"`
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
	var ras []raWithLoc
	var aps []apWithLoc
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
			ras = append(ras, raWithLoc{ra, path})
		case "AuthorizationPolicy":
			var ap authorizationPolicy
			if yaml.Unmarshal(doc, &ap) != nil {
				return nil
			}
			aps = append(aps, apWithLoc{ap, path})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d.assemble(scope, ras, aps), nil
}

// raWithLoc / apWithLoc pair a parsed CRD with its provenance locator (a file
// path for the manifest source, a cluster ref for the in-cluster source).
type raWithLoc struct {
	ra  requestAuthentication
	loc string
}
type apWithLoc struct {
	ap  authorizationPolicy
	loc string
}

// assemble is the shared mapping used by both the manifest and in-cluster
// sources: it turns parsed RequestAuthentications into consumers and merges
// required claims from AuthorizationPolicies onto the matching workload.
func (d *Discoverer) assemble(scope discovery.Scope, ras []raWithLoc, aps []apWithLoc) []model.Consumer {
	var consumers []model.Consumer
	claimsByWorkload := map[string][]string{} // "ns/service" -> required claims
	claimSource := map[string]string{}        // "ns/service" -> AuthorizationPolicy locator
	audByWorkload := map[string][]string{}    // "ns/service" -> audiences bound via AP when-conditions
	audSource := map[string]string{}          // "ns/service" -> AuthorizationPolicy locator

	for _, r := range ras {
		if len(r.ra.Spec.JWTRules) == 0 {
			continue
		}
		if len(scope.Namespaces) > 0 && !contains(scope.Namespaces, nsOrDefault(r.ra.Metadata.Namespace)) {
			continue
		}
		consumers = append(consumers, d.toConsumer(r.ra, r.loc, scope))
	}
	// Selector-less AuthorizationPolicies apply to every workload in their
	// namespace (Istio semantics), so their claims are tracked per-namespace.
	// nsAllClaims additionally unions the required claims of EVERY policy in a
	// namespace (any selector), used by the single-validator rule in the merge.
	nsWideClaims := map[string][]string{} // namespace -> claims (selector-less APs)
	nsWideSource := map[string]string{}
	nsAllClaims := map[string][]string{} // namespace -> claims (any AP)
	nsAllSource := map[string]string{}
	nsWideAud := map[string][]string{}   // namespace -> audiences (selector-less APs)
	nsWideAudSrc := map[string]string{}
	nsAllAud := map[string][]string{} // namespace -> audiences (any AP)
	nsAllAudSrc := map[string]string{}
	for _, a := range aps {
		ns := nsOrDefault(a.ap.Metadata.Namespace)
		if len(scope.Namespaces) > 0 && !contains(scope.Namespaces, ns) {
			continue
		}
		claims := requiredClaims(a.ap)
		auds := requiredAudiences(a.ap)
		if len(claims) == 0 && len(auds) == 0 {
			continue
		}
		selectorLess := len(a.ap.Spec.Selector.MatchLabels) == 0
		key := workloadKey(ns, apServiceName(a.ap))
		if len(claims) > 0 {
			nsAllClaims[ns] = unionAll(nsAllClaims[ns], claims)
			if nsAllSource[ns] == "" {
				nsAllSource[ns] = a.loc
			}
			if selectorLess {
				nsWideClaims[ns] = unionAll(nsWideClaims[ns], claims)
				nsWideSource[ns] = a.loc
			} else {
				claimsByWorkload[key] = unionAll(claimsByWorkload[key], claims)
				claimSource[key] = a.loc
			}
		}
		if len(auds) > 0 {
			nsAllAud[ns] = unionAll(nsAllAud[ns], auds)
			if nsAllAudSrc[ns] == "" {
				nsAllAudSrc[ns] = a.loc
			}
			if selectorLess {
				nsWideAud[ns] = unionAll(nsWideAud[ns], auds)
				nsWideAudSrc[ns] = a.loc
			} else {
				audByWorkload[key] = unionAll(audByWorkload[key], auds)
				audSource[key] = a.loc
			}
		}
	}

	// A namespace with exactly one JWT consumer has a single validator covering its
	// inbound traffic, so every required-claim policy in that namespace (whatever
	// workload it selects) constrains the tokens that consumer validates. This is
	// the common gateway pattern: one RequestAuthentication at the edge, with claim
	// policies on individual services that have no RA of their own.
	consumersPerNs := map[string]int{}
	for i := range consumers {
		consumersPerNs[consumers[i].Namespace]++
	}

	// Merge required claims into the matching consumers: those from a policy that
	// selects this workload, any namespace-wide policy, and — for a single-validator
	// namespace — every required-claim policy in the namespace.
	for i := range consumers {
		ns := consumers[i].Namespace
		key := workloadKey(ns, consumers[i].Name)

		// An audience can be bound in an AuthorizationPolicy `when` condition on
		// request.auth.audiences rather than in the RequestAuthentication jwtRule
		// (the same edge-RA / per-service-policy gateway pattern as claims). Without
		// this the consumer looks like it has no audience and is falsely counted as
		// unbound (P1). Attribute audiences with the same selector / namespace-wide /
		// single-validator rules used for claims.
		auds := unionAll(audByWorkload[key], nsWideAud[ns])
		if consumersPerNs[ns] == 1 {
			auds = unionAll(auds, nsAllAud[ns])
		}
		if len(auds) > 0 {
			before := len(consumers[i].Expects.Audiences)
			consumers[i].Expects.Audiences = unionAll(consumers[i].Expects.Audiences, auds)
			if len(consumers[i].Expects.Audiences) > before {
				aloc := audSource[key]
				if aloc == "" {
					aloc = nsWideAudSrc[ns]
				}
				if aloc == "" {
					aloc = nsAllAudSrc[ns]
				}
				consumers[i].Confidence["expects.audiences"] = 1.0
				consumers[i].Provenance["expects.audiences"] = append(
					consumers[i].Provenance["expects.audiences"],
					model.ProvenanceRecord{Source: "istio:AuthorizationPolicy", Locator: aloc, ObservedAt: d.now(), Confidence: 1.0},
				)
			}
		}

		claims := unionAll(claimsByWorkload[key], nsWideClaims[ns])
		if consumersPerNs[ns] == 1 {
			claims = unionAll(claims, nsAllClaims[ns])
		}
		if len(claims) == 0 {
			continue
		}
		loc := claimSource[key]
		if loc == "" {
			loc = nsWideSource[ns]
		}
		if loc == "" {
			loc = nsAllSource[ns]
		}
		consumers[i].Expects.RequiredClaims = unionAll(consumers[i].Expects.RequiredClaims, claims)
		consumers[i].Confidence["expects.required_claims"] = 1.0
		consumers[i].Provenance["expects.required_claims"] = []model.ProvenanceRecord{{
			Source: "istio:AuthorizationPolicy", Locator: loc, ObservedAt: d.now(), Confidence: 1.0,
		}}
	}
	return consumers
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

// requiredAudiences extracts the audiences an AuthorizationPolicy binds through
// a when-condition on request.auth.audiences. Istio matches the condition when a
// token's aud claim is in the listed values, so those values are accepted
// audiences for the workload the policy selects.
func requiredAudiences(ap authorizationPolicy) []string {
	var out []string
	for _, rule := range ap.Spec.Rules {
		for _, w := range rule.When {
			if strings.TrimSpace(w.Key) != "request.auth.audiences" {
				continue
			}
			for _, v := range w.Values {
				if v := strings.TrimSpace(v); v != "" {
					out = appendUnique(out, v)
				}
			}
		}
	}
	return out
}

// claimKey returns the top-level claim name from "request.auth.claims[<name>]".
// For a nested claim like "request.auth.claims[realm_access][roles]" it returns
// the first segment ("realm_access") — the claim the token must carry.
func claimKey(key string) (string, bool) {
	const prefix = "request.auth.claims["
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(key, prefix)
	idx := strings.IndexByte(rest, ']')
	if idx <= 0 {
		return "", false
	}
	return rest[:idx], true
}

// workloadLabelKeys are the selector labels, in preference order, that name the
// workload a policy targets. Beyond the app.kubernetes.io convention this
// recognizes Istio gateway (`istio`) and legacy (`k8s-app`) selectors, so a
// gateway RequestAuthentication is named for its workload rather than the policy
// (KI-33). Falls back to the resource name when no selector matches.
var workloadLabelKeys = []string{"app", "app.kubernetes.io/name", "istio", "k8s-app"}

func labelValue(labels map[string]string) (string, bool) {
	for _, k := range workloadLabelKeys {
		if v := labels[k]; v != "" {
			return v, true
		}
	}
	return "", false
}

func apServiceName(ap authorizationPolicy) string {
	if v, ok := labelValue(ap.Spec.Selector.MatchLabels); ok {
		return v
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
	ns := nsOrDefault(ra.Metadata.Namespace)
	stableID := discovery.StableID(discovery.IDParts{
		Cluster:     scope.KubeContext,
		Namespace:   ns,
		ServiceName: name,
	})

	var issuers, audiences []string
	var jwksURI string
	for _, r := range ra.Spec.JWTRules {
		// Trim surrounding whitespace: it is never part of a real issuer/audience
		// (a token's iss/aud claim can't carry it) and only causes false-positive
		// diffs. Trailing slashes are deliberately preserved — under strict OIDC
		// they change which tokens validate, so they are a real contract change
		// (KI-26).
		if iss := strings.TrimSpace(r.Issuer); iss != "" {
			issuers = appendUnique(issuers, iss)
		}
		for _, a := range r.Audiences {
			if a := strings.TrimSpace(a); a != "" {
				audiences = appendUnique(audiences, a)
			}
		}
		if jwksURI == "" && r.JWKSURI != "" {
			jwksURI = strings.TrimSpace(r.JWKSURI) // the rotation endpoint (KI-32)
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
		JWKSBehavior: model.JWKSBehavior{JWKSURI: jwksURI, Source: model.SrcConfig},
		Provenance:   provMap,
		Confidence:   conf,
		Probeable:    true,
	}
}

func serviceName(ra requestAuthentication) string {
	if v, ok := labelValue(ra.Spec.Selector.MatchLabels); ok {
		return v
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

// nsOrDefault normalizes an empty Kubernetes namespace to "default", matching how
// the API server treats an unqualified resource — so namespace scoping and the
// consumer's identity agree.
func nsOrDefault(ns string) string {
	if ns == "" {
		return "default"
	}
	return ns
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
