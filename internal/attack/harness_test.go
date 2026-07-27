package attack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCorpusIsValid is the anti-circularity check: a known-correct verifier (the
// go-jose-based oracle) must return the expected verdict for EVERY generated
// token — reject every attack, accept the control. If it doesn't, the generator
// produced a token that doesn't actually have the property it claims.
func TestCorpusIsValid(t *testing.T) {
	ctx, pol, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := Corpus(ctx)
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	if len(corpus) < 15 {
		t.Fatalf("corpus implausibly small: %d", len(corpus))
	}
	oracle := Oracle{Policy: pol}

	findings, err := Evaluate(context.Background(), corpus, oracle.Verify)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	for _, f := range findings {
		t.Errorf("oracle disagreed on %s/%s: expected %s, got %s (%s)",
			f.Token.ThreatID, f.Token.Name, f.Token.Expect, f.Actual, f.Token.Rationale)
	}
}

// TestCatchesVulnerableTarget proves the harness detects a broken verifier: a
// target that honors any token must produce a finding for every attack token.
func TestCatchesVulnerableTarget(t *testing.T) {
	ctx, _, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := Corpus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// A verifier that accepts everything (the classic "decodes but never verifies").
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	target := HTTPTarget{URL: srv.URL, Client: srv.Client()}
	findings, err := Evaluate(context.Background(), corpus, target.Verify)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	attacks := 0
	for _, tok := range corpus {
		if tok.Expect == Reject {
			attacks++
		}
	}
	vulns := 0
	for _, f := range findings {
		if f.Vulnerability() {
			vulns++
		}
	}
	if vulns != attacks {
		t.Fatalf("expected every attack (%d) to be flagged against an accept-all target; got %d", attacks, vulns)
	}
}

// TestClearsCorrectTarget proves no false positives: a target backed by the
// reference oracle produces zero findings.
func TestClearsCorrectTarget(t *testing.T) {
	ctx, pol, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := Corpus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	oracle := Oracle{Policy: pol}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		const p = "Bearer "
		if len(raw) < len(p) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		v, _ := oracle.Verify(r.Context(), raw[len(p):])
		if v == Accept {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()

	target := HTTPTarget{URL: srv.URL, Client: srv.Client()}
	findings, err := Evaluate(context.Background(), corpus, target.Verify)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(findings) != 0 {
		for _, f := range findings {
			t.Errorf("false positive on %s/%s: expected %s, got %s", f.Token.ThreatID, f.Token.Name, f.Token.Expect, f.Actual)
		}
	}
}

// TestCoveredThreatsAreSelfContained verifies the coverage bridge only counts
// end-to-end-detectable threats (excludes callback-dependent jku/x5u).
func TestCoveredThreatsAreSelfContained(t *testing.T) {
	ctx, _, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := Corpus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	covered := CoveredThreatIDs(corpus)
	if len(covered) < 12 {
		t.Fatalf("expected the harness to cover a broad set of threats, got %d: %v", len(covered), covered)
	}
	for _, id := range covered {
		if id == "HDR-01" || id == "HDR-02" {
			t.Errorf("%s is callback-dependent and must NOT count as covered", id)
		}
	}
}
