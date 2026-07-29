package threats

import "github.com/nometria/keyway/internal/model"

// agentThreats is the AI-agent auth threat surface (MCP, on-behalf-of delegation,
// agent identity). Everything here is currently a coverage gap: it is the honest
// denominator and roadmap for extending Keyway's verification model to agent auth
// (see docs/threat-coverage.md and the agent-auth landscape brief). Each entry is
// grounded in an external source — the MCP spec's normative MUSTs, OAuth RFCs
// (8693 token exchange, 8707 resource indicators, 9728 protected-resource
// metadata, 9700 OAuth BCP), OWASP's LLM/Agentic Top 10s, and published CVEs.
func agentThreats() []Threat {
	return []Threat{
		// ---- token binding / passthrough -----------------------------------
		{
			ID: "MCP-01", Title: "token passthrough (wrong-audience token accepted)", Category: CatTokenBinding, Severity: model.SeverityCritical,
			Description: "An MCP/resource server accepts an access token that was not issued to it and forwards it downstream, breaking audience validation, rate-limiting, and the audit trail — turning the server into an exfiltration proxy for a stolen token.",
			Invariant:   "A resource server MUST reject any token whose audience is not its own canonical URI; it MUST NOT accept or forward tokens issued for another party.",
			Sources:     []Source{mcp("token passthrough anti-pattern"), rfc("8707", "Resource Indicators for OAuth 2.0"), cwe("287", "Improper Authentication")},
			Detections:  []Detection{analyzerDet("aud_mismatch"), harnessDet("resource_passthrough")},
		},
		{
			ID: "MCP-02", Title: "missing resource indicator / unbound audience", Category: CatTokenBinding, Severity: model.SeverityHigh,
			Description: "The client omits the `resource` parameter, so the issued token is not bound to the specific MCP server and can be replayed against a different one.",
			Invariant:   "Clients MUST send the `resource` indicator on authorization and token requests; issued tokens MUST be audience-bound to the target resource.",
			Sources:     []Source{rfc("8707", "Resource Indicators for OAuth 2.0"), rfc("9728", "OAuth 2.0 Protected Resource Metadata"), mcp("authorization")},
			Detections:  []Detection{analyzerDet("aud_unbound"), harnessDet("unbound_audience")},
		},

		// ---- consent / confused deputy -------------------------------------
		{
			ID: "CD-01", Title: "confused deputy via static client ID", Category: CatConsent, Severity: model.SeverityCritical,
			Description: "A proxy MCP server using a static OAuth client ID to a third-party AS, plus dynamic client registration and a persisted consent cookie, lets an attacker craft a malicious redirect_uri that reuses the victim's existing consent to skip the consent screen and steal the authorization code.",
			Invariant:   "An OAuth proxy MUST obtain per-client user consent, enforce exact redirect_uri matching, and validate `state`; it MUST NOT let a static client ID + prior consent bypass approval.",
			Sources:     []Source{mcp("confused deputy"), owasp("CSA: AI agent confused deputy", "https://labs.cloudsecurityalliance.org/research/csa-research-note-ai-agent-confused-deputy-prompt-injection/"), cwe("441", "Unintended Proxy or Intermediary (Confused Deputy)")},
		},
		{
			ID: "CD-02", Title: "dynamic client registration abuse", Category: CatConsent, Severity: model.SeverityHigh,
			Description: "Open dynamic client registration lets an attacker register a client with an attacker-controlled redirect_uri or inflated metadata, seeding phishing/confused-deputy flows.",
			Invariant:   "DCR MUST validate and constrain client metadata (redirect_uris, grant types); prefer Client-ID Metadata Documents over open RFC 7591 registration.",
			Sources:     []Source{rfc("7591", "OAuth 2.0 Dynamic Client Registration"), mcp("client registration"), draft("OAuth Client ID Metadata Documents", "https://datatracker.ietf.org/doc/draft-ietf-oauth-client-id-metadata-document/")},
		},

		// ---- delegation ----------------------------------------------------
		{
			ID: "DEL-01", Title: "missing or forged act (delegation) claim", Category: CatDelegation, Severity: model.SeverityHigh,
			Description: "An on-behalf-of token omits the `act` claim (or carries a forged one), so the agent is treated as the user with no verifiable record of who is acting for whom — the delegation audit chain is broken.",
			Invariant:   "A delegated token MUST carry a verifiable `act` chain recording the actor; verifiers requiring delegation MUST reject tokens without it.",
			Sources:     []Source{rfc("8693", "OAuth 2.0 Token Exchange (act/may_act)"), draft("On-Behalf-Of User for AI Agents", "https://www.ietf.org/archive/id/draft-oauth-ai-agents-on-behalf-of-user-00.html")},
			Detections:  []Detection{analyzerDet("missing_act")},
		},
		{
			ID: "DEL-02", Title: "delegation-chain / transitive-trust abuse", Category: CatDelegation, Severity: model.SeverityHigh,
			Description: "In agent-to-agent or on-behalf-of chains, a token minted for one hop is accepted at another because each hop does not independently validate audience and delegation purpose — trust becomes transitive.",
			Invariant:   "Each hop MUST independently validate audience and the delegation purpose; a token bound to one hop MUST NOT be accepted at another.",
			Sources:     []Source{rfc("8693", "OAuth 2.0 Token Exchange"), owasp("OWASP Agentic Top 10 (2025)", "https://genai.owasp.org/2025/12/09/owasp-genai-security-project-releases-top-10-risks-and-mitigations-for-agentic-ai-security/"), rfc("9700", "OAuth 2.0 Security BCP")},
			Detections:  []Detection{harnessDet("sibling_resource")},
		},
		{
			ID: "DEL-03", Title: "may_act not enforced on token exchange", Category: CatDelegation, Severity: model.SeverityHigh,
			Description: "A token is exchanged for an actor the subject never authorized, because the authorization server does not enforce the `may_act` constraint.",
			Invariant:   "Token exchange MUST honor `may_act`: only pre-authorized actors may obtain a delegated token for a subject.",
			Sources:     []Source{rfc("8693", "OAuth 2.0 Token Exchange (may_act)")},
		},
		{
			ID: "DEL-04", Title: "impersonation where delegation is required", Category: CatDelegation, Severity: model.SeverityMedium,
			Description: "An impersonation token (agent fully assumes the user's identity) is issued/accepted where delegation was intended, erasing agent attribution and accountability.",
			Invariant:   "Where policy requires delegation, impersonation tokens MUST be rejected; the actor MUST remain attributable in the token and audit log.",
			Sources:     []Source{rfc("8693", "OAuth 2.0 Token Exchange (delegation vs impersonation)"), draft("OpenID: Identity Management for Agentic AI", "https://openid.net/wp-content/uploads/2025/10/Identity-Management-for-Agentic-AI.pdf")},
		},

		// ---- scope ---------------------------------------------------------
		{
			ID: "SCOPE-01", Title: "over-scoped agent credential", Category: CatScope, Severity: model.SeverityHigh,
			Description: "The agent holds omnibus scopes (files:*, admin:*, full-access) far beyond the tools it uses, so a single injection or token leak grants wide lateral movement and privilege chaining.",
			Invariant:   "Agent tokens MUST follow least privilege: granted scopes minimal versus the tools the agent actually declares/uses.",
			Sources:     []Source{mcp("scope minimization"), owasp("OWASP LLM06 Excessive Agency (2025)", "https://owasp.org/www-project-top-10-for-large-language-model-applications/")},
			Detections:  []Detection{analyzerDet("over_scope")},
		},
		{
			ID: "SCOPE-02", Title: "non-expiring / long-lived agent token", Category: CatScope, Severity: model.SeverityHigh,
			Description: "The agent authenticates with a static, long-lived key that is never rotated or revoked — the most common non-human-identity failure, with NHIs now vastly outnumbering humans.",
			Invariant:   "Agent credentials MUST be short-lived and rotatable, with zero standing privilege between tasks.",
			Sources:     []Source{owasp("2026 State of AI Agent Identity", "https://securityboulevard.com/2026/07/the-agent-identity-problem-non-human-identities-outnumber-humans-45-to-1-and-ai-agents-are-making-it-worse/"), cwe("798", "Use of Hard-coded Credentials")},
			Detections:  []Detection{analyzerDet("non_expiring")},
		},
		{
			ID: "SCOPE-03", Title: "agent reuses the human's root credential", Category: CatScope, Severity: model.SeverityHigh,
			Description: "The agent is handed the user's own broad credential rather than its own scoped identity, so it can do anything the user can and its actions are indistinguishable from theirs.",
			Invariant:   "An agent MUST act with its own scoped identity and delegated (act) token, never the user's raw credential.",
			Sources:     []Source{draft("Auth0: Auth for AI Agents (Token Vault)", "https://auth0.com/blog/auth0-for-ai-agents-generally-available/"), mcp("authorization")},
		},

		// ---- agent identity ------------------------------------------------
		{
			ID: "AID-01", Title: "session used as authentication", Category: CatAgentIdentity, Severity: model.SeverityHigh,
			Description: "An MCP server treats a session ID as proof of identity; a guessable or unbound session ID enables hijacking or impersonation.",
			Invariant:   "Servers MUST NOT use sessions for authentication; session IDs MUST be non-deterministic and bound to user identity.",
			Sources:     []Source{mcp("session hijacking"), cwe("384", "Session Fixation")},
		},
		{
			ID: "AID-02", Title: "no distinct workload/agent identity", Category: CatAgentIdentity, Severity: model.SeverityMedium,
			Description: "The agent has no verifiable workload identity (e.g. SPIFFE/WIMSE), so it cannot be authenticated, attributed, or governed as a first-class principal.",
			Invariant:   "Each agent/workload MUST have a verifiable identity credential distinct from the human and from other agents.",
			Sources:     []Source{draft("IETF WIMSE — Workload Identity", "https://datatracker.ietf.org/doc/draft-ietf-wimse-workload-identity-practices/"), draft("SPIFFE", "https://spiffe.io/")},
		},

		// ---- agency (injection → tool misuse) ------------------------------
		{
			ID: "AGT-01", Title: "prompt-injection-driven privilege escalation", Category: CatAgency, Severity: model.SeverityCritical,
			Description: "The master exploit: the agent cannot distinguish injected instructions (from web pages, tool output, documents) from legitimate ones, so any tool it holds becomes reachable by an attacker who controls the data it reads. Over-scoped tokens turn one injection into lateral movement.",
			Invariant:   "High-impact actions MUST require per-action authorization or human-in-the-loop; tools MUST be least-privilege so injection cannot escalate.",
			Sources:     []Source{owasp("OWASP LLM01 Prompt Injection (2025)", "https://owasp.org/www-project-top-10-for-large-language-model-applications/"), owasp("OWASP Agentic Top 10 (2025)", "https://genai.owasp.org/2025/12/09/owasp-genai-security-project-releases-top-10-risks-and-mitigations-for-agentic-ai-security/")},
		},
		{
			ID: "AGT-02", Title: "tool poisoning / rug-pull", Category: CatAgency, Severity: model.SeverityHigh,
			Description: "A tool mutates its description or behavior after the user approved it; because the host does not require re-approval on change, the malicious version is loaded silently.",
			Invariant:   "A tool's definition/behavior change MUST invalidate prior approval and require re-consent; tool definitions SHOULD be integrity-pinned.",
			Sources:     []Source{cve("CVE-2025-54136"), draft("MCP tool poisoning / rug-pull", "https://www.truefoundry.com/blog/blog-mcp-tool-poisoning-gateway-defense")},
		},
	}
}
