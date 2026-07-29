package agentauth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// makeToken builds a compact JWT string from claims (signature is irrelevant — the
// analyzer inspects claims, not the signature).
func makeToken(claims map[string]any) string {
	hdr, _ := json.Marshal(map[string]any{"alg": "none", "typ": "JWT"})
	pay, _ := json.Marshal(claims)
	b := base64.RawURLEncoding.EncodeToString
	return b(hdr) + "." + b(pay) + "."
}

func has(findings []Finding, threatID string) bool {
	for _, f := range findings {
		if f.ThreatID == threatID {
			return true
		}
	}
	return false
}

func TestAnalyze_CleanToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	tok := makeToken(map[string]any{
		"aud":   "https://mcp.example/api",
		"act":   map[string]any{"sub": "agent-1"},
		"scope": "files:read tasks:write",
		"iat":   float64(now.Unix()),
		"exp":   float64(now.Add(10 * time.Minute).Unix()),
	})
	pol := Policy{
		Audience: "https://mcp.example/api", RequireDelegation: true,
		MaxLifetime: time.Hour, AllowedScopes: []string{"files:read", "tasks:write"}, Now: now,
	}
	f, err := Analyze(tok, pol)
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("expected no findings for a clean token, got %+v", f)
	}
}

func TestAnalyze_AudienceUnbound(t *testing.T) {
	tok := makeToken(map[string]any{"exp": float64(time.Now().Add(time.Minute).Unix())})
	f, _ := Analyze(tok, Policy{Audience: "https://mcp.example/api"})
	if !has(f, ThreatAudienceUnbound) {
		t.Fatalf("expected MCP-02 (unbound audience), got %+v", f)
	}
}

func TestAnalyze_AudienceMismatch(t *testing.T) {
	tok := makeToken(map[string]any{"aud": "https://other.example", "exp": float64(time.Now().Add(time.Minute).Unix())})
	f, _ := Analyze(tok, Policy{Audience: "https://mcp.example/api"})
	if !has(f, ThreatAudienceMismatch) {
		t.Fatalf("expected MCP-01 (audience mismatch), got %+v", f)
	}
}

func TestAnalyze_MissingDelegation(t *testing.T) {
	tok := makeToken(map[string]any{"aud": "r", "exp": float64(time.Now().Add(time.Minute).Unix())})
	f, _ := Analyze(tok, Policy{RequireDelegation: true})
	if !has(f, ThreatMissingDelegation) {
		t.Fatalf("expected DEL-01 (missing act), got %+v", f)
	}
}

func TestAnalyze_OverScope(t *testing.T) {
	now := time.Now()
	// Wildcard scope is always over-broad.
	tok := makeToken(map[string]any{"scope": "files:* profile", "exp": float64(now.Add(time.Minute).Unix())})
	f, _ := Analyze(tok, Policy{})
	if !has(f, ThreatOverScope) {
		t.Fatalf("expected SCOPE-01 for a wildcard scope, got %+v", f)
	}

	// Beyond an explicit allowlist.
	tok2 := makeToken(map[string]any{"scope": "files:read billing:write", "exp": float64(now.Add(time.Minute).Unix())})
	f2, _ := Analyze(tok2, Policy{AllowedScopes: []string{"files:read"}})
	if !has(f2, ThreatOverScope) {
		t.Fatalf("expected SCOPE-01 for a scope beyond the allowlist, got %+v", f2)
	}
}

func TestAnalyze_Expiry(t *testing.T) {
	// No exp at all.
	f, _ := Analyze(makeToken(map[string]any{"aud": "r"}), Policy{})
	if !has(f, ThreatNonExpiring) {
		t.Fatalf("expected SCOPE-02 for a token with no exp, got %+v", f)
	}

	// Lifetime beyond the bound.
	now := time.Unix(1_700_000_000, 0)
	tok := makeToken(map[string]any{
		"iat": float64(now.Unix()), "exp": float64(now.Add(30 * 24 * time.Hour).Unix()),
	})
	f2, _ := Analyze(tok, Policy{MaxLifetime: time.Hour, Now: now})
	if !has(f2, ThreatNonExpiring) {
		t.Fatalf("expected SCOPE-02 for an over-long lifetime, got %+v", f2)
	}
}

func TestAnalyze_RejectsNonJWT(t *testing.T) {
	if _, err := Analyze("not-a-token", Policy{}); err == nil {
		t.Fatal("expected an error for a non-JWT input")
	}
}
