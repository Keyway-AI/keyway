// Package contract assembles the derived contract graph, hashes it canonically,
// and manages versioning / the baseline flow.
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/nometria/keyway/internal/model"
)

// Hash returns the canonical SHA-256 (hex) of a contract version, per PRD §8.1.
//
// Determinism guarantees (AC-1: two runs against an unchanged system must
// produce an identical hash):
//
//   - All volatile/observation fields are excluded from the canonical form:
//     every timestamp (CreatedAt, FirstSeenInJWKS, LastVerified,
//     LastObservedRefresh, ...), the volatile UUID IDs, all ProbeResults, and
//     provenance/confidence metadata (evidence about the contract, not the
//     contract itself).
//   - Edges are keyed by *stable* identifiers (issuer URL + consumer StableID),
//     never by the volatile UUID IDs they carry at runtime.
//   - Every slice is sorted deterministically and every inner string slice is
//     sorted lexicographically before marshaling.
//
// The canonical form contains no Go maps, so field/element ordering is fully
// determined by struct declaration order and the sorts below.
func Hash(v model.ContractVersion) string {
	c := canonicalize(v)
	b, err := json.Marshal(c)
	if err != nil {
		// The canonical form contains only JSON-safe scalars/slices; marshaling
		// cannot fail. Panic would only fire on a programming error here.
		panic("contract: canonical marshal failed: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// --- canonical, hash-stable projections -------------------------------------

type canonContract struct {
	Issuers   []canonIssuer   `json:"issuers"`
	Consumers []canonConsumer `json:"consumers"`
	Edges     []canonEdge     `json:"edges"`
}

type canonIssuer struct {
	Name               string       `json:"name"`
	Type               string       `json:"type"`
	IssuerURL          string       `json:"issuer_url"`
	JWKSURI            string       `json:"jwks_uri"`
	ControlsPrivateKey bool         `json:"controls_private_key"`
	Keys               []canonKey   `json:"keys"`
	Claims             []canonClaim `json:"claims"`
}

type canonKey struct {
	KID          string `json:"kid"`
	Alg          string `json:"alg"`
	Use          string `json:"use"`
	PublicKeyPEM string `json:"public_key_pem"`
	Status       string `json:"status"`
}

type canonClaim struct {
	Name         string  `json:"name"`
	PresenceRate float64 `json:"presence_rate"`
}

type canonConsumer struct {
	StableID  string          `json:"stable_id"`
	Kind      string          `json:"kind"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	OwnerTeam string          `json:"owner_team"`
	Endpoints []canonEndpoint `json:"endpoints"`
	Expects   canonExpect     `json:"expects"`
	JWKS      canonJWKS       `json:"jwks_behavior"`
	Library   *canonLib       `json:"library,omitempty"`
	Probeable bool            `json:"probeable"`
}

type canonEndpoint struct {
	URL           string `json:"url"`
	Method        string `json:"method"`
	SafeProbePath string `json:"safe_probe_path"`
}

type canonExpect struct {
	Issuers        []string `json:"issuers"`
	Audiences      []string `json:"audiences"`
	Algorithms     []string `json:"algorithms"`
	RequiredClaims []string `json:"required_claims"`
	ClockSkewSec   int      `json:"clock_skew_sec"`
}

type canonJWKS struct {
	CacheTTLSec           *int   `json:"cache_ttl_sec"`
	RefreshIntervalSec    *int   `json:"refresh_interval_sec"`
	RefreshesOnUnknownKID *bool  `json:"refreshes_on_unknown_kid"`
	Source                string `json:"source"`
}

type canonLib struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Lang    string `json:"lang"`
}

type canonEdge struct {
	IssuerURL        string      `json:"issuer_url"`
	ConsumerStableID string      `json:"consumer_stable_id"`
	Expects          canonExpect `json:"expects"`
}

func canonicalize(v model.ContractVersion) canonContract {
	// Resolve volatile IDs to stable keys for edge canonicalisation.
	issuerURLByID := make(map[string]string, len(v.Issuers))
	for _, is := range v.Issuers {
		issuerURLByID[is.ID] = is.IssuerURL
	}
	stableIDByID := make(map[string]string, len(v.Consumers))
	for _, c := range v.Consumers {
		stableIDByID[c.ID] = c.StableID
	}

	out := canonContract{
		Issuers:   make([]canonIssuer, 0, len(v.Issuers)),
		Consumers: make([]canonConsumer, 0, len(v.Consumers)),
		Edges:     make([]canonEdge, 0, len(v.Edges)),
	}

	for _, is := range v.Issuers {
		ci := canonIssuer{
			Name:               is.Name,
			Type:               string(is.Type),
			IssuerURL:          is.IssuerURL,
			JWKSURI:            is.JWKSURI,
			ControlsPrivateKey: is.ControlsPrivateKey,
		}
		for _, k := range is.Keys {
			ci.Keys = append(ci.Keys, canonKey{
				KID:          k.KID,
				Alg:          k.Alg,
				Use:          k.Use,
				PublicKeyPEM: k.PublicKeyPEM,
				Status:       string(k.Status),
			})
		}
		// Total comparators (tie-break beyond the primary key) so two keys sharing
		// a KID — or two claims sharing a Name — still sort to a deterministic
		// order regardless of input order, preserving the AC-1 hash guarantee.
		sort.Slice(ci.Keys, func(i, j int) bool {
			if ci.Keys[i].KID != ci.Keys[j].KID {
				return ci.Keys[i].KID < ci.Keys[j].KID
			}
			if ci.Keys[i].Alg != ci.Keys[j].Alg {
				return ci.Keys[i].Alg < ci.Keys[j].Alg
			}
			return ci.Keys[i].Status < ci.Keys[j].Status
		})
		for _, cl := range is.ClaimSchema {
			ci.Claims = append(ci.Claims, canonClaim{Name: cl.Name, PresenceRate: cl.PresenceRate})
		}
		sort.Slice(ci.Claims, func(i, j int) bool {
			if ci.Claims[i].Name != ci.Claims[j].Name {
				return ci.Claims[i].Name < ci.Claims[j].Name
			}
			return ci.Claims[i].PresenceRate < ci.Claims[j].PresenceRate
		})
		out.Issuers = append(out.Issuers, ci)
	}
	sort.Slice(out.Issuers, func(i, j int) bool { return out.Issuers[i].IssuerURL < out.Issuers[j].IssuerURL })

	for _, c := range v.Consumers {
		cc := canonConsumer{
			StableID:  c.StableID,
			Kind:      string(c.Kind),
			Name:      c.Name,
			Namespace: c.Namespace,
			OwnerTeam: c.OwnerTeam,
			Expects:   canonExpectOf(c.Expects),
			JWKS: canonJWKS{
				CacheTTLSec:           c.JWKSBehavior.CacheTTLSec,
				RefreshIntervalSec:    c.JWKSBehavior.RefreshIntervalSec,
				RefreshesOnUnknownKID: c.JWKSBehavior.RefreshesOnUnknownKID,
				Source:                string(c.JWKSBehavior.Source),
			},
			Probeable: c.Probeable,
		}
		for _, e := range c.Endpoints {
			cc.Endpoints = append(cc.Endpoints, canonEndpoint{URL: e.URL, Method: e.Method, SafeProbePath: e.SafeProbePath})
		}
		sort.Slice(cc.Endpoints, func(i, j int) bool {
			a, b := cc.Endpoints[i], cc.Endpoints[j]
			if a.URL != b.URL {
				return a.URL < b.URL
			}
			if a.Method != b.Method {
				return a.Method < b.Method
			}
			return a.SafeProbePath < b.SafeProbePath
		})
		if c.Library != nil {
			cc.Library = &canonLib{Name: c.Library.Name, Version: c.Library.Version, Lang: c.Library.Lang}
		}
		out.Consumers = append(out.Consumers, cc)
	}
	sort.Slice(out.Consumers, func(i, j int) bool { return out.Consumers[i].StableID < out.Consumers[j].StableID })

	for _, e := range v.Edges {
		out.Edges = append(out.Edges, canonEdge{
			IssuerURL:        issuerURLByID[e.IssuerID],
			ConsumerStableID: stableIDByID[e.ConsumerID],
			Expects:          canonExpectOf(e.Expects),
		})
	}
	sort.Slice(out.Edges, func(i, j int) bool {
		a, b := out.Edges[i], out.Edges[j]
		if a.IssuerURL != b.IssuerURL {
			return a.IssuerURL < b.IssuerURL
		}
		return a.ConsumerStableID < b.ConsumerStableID
	})

	return out
}

func canonExpectOf(e model.Expectations) canonExpect {
	return canonExpect{
		Issuers:        sortedCopy(e.Issuers),
		Audiences:      sortedCopy(e.Audiences),
		Algorithms:     sortedCopy(e.Algorithms),
		RequiredClaims: sortedCopy(e.RequiredClaims),
		ClockSkewSec:   e.ClockSkewSec,
	}
}

// sortedCopy returns a lexicographically sorted copy. make() always yields a
// non-nil slice, so nil and empty inputs both marshal to [] — the canonical
// form is stable regardless of the nil-vs-empty distinction.
func sortedCopy(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}
