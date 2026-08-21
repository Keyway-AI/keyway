// Package dedup collapses duplicate and near-duplicate auth configs so the
// measurement denominator counts distinct contracts, not the same manifest pasted
// across many repos. It replaces the exact byte-signature dedup (which missed
// trivial variants) with a canonical signature, and adds a near-duplicate
// diagnostic so the study can report how much residual copying remains — the
// "fork/near-duplicate handling" gap named in FINDINGS.md.
//
// Canonicalization is deliberately conservative: it normalizes only differences
// that cannot change meaning (issuer host case, a trailing slash, algorithm case),
// so it never merges two genuinely different contracts. Fuzzy near-duplicate
// collapsing is opt-in and reported, never silently applied.
package dedup

import (
	"net/url"
	"sort"
	"strings"
)

// Config is the subset of a discovered consumer that defines its contract.
type Config struct {
	Issuers, Audiences, Algorithms, Claims []string
}

// Signature is a canonical, order-independent fingerprint. Two configs with the
// same Signature are the same contract up to meaning-preserving normalization:
// issuer scheme/host lowercased and a trailing slash dropped, algorithms
// upper-cased, list order and duplicates removed.
func Signature(c Config) string {
	return strings.Join([]string{
		"iss=" + joinNorm(c.Issuers, canonIssuer),
		"aud=" + joinNorm(c.Audiences, canonScalar),
		"alg=" + joinNorm(c.Algorithms, strings.ToUpper),
		"claim=" + joinNorm(c.Claims, strings.TrimSpace),
	}, "|")
}

// Tokens is the prefixed feature set of a config, for Jaccard similarity. Prefixes
// keep an issuer "x" distinct from an audience "x".
func Tokens(c Config) map[string]struct{} {
	set := map[string]struct{}{}
	add := func(prefix string, xs []string, fn func(string) string) {
		for _, x := range xs {
			set[prefix+fn(x)] = struct{}{}
		}
	}
	add("iss:", c.Issuers, canonIssuer)
	add("aud:", c.Audiences, canonScalar)
	add("alg:", c.Algorithms, strings.ToUpper)
	add("claim:", c.Claims, strings.TrimSpace)
	return set
}

// Jaccard is |A∩B| / |A∪B| over the two configs' feature sets: 1.0 identical,
// 0.0 disjoint. Two empty configs are treated as identical (1.0).
func Jaccard(a, b Config) float64 {
	ta, tb := Tokens(a), Tokens(b)
	if len(ta) == 0 && len(tb) == 0 {
		return 1
	}
	inter := 0
	for t := range ta {
		if _, ok := tb[t]; ok {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// ClusterIDs groups configs by single-linkage similarity: any two configs with
// Jaccard >= threshold end up in the same cluster. Returns a cluster id per input
// index. threshold <= 0 puts everything in one cluster; threshold > 1 makes every
// item its own cluster. Single-linkage can chain, so callers should treat the
// result as a diagnostic upper bound on duplication, not ground truth.
func ClusterIDs(cs []Config, threshold float64) []int {
	uf := newUnionFind(len(cs))
	for i := 0; i < len(cs); i++ {
		for j := i + 1; j < len(cs); j++ {
			if Jaccard(cs[i], cs[j]) >= threshold {
				uf.union(i, j)
			}
		}
	}
	// Relabel roots to dense 0..k-1 ids for stable, readable output.
	label := map[int]int{}
	ids := make([]int, len(cs))
	next := 0
	for i := range cs {
		r := uf.find(i)
		if _, ok := label[r]; !ok {
			label[r] = next
			next++
		}
		ids[i] = label[r]
	}
	return ids
}

// NearDupStat is the near-duplicate summary at one similarity threshold.
type NearDupStat struct {
	Threshold float64 `json:"threshold"`
	Clusters  int     `json:"clusters"`  // distinct configs after near-collapse
	Collapsed int     `json:"collapsed"` // how many configs would be removed
}

// NearDupReport computes, for each threshold, how many configs would collapse if
// near-duplicates were merged. It does not modify the input; it is a diagnostic.
func NearDupReport(cs []Config, thresholds ...float64) []NearDupStat {
	out := make([]NearDupStat, 0, len(thresholds))
	for _, t := range thresholds {
		clusters := countDistinct(ClusterIDs(cs, t))
		out = append(out, NearDupStat{
			Threshold: t, Clusters: clusters, Collapsed: len(cs) - clusters,
		})
	}
	return out
}

// --- canonicalization ---

// canonIssuer normalizes a JWT issuer: it is a URL, so scheme and host are
// case-insensitive and a trailing slash is not significant. Path and query are
// preserved (OIDC issuers can carry a realm path). Non-URL issuers fall back to a
// trimmed, trailing-slash-stripped string.
func canonIssuer(s string) string {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return strings.TrimRight(s, "/")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if p := strings.TrimRight(u.Path, "/"); p != u.Path {
		u.Path = p
	}
	return strings.TrimRight(u.String(), "/")
}

// canonScalar trims whitespace and a single trailing slash. It intentionally does
// NOT lowercase: audiences and other identifiers can be case-sensitive, and
// merging them by case would over-collapse distinct contracts.
func canonScalar(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}

func joinNorm(xs []string, fn func(string) string) string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		c := fn(x)
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func countDistinct(ids []int) int {
	seen := map[int]struct{}{}
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	return len(seen)
}

// --- union-find ---

type unionFind struct{ parent, rank []int }

func newUnionFind(n int) *unionFind {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &unionFind{parent: p, rank: make([]int, n)}
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]] // path halving
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return
	}
	switch {
	case u.rank[ra] < u.rank[rb]:
		u.parent[ra] = rb
	case u.rank[ra] > u.rank[rb]:
		u.parent[rb] = ra
	default:
		u.parent[rb] = ra
		u.rank[ra]++
	}
}
