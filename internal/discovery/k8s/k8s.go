// Package k8s discovers consumers from Kubernetes Services, their backing
// workloads, and projected service-account token volumes (PRD §7.3). It reads
// manifests from the configured paths; the in-cluster client-go path is future.
package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nometria/keyway/internal/discovery"
	"github.com/nometria/keyway/internal/model"
)

// ownerLabelPriority is the label lookup order for OwnerTeam (PRD §7.3).
var ownerLabelPriority = []string{"team", "owner", "app.kubernetes.io/part-of"}

// Discoverer enumerates workloads and their Services.
type Discoverer struct {
	now func() time.Time
}

// New constructs a Kubernetes discoverer.
func New() *Discoverer { return &Discoverer{now: time.Now} }

// WithClock injects a clock (tests).
func (d *Discoverer) WithClock(now func() time.Time) *Discoverer { d.now = now; return d }

var _ discovery.Discoverer = (*Discoverer)(nil)

// Name identifies this source in provenance records.
func (d *Discoverer) Name() string { return "k8s" }

type objectMeta struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

type service struct {
	Kind     string     `yaml:"kind"`
	Metadata objectMeta `yaml:"metadata"`
	Spec     struct {
		Selector map[string]string `yaml:"selector"`
		Ports    []struct {
			Port int    `yaml:"port"`
			Name string `yaml:"name"`
		} `yaml:"ports"`
	} `yaml:"spec"`
}

type workload struct {
	Kind     string     `yaml:"kind"`
	Metadata objectMeta `yaml:"metadata"`
	Spec     struct {
		Template struct {
			Metadata objectMeta `yaml:"metadata"`
			Spec     struct {
				ServiceAccountName string `yaml:"serviceAccountName"`
				Containers         []struct {
					Env []struct {
						Name  string `yaml:"name"`
						Value string `yaml:"value"`
					} `yaml:"env"`
					Ports []struct {
						ContainerPort int `yaml:"containerPort"`
					} `yaml:"ports"`
				} `yaml:"containers"`
				Volumes []struct {
					Projected struct {
						Sources []struct {
							ServiceAccountToken struct {
								Audience string `yaml:"audience"`
							} `yaml:"serviceAccountToken"`
						} `yaml:"sources"`
					} `yaml:"projected"`
				} `yaml:"volumes"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

// Discover parses Services and workloads under scope.ConfigPaths.
func (d *Discoverer) Discover(_ context.Context, scope discovery.Scope) ([]model.Consumer, error) {
	if len(scope.ConfigPaths) == 0 {
		return nil, nil
	}
	var services []service
	var workloads []workload

	err := discovery.WalkYAML(scope.ConfigPaths, func(_ string, doc []byte) error {
		var probe struct {
			Kind string `yaml:"kind"`
		}
		if err := yaml.Unmarshal(doc, &probe); err != nil {
			return nil
		}
		switch probe.Kind {
		case "Service":
			var s service
			if yaml.Unmarshal(doc, &s) == nil {
				services = append(services, s)
			}
		case "Deployment", "StatefulSet", "DaemonSet":
			var w workload
			if yaml.Unmarshal(doc, &w) == nil {
				workloads = append(workloads, w)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var consumers []model.Consumer
	for _, w := range workloads {
		// Match against the normalized namespace: a manifest that omits
		// metadata.namespace lands in "default", so `--namespace default` must
		// still select it.
		if len(scope.Namespaces) > 0 && !contains(scope.Namespaces, firstNonEmpty(w.Metadata.Namespace, "default")) {
			continue
		}
		consumers = append(consumers, d.toConsumer(w, services, scope))
	}
	return consumers, nil
}

func (d *Discoverer) toConsumer(w workload, services []service, scope discovery.Scope) model.Consumer {
	ns := firstNonEmpty(w.Metadata.Namespace, "default")
	podLabels := merge(w.Metadata.Labels, w.Spec.Template.Metadata.Labels)
	sa := w.Spec.Template.Spec.ServiceAccountName

	issuers, audiences, envConf := envHints(w)
	saAudience := projectedAudience(w)
	if saAudience != "" {
		audiences = appendUnique(audiences, saAudience)
	}

	svc := matchService(services, ns, podLabels)
	serviceName := w.Metadata.Name
	var endpoints []model.Endpoint
	if svc != nil {
		serviceName = svc.Metadata.Name
		endpoints = serviceEndpoints(*svc, ns)
	}

	stableID := discovery.StableID(discovery.IDParts{
		Cluster:        scope.KubeContext,
		Namespace:      ns,
		ServiceAccount: saOrEmpty(sa, saAudience),
		ServiceName:    serviceName,
	})

	// When the primary identity is service-account-based (a projected token ties
	// the workload to an SA-issued audience), also expose the service-name
	// identity as an alias, so an Istio consumer keyed by service name merges
	// with this one instead of producing a duplicate (KI-28).
	var aliases []string
	if saOrEmpty(sa, saAudience) != "" {
		svcID := discovery.StableID(discovery.IDParts{Cluster: scope.KubeContext, Namespace: ns, ServiceName: serviceName})
		if svcID != stableID {
			aliases = append(aliases, svcID)
		}
	}

	source := "k8s:" + w.Kind + "/" + w.Metadata.Name
	prov := model.ProvenanceRecord{Source: source, Locator: source, ObservedAt: d.now(), Confidence: confidenceFor(saAudience, envConf)}
	provMap := map[string][]model.ProvenanceRecord{"expects.audiences": {prov}}

	confOverall := envConf
	if saAudience != "" {
		confOverall = 1.0 // audience read directly from the projection
	}

	return model.Consumer{
		ID:        stableID,
		StableID:  stableID,
		Aliases:   aliases,
		Kind:      model.ConsumerService,
		Name:      serviceName,
		Namespace: ns,
		OwnerTeam: ownerTeam(podLabels),
		Endpoints: endpoints,
		Expects: model.Expectations{
			Issuers:   issuers,
			Audiences: audiences,
		},
		JWKSBehavior: model.JWKSBehavior{Source: model.SrcConfig},
		Provenance:   provMap,
		Confidence:   map[string]float64{"overall": confOverall},
		Probeable:    len(endpoints) > 0,
	}
}

// envHints scans container env for issuer/audience hints (confidence 0.5).
func envHints(w workload) (issuers, audiences []string, confidence float64) {
	for _, c := range w.Spec.Template.Spec.Containers {
		for _, e := range c.Env {
			name := strings.ToUpper(e.Name)
			if !hasAuthPrefix(name) || e.Value == "" {
				continue
			}
			switch {
			case strings.Contains(name, "ISSUER"):
				issuers = appendUnique(issuers, e.Value)
				confidence = 0.5
			case strings.Contains(name, "AUDIENCE") || strings.HasSuffix(name, "_AUD"):
				audiences = appendUnique(audiences, e.Value)
				confidence = 0.5
			}
		}
	}
	return issuers, audiences, confidence
}

func hasAuthPrefix(name string) bool {
	for _, p := range []string{"OIDC_", "JWT_", "AUTH_", "OAUTH_"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func projectedAudience(w workload) string {
	for _, v := range w.Spec.Template.Spec.Volumes {
		for _, s := range v.Projected.Sources {
			if s.ServiceAccountToken.Audience != "" {
				return s.ServiceAccountToken.Audience
			}
		}
	}
	return ""
}

func matchService(services []service, ns string, podLabels map[string]string) *service {
	for i := range services {
		s := services[i]
		if s.Metadata.Namespace != ns && (s.Metadata.Namespace != "" || ns != "default") {
			continue
		}
		if len(s.Spec.Selector) == 0 {
			continue
		}
		if selectorMatches(s.Spec.Selector, podLabels) {
			return &s
		}
	}
	return nil
}

func selectorMatches(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func serviceEndpoints(s service, ns string) []model.Endpoint {
	host := fmt.Sprintf("%s.%s.svc.cluster.local", s.Metadata.Name, ns)
	var eps []model.Endpoint
	for _, p := range s.Spec.Ports {
		eps = append(eps, model.Endpoint{
			URL:           fmt.Sprintf("http://%s:%d", host, p.Port),
			Method:        "GET",
			SafeProbePath: "/",
		})
	}
	if len(eps) == 0 {
		eps = append(eps, model.Endpoint{URL: "http://" + host, Method: "GET", SafeProbePath: "/"})
	}
	return eps
}

// --- helpers ----------------------------------------------------------------

func confidenceFor(saAudience string, envConf float64) float64 {
	if saAudience != "" {
		return 1.0
	}
	if envConf > 0 {
		return envConf
	}
	return 0.5
}

func saOrEmpty(sa, saAudience string) string {
	// Use the SA in the StableID only when a projected token ties this workload
	// to an SA-issued audience; otherwise prefer the service name.
	if saAudience != "" {
		return sa
	}
	return ""
}

func ownerTeam(labels map[string]string) string {
	for _, key := range ownerLabelPriority {
		if v := labels[key]; v != "" {
			return v
		}
	}
	return ""
}

func merge(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
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

func appendUnique(ss []string, v string) []string {
	if contains(ss, v) {
		return ss
	}
	return append(ss, v)
}
