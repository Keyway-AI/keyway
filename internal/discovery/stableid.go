package discovery

import "strings"

// IDParts carries the identifiers used to derive a consumer StableID.
type IDParts struct {
	Cluster        string
	Namespace      string
	ServiceAccount string
	ServiceName    string
	Gateway        string
	Route          string
	Host           string
	PathPrefix     string
}

// StableID derives a consumer's stable identity (PRD §4.2), first match wins:
//
//  1. k8s://{cluster}/{namespace}/{service-account-name}
//  2. k8s://{cluster}/{namespace}/{service-name}
//  3. route://{gateway-name}/{route-name}
//  4. url://{normalized-host}{path-prefix}
//
// The result must survive redeploys and IP changes.
func StableID(p IDParts) string {
	cluster := p.Cluster
	if cluster == "" {
		cluster = "local"
	}
	ns := p.Namespace
	if ns == "" {
		ns = "default"
	}
	switch {
	case p.ServiceAccount != "":
		return "k8s://" + cluster + "/" + ns + "/" + p.ServiceAccount
	case p.ServiceName != "":
		return "k8s://" + cluster + "/" + ns + "/" + p.ServiceName
	case p.Gateway != "" || p.Route != "":
		return "route://" + p.Gateway + "/" + p.Route
	default:
		return "url://" + normalizeHost(p.Host) + p.PathPrefix
	}
}

// normalizeHost lowercases and strips a scheme and trailing slash from a host.
func normalizeHost(host string) string {
	h := strings.ToLower(host)
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	return strings.TrimRight(h, "/")
}
