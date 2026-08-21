// Package cooccur measures how the prevalence checks co-occur across the corpus:
// given each config's set of flagged weaknesses, it reports, for every pair of
// checks, how often they appear together and the conditional probability of one
// given the other. It answers questions the marginal prevalences cannot, e.g. "of
// the configs with an unbound audience, how many also require no claims?" — the
// co-occurrence analysis named as future work in the Paper A gate.
//
// It is descriptive only: counts and ratios over the observed denominator, no
// inference. Lift (observed joint / independence) flags pairs that travel together
// more than chance, but at small N it is noisy and is reported, not interpreted.
package cooccur

import "sort"

// Pair is the co-occurrence of checks A and B over N configs.
type Pair struct {
	A, B        string  `json:"-"`
	Key         string  `json:"pair"`
	N           int     `json:"n"`
	CountA      int     `json:"count_a"`
	CountB      int     `json:"count_b"`
	Both        int     `json:"both"`
	JointPrev   float64 `json:"joint_prevalence"` // Both / N
	ProbBGivenA float64 `json:"prob_b_given_a"`   // Both / CountA
	ProbAGivenB float64 `json:"prob_a_given_b"`   // Both / CountB
	Lift        float64 `json:"lift"`             // JointPrev / (pA*pB); 1 == independent
}

// Analyze computes a Pair for every unordered pair of checks. ids gives the check
// order (pairs are emitted in that order, i<j); flags is one weakness set per
// config: flags[k][id] is true when config k is flagged for check id. Configs whose
// map lacks an id count as not flagged.
func Analyze(ids []string, flags []map[string]bool) []Pair {
	n := len(flags)
	count := make(map[string]int, len(ids))
	for _, id := range ids {
		for _, f := range flags {
			if f[id] {
				count[id]++
			}
		}
	}
	var pairs []Pair
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := ids[i], ids[j]
			both := 0
			for _, f := range flags {
				if f[a] && f[b] {
					both++
				}
			}
			p := Pair{
				A: a, B: b, Key: a + " ∧ " + b, N: n,
				CountA: count[a], CountB: count[b], Both: both,
			}
			if n > 0 {
				p.JointPrev = float64(both) / float64(n)
			}
			if count[a] > 0 {
				p.ProbBGivenA = float64(both) / float64(count[a])
			}
			if count[b] > 0 {
				p.ProbAGivenB = float64(both) / float64(count[b])
			}
			// Lift = P(A∧B) / (P(A)·P(B)); >1 co-occur more than chance, <1 less.
			if n > 0 && count[a] > 0 && count[b] > 0 {
				pa := float64(count[a]) / float64(n)
				pb := float64(count[b]) / float64(n)
				p.Lift = p.JointPrev / (pa * pb)
			}
			pairs = append(pairs, p)
		}
	}
	return pairs
}

// ByJointDesc sorts pairs by joint prevalence, most-common combination first, so
// the printed table leads with the weaknesses that most often travel together.
func ByJointDesc(pairs []Pair) {
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].JointPrev > pairs[j].JointPrev
	})
}
