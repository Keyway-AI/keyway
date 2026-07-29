// Package agentauth statically analyzes an agent / MCP / on-behalf-of token
// against the agent-auth invariants in the threat taxonomy (internal/threats,
// "agent" domain). Unlike the live probe/harness, this needs no endpoint and does
// not verify the signature — it inspects the token's own hygiene: is its audience
// bound to the resource, does it carry the delegation (act) claim, are its scopes
// minimal, and does it actually expire? Each finding is keyed to a threat ID so
// the coverage taxonomy can mark those threats detected.
package agentauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nometria/keyway/internal/model"
)

// Threat IDs this analyzer checks — the bridge to internal/threats. A test asserts
// the taxonomy marks exactly these as analyzer-covered.
const (
	ThreatAudienceMismatch  = "MCP-01"   // aud present but not bound to the resource
	ThreatAudienceUnbound   = "MCP-02"   // aud absent → token bound to nothing
	ThreatMissingDelegation = "DEL-01"   // on-behalf-of token missing act
	ThreatOverScope         = "SCOPE-01" // omnibus / beyond-allowlist scopes
	ThreatNonExpiring       = "SCOPE-02" // no exp, or lifetime beyond the bound
)

// CheckedThreatIDs returns the distinct, sorted threat IDs this analyzer can
// detect. Keep in lockstep with the taxonomy's DetAnalyzer marks (bridge test).
func CheckedThreatIDs() []string {
	ids := []string{ThreatAudienceMismatch, ThreatAudienceUnbound, ThreatMissingDelegation, ThreatOverScope, ThreatNonExpiring}
	sort.Strings(ids)
	return ids
}

// Policy is what a correct agent token must satisfy for this deployment.
type Policy struct {
	Audience          string        // expected resource URI / audience (RFC 8707/9728)
	RequireDelegation bool          // an agent/OBO token MUST carry act (RFC 8693)
	MaxLifetime       time.Duration // 0 = don't bound; otherwise exp-iat must be ≤ this
	AllowedScopes     []string      // if set, scopes must be a subset (wildcards always flagged)
	Now               time.Time     // defaults to time.Now
}

// Finding is one violated invariant.
type Finding struct {
	ThreatID string         `json:"threat_id"`
	Severity model.Severity `json:"severity"`
	Message  string         `json:"message"`
}

// omnibusScopes are always-over-broad scope tokens regardless of allowlist.
var omnibusScopes = map[string]bool{
	"*": true, "admin": true, "root": true, "superuser": true,
	"full_access": true, "full-access": true, "all": true,
}

// Analyze parses the token's claims (no signature verification) and returns every
// agent-auth invariant it violates under the policy.
func Analyze(token string, p Policy) ([]Finding, error) {
	claims, err := decodeClaims(token)
	if err != nil {
		return nil, err
	}
	now := p.Now
	if now.IsZero() {
		now = time.Now()
	}
	var out []Finding

	// --- audience binding (MCP-01 / MCP-02) --------------------------------
	if p.Audience != "" {
		aud, present := claims["aud"]
		switch {
		case !present || isEmptyAud(aud):
			out = append(out, Finding{ThreatAudienceUnbound, model.SeverityHigh,
				"token carries no audience — it is not bound to any resource (RFC 8707/9728)"})
		case !audienceContains(aud, p.Audience):
			out = append(out, Finding{ThreatAudienceMismatch, model.SeverityCritical,
				fmt.Sprintf("audience does not include the resource %q — token could be replayed or passed through to another server", p.Audience)})
		}
	}

	// --- delegation / act (DEL-01) -----------------------------------------
	if p.RequireDelegation {
		if act, ok := claims["act"]; !ok || act == nil {
			out = append(out, Finding{ThreatMissingDelegation, model.SeverityHigh,
				"on-behalf-of token is missing the act claim — the delegation chain is unverifiable (RFC 8693)"})
		}
	}

	// --- scope minimization (SCOPE-01) -------------------------------------
	scopes := parseScopes(claims)
	allowed := toSet(p.AllowedScopes)
	for _, s := range scopes {
		low := strings.ToLower(s)
		switch {
		case omnibusScopes[low] || strings.HasSuffix(s, ":*") || strings.HasSuffix(s, ".*") || strings.HasPrefix(low, "admin:"):
			out = append(out, Finding{ThreatOverScope, model.SeverityHigh,
				fmt.Sprintf("over-broad scope %q — grant least privilege, not omnibus access", s)})
		case len(allowed) > 0 && !allowed[s]:
			out = append(out, Finding{ThreatOverScope, model.SeverityMedium,
				fmt.Sprintf("scope %q is beyond the declared allowlist", s)})
		}
	}

	// --- expiry (SCOPE-02) -------------------------------------------------
	exp, hasExp := numClaim(claims, "exp")
	switch {
	case !hasExp:
		out = append(out, Finding{ThreatNonExpiring, model.SeverityHigh,
			"token has no exp — a non-expiring agent credential"})
	case p.MaxLifetime > 0:
		// Lifetime from iat when present; otherwise the remaining time from now.
		life := time.Unix(int64(exp), 0).Sub(now)
		if iat, ok := numClaim(claims, "iat"); ok {
			life = time.Duration(exp-iat) * time.Second
		}
		if life > p.MaxLifetime {
			out = append(out, Finding{ThreatNonExpiring, model.SeverityMedium,
				fmt.Sprintf("token lifetime %s exceeds the maximum %s", life, p.MaxLifetime)})
		}
	}
	return out, nil
}

// --- claim helpers ----------------------------------------------------------

func decodeClaims(token string) (map[string]any, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("agentauth: not a JWT (want at least header.payload)")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, fmt.Errorf("agentauth: decode payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, fmt.Errorf("agentauth: parse claims: %w", err)
	}
	return claims, nil
}

func numClaim(c map[string]any, key string) (float64, bool) {
	f, ok := c[key].(float64)
	return f, ok
}

func isEmptyAud(aud any) bool {
	switch v := aud.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case nil:
		return true
	}
	return false
}

func audienceContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

// parseScopes reads OAuth scopes from `scope` (space-delimited string) or `scp`
// (string or array), the two common encodings.
func parseScopes(c map[string]any) []string {
	var out []string
	if s, ok := c["scope"].(string); ok {
		out = append(out, strings.Fields(s)...)
	}
	switch v := c["scp"].(type) {
	case string:
		out = append(out, strings.Fields(v)...)
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func toSet(ss []string) map[string]bool {
	if len(ss) == 0 {
		return nil
	}
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
