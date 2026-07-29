package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nometria/keyway/internal/agentauth"
	"github.com/nometria/keyway/internal/threats"
	"github.com/nometria/keyway/pkg/apitypes"
)

// handleThreatCoverage returns the threat taxonomy and Keyway's measured coverage
// of it (both domains), for the web Coverage view and CI dashboards.
func (s *Server) handleThreatCoverage(w http.ResponseWriter, r *http.Request) {
	cat := threats.Catalog()
	rep := threats.Compute(cat)

	out := apitypes.ThreatCoverageResponse{
		Total: rep.Total, Covered: rep.Covered, Gaps: len(rep.Gaps), Percent: rep.Pct(),
	}
	for _, d := range rep.Domains {
		out.Domains = append(out.Domains, apitypes.CoverageDomain{
			Domain: string(d.Domain), Covered: d.Covered, Total: d.Total, Percent: d.Pct(),
		})
	}
	for _, c := range rep.Categories {
		out.Categories = append(out.Categories, apitypes.CoverageCategory{
			Category: string(c.Category), Covered: c.Covered, Total: c.Total,
		})
	}
	for _, t := range cat {
		ct := apitypes.CoverageThreat{
			ID: t.ID, Domain: string(t.Domain), Category: string(t.Category),
			Severity: t.Severity, Title: t.Title, Invariant: t.Invariant,
			Detectors: make([]string, 0, len(t.Detections)),
		}
		for _, src := range t.Sources {
			ct.Sources = append(ct.Sources, apitypes.CoverageSource{Ref: src.Ref, URL: src.URL})
		}
		for _, d := range t.Detections {
			ct.Detectors = append(ct.Detectors, fmt.Sprintf("%s:%s", d.Kind, d.ID))
		}
		out.Threats = append(out.Threats, ct)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAgentInspect statically checks a submitted agent/MCP/OBO token against the
// agent-auth invariants (no signature verification) and returns the findings.
func (s *Server) handleAgentInspect(w http.ResponseWriter, r *http.Request) {
	var req apitypes.AgentInspectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "no_token", "token is required")
		return
	}
	findings, err := agentauth.Analyze(req.Token, agentauth.Policy{
		Audience:          req.Audience,
		RequireDelegation: req.RequireDelegation,
		MaxLifetime:       time.Duration(req.MaxLifetimeSeconds) * time.Second,
		AllowedScopes:     req.AllowedScopes,
		Now:               time.Now(),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_token", err.Error())
		return
	}
	if findings == nil {
		findings = []agentauth.Finding{}
	}
	writeJSON(w, http.StatusOK, apitypes.AgentInspectResponse{Findings: findings, Count: len(findings)})
}
