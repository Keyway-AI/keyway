package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThreatCoverageEndpoint(t *testing.T) {
	h := testServer(t).Routes()
	w := do(t, h, "GET", "/v1/threats/coverage", "secret", nil)
	require.Equal(t, 200, w.Code)

	var body struct {
		Total   int `json:"total"`
		Covered int `json:"covered"`
		Domains []struct {
			Domain  string `json:"domain"`
			Covered int    `json:"covered"`
			Total   int    `json:"total"`
		} `json:"domains"`
		Threats []struct {
			ID     string `json:"id"`
			Domain string `json:"domain"`
		} `json:"threats"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Greater(t, body.Total, 40, "the taxonomy should span both domains")
	assert.Len(t, body.Domains, 2)
	assert.NotEmpty(t, body.Threats)
}

func TestAgentInspectEndpoint(t *testing.T) {
	h := testServer(t).Routes()
	// A token with no aud, no exp, and an omnibus scope should flag several threats.
	// header {"alg":"none"} . payload {"scope":"admin:*","sub":"a"} .
	token := "eyJhbGciOiJub25lIn0.eyJzY29wZSI6ImFkbWluOioiLCJzdWIiOiJhIn0."
	w := doKey(t, h, "POST", "/v1/agent/inspect", "secret", "k1", map[string]any{
		"token": token, "audience": "https://mcp.example/api", "require_delegation": true,
	})
	require.Equal(t, 200, w.Code)

	var body struct {
		Count    int `json:"count"`
		Findings []struct {
			ThreatID string `json:"threat_id"`
		} `json:"findings"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.GreaterOrEqual(t, body.Count, 3, "unbound aud + missing act + over-scope + no exp")

	ids := map[string]bool{}
	for _, f := range body.Findings {
		ids[f.ThreatID] = true
	}
	assert.True(t, ids["MCP-02"] && ids["DEL-01"] && ids["SCOPE-01"] && ids["SCOPE-02"])
}
