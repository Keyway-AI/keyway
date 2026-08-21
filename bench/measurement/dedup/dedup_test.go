package dedup

import (
	"math"
	"testing"
)

func TestSignatureCanonicalization(t *testing.T) {
	// These three configs differ only in meaning-preserving ways: a trailing slash
	// on the issuer, issuer host case, and algorithm case, plus list order. They
	// must collapse to one signature.
	base := Config{
		Issuers:    []string{"https://accounts.example.com"},
		Audiences:  []string{"api://gateway"},
		Algorithms: []string{"RS256"},
		Claims:     []string{"groups"},
	}
	variants := []Config{
		{Issuers: []string{"https://accounts.example.com/"}, Audiences: []string{"api://gateway"}, Algorithms: []string{"RS256"}, Claims: []string{"groups"}},
		{Issuers: []string{"https://Accounts.Example.com"}, Audiences: []string{"api://gateway"}, Algorithms: []string{"rs256"}, Claims: []string{"groups"}},
		{Issuers: []string{"https://accounts.example.com"}, Audiences: []string{"api://gateway"}, Algorithms: []string{"RS256"}, Claims: []string{"groups"}},
	}
	want := Signature(base)
	for i, v := range variants {
		if got := Signature(v); got != want {
			t.Errorf("variant %d signature = %q, want %q", i, got, want)
		}
	}
}

func TestSignatureDistinct(t *testing.T) {
	// A genuinely different audience must NOT collapse.
	a := Config{Issuers: []string{"https://x"}, Audiences: []string{"api://a"}}
	b := Config{Issuers: []string{"https://x"}, Audiences: []string{"api://b"}}
	if Signature(a) == Signature(b) {
		t.Error("configs with different audiences share a signature (over-collapse)")
	}
	// Case-sensitive audiences must stay distinct (we do not lowercase them).
	c := Config{Audiences: []string{"MyAud"}}
	d := Config{Audiences: []string{"myaud"}}
	if Signature(c) == Signature(d) {
		t.Error("audiences differing only in case were merged")
	}
}

func TestJaccard(t *testing.T) {
	a := Config{Issuers: []string{"https://x"}, Audiences: []string{"a", "b"}}
	b := Config{Issuers: []string{"https://x"}, Audiences: []string{"a", "c"}}
	// tokens(a) = {iss:https://x, aud:a, aud:b}; tokens(b) = {iss:https://x, aud:a, aud:c}
	// intersection = 2 (iss, aud:a); union = 4 → 0.5
	if got := Jaccard(a, b); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("Jaccard = %v, want 0.5", got)
	}
	if got := Jaccard(a, a); got != 1 {
		t.Errorf("Jaccard(a,a) = %v, want 1", got)
	}
	disjoint := Config{Issuers: []string{"https://y"}, Audiences: []string{"z"}}
	if got := Jaccard(a, disjoint); got != 0 {
		t.Errorf("Jaccard(disjoint) = %v, want 0", got)
	}
}

func TestNearDupReport(t *testing.T) {
	// Two near-identical configs (differ by one extra claim) and one clearly
	// different config. At a moderate threshold the near pair collapses; the
	// distinct one stays separate.
	cs := []Config{
		{Issuers: []string{"https://x"}, Audiences: []string{"api"}, Claims: []string{"groups"}},
		{Issuers: []string{"https://x"}, Audiences: []string{"api"}, Claims: []string{"groups", "roles"}},
		{Issuers: []string{"https://other"}, Audiences: []string{"different"}},
	}
	// Exact signatures: all three distinct (the extra claim differs).
	exact := map[string]struct{}{}
	for _, c := range cs {
		exact[Signature(c)] = struct{}{}
	}
	if len(exact) != 3 {
		t.Fatalf("expected 3 exact signatures, got %d", len(exact))
	}
	// Jaccard of the near pair: tokens {iss:x,aud:api,claim:groups} vs
	// {iss:x,aud:api,claim:groups,claim:roles} → 3/4 = 0.75.
	rep := NearDupReport(cs, 0.7, 0.9)
	if rep[0].Threshold != 0.7 || rep[0].Clusters != 2 || rep[0].Collapsed != 1 {
		t.Errorf("at 0.7: got clusters=%d collapsed=%d, want 2/1", rep[0].Clusters, rep[0].Collapsed)
	}
	if rep[1].Threshold != 0.9 || rep[1].Clusters != 3 || rep[1].Collapsed != 0 {
		t.Errorf("at 0.9: got clusters=%d collapsed=%d, want 3/0", rep[1].Clusters, rep[1].Collapsed)
	}
}

func TestClusterIDsTransitive(t *testing.T) {
	// Single-linkage: a~b and b~c (but a≁c) still land in one cluster.
	cs := []Config{
		{Audiences: []string{"a", "b", "c", "d"}},
		{Audiences: []string{"a", "b", "c", "e"}}, // ~ config 0 (3/5=0.6)
		{Audiences: []string{"a", "b", "f", "e"}}, // ~ config 1 but less like 0
	}
	ids := ClusterIDs(cs, 0.5)
	if ids[0] != ids[1] || ids[1] != ids[2] {
		t.Errorf("single-linkage chain not merged: ids = %v", ids)
	}
}
