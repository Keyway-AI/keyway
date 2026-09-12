package cli

import (
	"testing"

	"github.com/Keyway-AI/keyway/internal/agentauth"
)

// TestDemoTokenSurfacesFindings guards the `--demo` onboarding experience: the
// built-in sample token must keep surfacing the spread of findings the README
// promises, so a newcomer's first run always shows something.
func TestDemoTokenSurfacesFindings(t *testing.T) {
	findings, err := agentauth.Analyze(demoToken(), agentauth.Policy{
		Audience: "https://mcp.example/api", RequireDelegation: true,
	})
	if err != nil {
		t.Fatalf("demo token failed to analyze: %v", err)
	}
	want := map[string]bool{"MCP-02": false, "SCOPE-01": false, "SCOPE-02": false, "DEL-02": false}
	for _, f := range findings {
		if _, ok := want[f.ThreatID]; ok {
			want[f.ThreatID] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("demo token no longer surfaces %s (got %d findings)", id, len(findings))
		}
	}
}
