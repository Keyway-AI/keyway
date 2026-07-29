package attack

import (
	"context"
	"testing"
)

// TestAgentAudienceBindingAttacks verifies the corpus carries live agent-auth
// attacks (resource passthrough, unbound audience, sibling-hop token) and that a
// correct verifier — one that validates audience — rejects every one of them,
// while a passthrough-vulnerable verifier would accept them.
func TestAgentAudienceBindingAttacks(t *testing.T) {
	ctx, pol, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := Corpus(ctx)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{"MCP-01": false, "MCP-02": false, "DEL-02": false}
	oracle := Oracle{Policy: pol}
	for _, tok := range corpus {
		if _, tracked := want[tok.ThreatID]; !tracked {
			continue
		}
		want[tok.ThreatID] = true
		if tok.Expect != Reject || !tok.SelfContained {
			t.Errorf("%s/%s should be a self-contained reject attack", tok.ThreatID, tok.Name)
		}
		// A correct, audience-validating verifier must reject each.
		if v, _ := oracle.Verify(context.Background(), tok.JWS); v != Reject {
			t.Errorf("%s/%s: a correct verifier must reject a mis-bound token", tok.ThreatID, tok.Name)
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("expected an agent audience-binding attack for %s in the corpus", id)
		}
	}
}
