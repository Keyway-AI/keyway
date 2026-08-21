package cooccur

import (
	"math"
	"testing"
)

func flag(pairs ...string) map[string]bool {
	m := map[string]bool{}
	for _, p := range pairs {
		m[p] = true
	}
	return m
}

func find(t *testing.T, pairs []Pair, a, b string) Pair {
	t.Helper()
	for _, p := range pairs {
		if p.A == a && p.B == b {
			return p
		}
	}
	t.Fatalf("pair %s∧%s not found", a, b)
	return Pair{}
}

func TestAnalyze(t *testing.T) {
	ids := []string{"P1", "P2", "P3"}
	// 4 configs:
	//  c0: P1,P2      c1: P1,P2,P3     c2: P2,P3      c3: (none)
	// count: P1=2, P2=3, P3=2
	flags := []map[string]bool{
		flag("P1", "P2"),
		flag("P1", "P2", "P3"),
		flag("P2", "P3"),
		flag(),
	}
	pairs := Analyze(ids, flags)
	if len(pairs) != 3 { // C(3,2)
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	p12 := find(t, pairs, "P1", "P2")
	if p12.Both != 2 || p12.CountA != 2 || p12.CountB != 3 || p12.N != 4 {
		t.Errorf("P1∧P2 counts = both %d, A %d, B %d, N %d; want 2,2,3,4",
			p12.Both, p12.CountA, p12.CountB, p12.N)
	}
	// Every P1 config also has P2 → P(P2|P1) = 1.
	if math.Abs(p12.ProbBGivenA-1) > 1e-9 {
		t.Errorf("P(P2|P1) = %v, want 1", p12.ProbBGivenA)
	}
	// P(P1|P2) = 2/3.
	if math.Abs(p12.ProbAGivenB-2.0/3.0) > 1e-9 {
		t.Errorf("P(P1|P2) = %v, want 0.667", p12.ProbAGivenB)
	}
	// Joint prevalence = 2/4 = 0.5.
	if math.Abs(p12.JointPrev-0.5) > 1e-9 {
		t.Errorf("joint = %v, want 0.5", p12.JointPrev)
	}

	// P1∧P3: only c1 has both → both=1; P(P3|P1)=1/2, P(P1|P3)=1/2.
	p13 := find(t, pairs, "P1", "P3")
	if p13.Both != 1 || math.Abs(p13.ProbBGivenA-0.5) > 1e-9 {
		t.Errorf("P1∧P3: both=%d P(P3|P1)=%v, want 1 / 0.5", p13.Both, p13.ProbBGivenA)
	}
}

func TestLiftIndependent(t *testing.T) {
	// Construct near-independence: P(A)=0.5, P(B)=0.5, P(A∧B)=0.25 → lift 1.
	ids := []string{"A", "B"}
	flags := []map[string]bool{
		flag("A", "B"), flag("A"), flag("B"), flag(),
	}
	p := Analyze(ids, flags)[0]
	if math.Abs(p.Lift-1) > 1e-9 {
		t.Errorf("lift = %v, want 1 (independent)", p.Lift)
	}
}

func TestByJointDesc(t *testing.T) {
	pairs := []Pair{{Key: "low", JointPrev: 0.1}, {Key: "high", JointPrev: 0.9}}
	ByJointDesc(pairs)
	if pairs[0].Key != "high" {
		t.Errorf("sort order wrong: %v", pairs)
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	pairs := Analyze([]string{"P1", "P2"}, nil)
	if len(pairs) != 1 || pairs[0].N != 0 || pairs[0].Lift != 0 {
		t.Errorf("empty corpus: got %+v", pairs)
	}
}
