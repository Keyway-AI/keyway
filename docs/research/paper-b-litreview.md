# Paper B Literature Survey: Authorization for Autonomous Agents

Research literature on authorization, delegation, identity, and token security for autonomous AI agents and the Model Context Protocol (MCP), 2023–2026. This survey answers the SSR 2026 reviewer complaint that the prior related-work section cited only *specifications* (OAuth RFCs, the MCP spec) and did not systematize the actual *research* literature.

**Scope and method.** Candidate works were gathered via web search and every arXiv identifier below was verified to exist against the arXiv API (`export.arxiv.org`) on 2026-10-06; titles and author lists are taken from the returned metadata. Works that could not be verified were excluded. One frequently-surfaced paper, "Give Them an Inch and They Will Take a Mile" (arXiv:2603.07473) on MCP caller-identity confusion, was **excluded** because its authors formally withdrew it for methodology and ethics flaws. BibTeX keys referenced here are defined in `paper-b-refs-new.bib`.

**Peer-review status.** Most agent-authorization work is recent and lives on arXiv. We mark peer-reviewed/published venues explicitly; everything else should be read as a preprint or technical report (not yet peer-reviewed). Verified peer-reviewed works in this set: Greshake et al. (ACM AISec 2023), AgentDojo (NeurIPS 2024 D&B), IsolateGPT (NDSS 2025), CaMeL / "Defeating Prompt Injections by Design" (IEEE SaTML 2026), and MCPXKIT (accepted, IEEE TDSC).

**Threat categories** used throughout (the columns of the matrix in §2):

- **AudBind** — audience binding / token passthrough (RFC 8707 resource indicators; the MCP prohibition on forwarding a client token to an upstream API).
- **OBO** — on-behalf-of delegation and the `act`/`azp` claims (distinguishing the acting agent from the delegating user).
- **ScopeMin** — scope minimization / least privilege for tool calls.
- **ConfDep** — confused-deputy risk and dynamic client registration (RFC 7591).
- **WID** — workload / agent identity (WIMSE, SPIFFE/SPIRE, non-human identity).
- **PromptInj** — prompt-injection-driven privilege escalation.
- **ToolPoison** — tool poisoning (malicious or concealed tool descriptions/metadata).

---

## 1. Grouped survey

### Theme A — MCP security landscape, taxonomies, and SoKs

**Hou, Zhao, Wang & Wang — MCP: Landscape, Security Threats, and Future Research Directions** [`agentauth2025mcplandscape`, arXiv:2503.23278, 2025]. The foundational and most-cited MCP security survey. It models the MCP server lifecycle (creation, deployment, operation, maintenance) and enumerates 16 representative threats across four attacker types, including tool poisoning, installer spoofing, and unauthorized access. Broad but not authorization-specific; treats token and consent handling only at the level of named risks.

**Gaire et al. — SoK: Security and Safety in the MCP Ecosystem** [`agentauth2025sokmcp`, arXiv:2512.08290, 2025]. A systematization that consolidates MCP attack surfaces and defenses across the ecosystem and separates "security" from "safety" concerns. Useful as a structural precedent for our own SoK, though it organizes by ecosystem component rather than by authorization property or testability.

**Guo et al. — MCPXKIT: The Unified Toolkit for Analyzing MCP Security** [`agentauth2025mcpxkit`, arXiv:2508.12538, accepted IEEE TDSC]. Builds a unified analysis toolkit and empirically characterizes MCP security weaknesses at scale. Peer-reviewed; strong on measurement of injection and poisoning but light on delegation/token semantics.

**Shen, Toyoda & Leung — MCP-38: A Comprehensive Threat Taxonomy** [`agentauth2026mcp38`, arXiv:2603.18063, 2026]. Proposes a 38-item threat taxonomy for MCP systems, explicitly including authorization-adjacent threats (confused deputy, over-broad consent). One of the few taxonomies to name confused-deputy and registration risks as first-class entries.

**Rostamzadeh et al. — MCP-DPT: A Defense-Placement Taxonomy and Coverage Analysis** [`agentauth2026mcpdpt`, arXiv:2604.07551, 2026]. Classifies MCP defenses by *where* in the architecture they sit (client, host, server, gateway) and analyzes coverage gaps. The placement lens is complementary to, but distinct from, our static-vs-runtime enforcement line.

**Narajala & Habler — Enterprise-Grade Security for MCP: Frameworks and Mitigation Strategies** [`agentauth2025enterprisemcp`, arXiv:2504.08623, 2025]. A practitioner-oriented framework spanning authentication, token handling, least privilege, and confused-deputy mitigation for enterprise MCP deployments. The broadest single treatment of the authorization threats we care about, though prescriptive rather than evaluative.

### Theme B — MCP protocol-level flaws: audience/origin binding, confused deputy, passthrough

**Maloyan & Namiot — Breaking the Protocol** [`agentauth2026breakingprotocol`, arXiv:2601.17549, 2026]. A rigorous security analysis of the MCP specification itself, identifying protocol-level weaknesses: absence of capability attestation, bidirectional sampling without origin authentication (enabling server-side injection), and implicit trust propagation across chained servers. Directly relevant to audience/origin binding and confused-deputy reasoning.

**Felendler, Gandhi, Habler, Elovici & Shabtai — From Tool Orchestration to Code Execution** [`agentauth2026mcpdesign`, arXiv:2602.15945, 2026]. Shows how MCP design choices (how tool outputs and descriptions flow into the model and the host) escalate from orchestration to arbitrary code execution. Connects design-level trust assumptions to concrete poisoning and injection outcomes.

**Yang et al. — MCPSecBench** [`agentauth2025mcpsecbench`, arXiv:2508.13220, 2025]. A systematic security benchmark and playground for MCP, exercising injection, poisoning, and access-control scenarios against real server implementations. Valuable as an evaluation harness; less a source of authorization theory.

### Theme C — On-behalf-of delegation, `act`/`azp` claims, delegated-authorization protocols

**Goswami — Agentic JWT: A Secure Delegation Protocol** [`agentauth2025agenticjwt`, arXiv:2509.13597, 2025]. Proposes a dual-faceted token that cryptographically binds each agent action to a verifiable user intent and optionally to a workflow step, so that stochastic reasoning or prompt injection cannot silently expand privilege. The clearest research articulation of on-behalf-of binding plus per-action scope narrowing in a token.

**Prakash — AIP: Agent Identity Protocol for Verifiable Delegation Across MCP and A2A** [`agentauth2026aip`, arXiv:2603.24775, 2026]. Defines an identity and delegation protocol that carries verifiable delegation chains across both MCP and A2A boundaries. Addresses OBO and agent identity together; cross-protocol delegation is its distinctive contribution.

**Muruaga — Bounded Agents: Delegation Security for Multi-Agent AI Systems** [`agentauth2026boundedagents`, arXiv:2608.15888, 2026]. Formalizes bounded delegation so a sub-agent's authority cannot exceed what was explicitly delegated, with code and data released. Strong on scope-minimizing delegation in multi-agent chains.

**Tong, Dai & Guo — AID-Guard: Stateful Authorization for Delegated Agent Effects** [`agentauth2026aidguard`, arXiv:2608.21159, 2026]. Adds stateful authorization that tracks the cumulative *effects* of delegated actions rather than checking calls independently. Targets the gap where per-call checks miss aggregate privilege escalation.

**Yu, Geng, Zeng & Knottenbelt — SUDP: Secret-Use Delegation Protocol** [`agentauth2026sudp`, arXiv:2604.24920, 2026]. A protocol for delegating the *use* of secrets to agents without exposing the secrets themselves, with scoped, auditable use. Combines delegation with least-privilege secret handling.

**Choi, Jeong, Choi & Lee — ResidualAuth** [`agentauth2026residualauth`, arXiv:2609.08062, 2026]. Asks which authorization state a language agent must preserve so that revocation of a delegation actually takes effect mid-task. One of the few works to treat revocation and residual authority rigorously.

**Dantuluri & Sundi — Delegation Without Trust** [`agentauth2026delegationwithouttrust`, arXiv:2609.00267, 2026]. An empirical gap analysis of identity, authorization, and runtime governance in multi-agent LLM systems, cataloguing where current stacks assume trust they have not established. Explicitly frames the identity/authz/governance triad we build on.

### Theme D — Runtime / pre-action authorization and privilege (scope) control

**Shi, He, Wang, Li, Wu, Guo & Song — Progent: Securing AI Agents with Privilege Control** [`agentauth2025progent`, arXiv:2504.11703, 2025]. A programmable privilege-control framework enforcing least privilege via symbolic rules over tool names and arguments, with *monotonic confinement*: the effective action space can only narrow without approval, blocking silent escalation even under adversarial input. The strongest scope-minimization result and a widely-cited reference point.

**Uchibeke — Before the Tool Call: Deterministic Pre-Action Authorization** [`agentauth2026preaction`, arXiv:2603.20953, 2026]. Argues for a deterministic authorization check *before* each tool call, moving the decision out of the (stochastic) model. Directly motivates the static-vs-runtime line our SoK draws.

**Zhu & Wang — Runtime Authorization for Resources Acquired by AI Agents** [`agentauth2026runtimeauthz`, arXiv:2609.14744, 2026]. Formalizes authorization over resources an agent acquires at runtime, with algorithms for granting and checking access as the agent's resource set evolves. Explicitly adopts standard OAuth terminology for the agent setting.

**Kodathala — aiAuthZ: Off-Host, Identity-Bound Authorization** [`agentauth2026aiauthz`, arXiv:2607.05518, 2026, tech report]. Places authorization off-host and binds it to agent identity so the enforcement point is outside the manipulable agent runtime. Shares the "trusted boundary" intuition with Progent and the out-of-band work below.

**Millstone, Akidau, Brüderl & Pekker — If Agents Were Angels** [`agentauth2026angels`, arXiv:2608.27646, 2026]. Enforces policy out-of-band at a trusted tool boundary, on the premise that the agent model must be treated as fully untrusted. A clean statement of boundary-based enforcement that our enforcement-line taxonomy can classify.

**Ji et al. — Taming Privilege Escalation in LLM Agents: A Mandatory Access Control Framework** [`agentauth2026mac`, arXiv:2601.11893, 2026]. Applies MAC (labels and a reference monitor) to LLM-agent tool use to contain privilege escalation across diverse attack paths. Brings a classical OS-security construct to agent authorization.

**Li, Li, Ma, Xu, Zhang & Cheng — We Urgently Need Privilege Management in MCP** [`agentauth2025privmgmtmcp`, arXiv:2507.06250, 2025]. Measures API usage across real MCP ecosystems and quantifies systemic over-privilege. Provides the empirical case that scope minimization is unmet in practice.

**Wu, Roesner, Kohno, Zhang & Iqbal — IsolateGPT** [`agentauth2024isolategpt`, arXiv:2403.04960, NDSS 2025]. An execution-isolation architecture (hub-and-spoke) that confines each tool/app in its own isolated environment with mediated cross-spoke interaction. Peer-reviewed; an early and influential least-privilege/isolation design for agentic systems.

**Syros, Suri, Ginesin, Nita-Rotaru & Oprea — SAGA: A Security Architecture for Governing AI Agentic Systems** [`agentauth2025saga`, arXiv:2504.21034, 2025]. A full governance architecture covering agent registration/identity, user oversight, and access tokens that gate inter-agent interaction. One of the most complete end-to-end treatments spanning identity, delegation, and runtime control.

### Theme E — Workload / agent identity (WIMSE, SPIFFE/SPIRE, non-human identity)

**Otsuka, Toyoda & Leung — AI Identity: Standards, Gaps, and Research Directions** [`agentauth2026aiidentity`, arXiv:2604.23280, 2026]. Surveys the identity-standards landscape for AI agents (OAuth/OIDC, SPIFFE/WIMSE, emerging agent-identity drafts) and maps the gaps. The best single orientation to where workload/agent identity standards stand and fall short.

**Sharma — Ethical Hyper-Velocity (EHV): Hardware-Rooted Zero-Trust Runtime Enforcement** [`agentauth2026ehv`, arXiv:2605.17909, 2026]. Integrates SPIFFE/SPIRE to issue ephemeral, session-bound X.509 SVID credentials to each agent workload under a zero-trust model. A concrete application of workload identity to short-lived agent instances.

**Prakash — LDP: An Identity-Aware Protocol for Multi-Agent LLM Systems** [`agentauth2026ldp`, arXiv:2603.08852, 2026]. Makes agent identity a first-class, verifiable element of inter-agent messaging to defend against spoofing in multi-agent settings. Pairs with the same author's AIP delegation work.

### Theme F — Prompt-injection-driven privilege escalation and by-design defenses

**Greshake, Abdelnabi, Mishra, Endres, Holz & Fritz — Not What You've Signed Up For** [`agentauth2023indirectpi`, arXiv:2302.12173, ACM AISec 2023]. The seminal indirect-prompt-injection paper: adversaries plant instructions in data the model later retrieves, compromising real LLM-integrated applications. Peer-reviewed; establishes the threat model that makes runtime authorization necessary.

**Debenedetti et al. — Defeating Prompt Injections by Design (CaMeL)** [`agentauth2025camel`, arXiv:2503.18813, IEEE SaTML 2026]. Extracts control and data flow from the trusted query so untrusted data can never alter program flow, giving provable guarantees with modest utility loss on AgentDojo. Peer-reviewed; the leading by-design (rather than detection-based) defense and a key precedent for static enforcement.

**Debenedetti, Zhang, Balunović, Beurer-Kellner, Fischer & Tramèr — AgentDojo** [`agentauth2024agentdojo`, arXiv:2406.13352, NeurIPS 2024 D&B]. A dynamic benchmark of 97 tasks and 629 security cases measuring utility and attack success for tool-using agents under prompt injection. Peer-reviewed; the standard evaluation substrate for injection defenses.

**He et al. — When Context Gets Root: Privilege Escalation in LLM Harnesses** [`agentauth2026contextroot`, arXiv:2608.27299, 2026]. Shows how injected context can escalate to effectively root-level control of an agent harness, bridging prompt injection and OS-style privilege escalation. Motivates treating the harness, not just the model, as part of the attack surface.

**Wang et al. — The Landscape of Prompt Injection Threats in LLM Agents** [`agentauth2026pitaxonomy`, arXiv:2602.10453, 2026]. A taxonomy-to-analysis survey of prompt-injection attack vectors and defenses specific to agents. A useful map of the injection sub-field that our SoK can fold into its privilege-escalation category.

### Theme G — Tool poisoning (malicious or concealed tool descriptions/metadata)

**Wang et al. — MCPTox** [`agentauth2025mcptox`, arXiv:2508.14925, 2025]. The first large-scale empirical benchmark of tool-poisoning attacks, with adversarial variants of 353 real tools from 45 live MCP servers; average attack success 36.5% across 20 models, exceeding 60% for some. The headline measurement motivating pre-deployment description vetting.

**Ye, Zhang, Jia & Hu — TRUSTDESC** [`agentauth2026trustdesc`, arXiv:2604.07536, 2026]. Defends against tool poisoning by generating trusted tool descriptions, reducing reliance on attacker-controlled metadata. A representative mitigation keyed to the description surface.

**Rashidi — Unicode TAG-Block Concealment of Tool-Metadata Payloads** [`agentauth2026unicodetag`, arXiv:2607.05744, 2026]. Demonstrates concealment of malicious tool-metadata payloads using Unicode TAG blocks, exposing an approval-view fidelity gap across three independent MCP server implementations. Shows that human approval UIs can be bypassed by invisible metadata.

**Zhang et al. — No-Box Vulnerability Analysis** [`agentauth2026noboxmcp`, arXiv:2609.10854, 2026]. Detects indirect-prompt-injection vulnerabilities in MCP servers from tool descriptions alone ("no-box"), enabling scanning without server access. Directly relevant to pre-deployment, description-only testability.

**Maloyan & Namiot — Prompt Injection Attacks on Agentic Coding Assistants** [`agentauth2026codingassistants`, arXiv:2601.17548, 2026]. Systematically analyzes injection and poisoning vulnerabilities across skills, tools, and protocol ecosystems in agentic coding assistants. Links tool poisoning to concrete developer-facing exploits.

### Theme H — Multi-agent / interoperability protocol security (A2A + MCP)

**Ehtesham, Singh, Gupta & Kumar — A Survey of Agent Interoperability Protocols** [`agentauth2025interopsurvey`, arXiv:2505.02279, 2025]. Compares MCP, ACP, A2A, and ANP, including their authentication and trust assumptions. The reference map of the protocol landscape our SoK sits in.

**Li & Xie — From Glue-Code to Protocols** [`agentauth2025gluecode`, arXiv:2505.03864, 2025]. A critical analysis of A2A + MCP integration for scalable agent systems, surfacing trust-boundary and delegation issues at protocol seams. Useful for the confused-deputy and cross-protocol delegation discussion.

**Kong et al. — A Survey of LLM-Driven AI Agent Communication** [`agentauth2025commsurvey`, arXiv:2506.19676, 2025]. A 48-page survey of agent communication protocols, security risks, and defenses across the user-agent, agent-agent, and agent-environment layers. One of the broadest security surveys; thin on formal authorization properties.

**Schroeder de Witt et al. — Open Challenges in Multi-Agent Security** [`agentauth2025multiagentchallenges`, arXiv:2505.02077, 2025]. A 24-author agenda-setting paper framing security for systems of interacting agents, including identity, delegation, and the need to treat the model as untrusted. Strong framing, deliberately non-prescriptive.

**Yang, Xu, Liu, Fendley, Hong, Li & Cao — SoK: When Safe Agents Fail Together** [`agentauth2026sokmultiagent`, arXiv:2609.00595, 2026]. A systematization of multi-agent LLM security organizing attacks and defenses end-to-end (configuration, interfaces, attack paths, defenses, evaluation). The closest recent SoK in spirit; scoped to multi-agent systems rather than the authorization/token layer.

**Louck, Stulman & Dvir — Security Analysis of Agentic AI Communication Protocols** [`agentauth2025protocolcompare`, arXiv:2511.03841, 2025]. A comparative security evaluation of agentic communication protocols against a common threat model. Provides cross-protocol grounding for our normative-source mapping.

---

## 2. Comparison matrix

Coverage of each agent-authorization threat category per work. **C** = covered (a central contribution), **P** = partial (addressed but not a focus), **—** = not addressed.

| Work | AudBind | OBO | ScopeMin | ConfDep | WID | PromptInj | ToolPoison |
|---|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| MCP Landscape (Hou+ 2025) | P | P | P | P | P | C | C |
| SoK MCP (Gaire+ 2025) | P | P | P | P | P | C | C |
| MCPXKIT (Guo+ 2025, TDSC) | P | — | P | P | — | C | C |
| MCP-38 taxonomy (Shen+ 2026) | P | P | P | C | P | C | C |
| MCP-DPT (Rostamzadeh+ 2026) | P | P | P | C | P | C | C |
| Enterprise MCP (Narajala+ 2025) | C | C | C | C | P | C | C |
| Breaking the Protocol (Maloyan+ 2026) | P | — | — | P | P | C | C |
| MCP Design Choices (Felendler+ 2026) | — | — | P | P | — | C | C |
| MCPSecBench (Yang+ 2025) | P | — | P | P | P | C | C |
| Agentic JWT (Goswami 2025) | C | C | C | P | C | P | — |
| AIP (Prakash 2026) | P | C | P | P | C | — | — |
| Bounded Agents (Muruaga 2026) | P | C | C | P | P | P | — |
| AID-Guard (Tong+ 2026) | P | C | C | P | P | P | — |
| SUDP (Yu+ 2026) | P | C | C | — | P | P | — |
| ResidualAuth (Choi+ 2026) | P | C | C | P | P | — | — |
| Delegation Without Trust (Dantuluri+ 2026) | P | C | P | P | C | P | — |
| Progent (Shi+ 2025) | — | — | C | — | — | C | P |
| Pre-Action Authorization (Uchibeke 2026) | P | P | C | P | P | C | — |
| Runtime Authorization (Zhu+ 2026) | P | C | C | P | P | P | — |
| aiAuthZ (Kodathala 2026) | C | P | P | P | C | — | — |
| If Agents Were Angels (Millstone+ 2026) | P | P | C | P | P | C | P |
| MAC framework (Ji+ 2026) | — | P | C | P | — | C | P |
| Privilege Mgmt in MCP (Li+ 2025) | — | — | C | P | — | P | P |
| IsolateGPT (Wu+ 2025, NDSS) | — | — | C | P | P | C | P |
| SAGA (Syros+ 2025) | P | C | C | P | C | P | — |
| AI Identity (Otsuka+ 2026) | P | C | P | P | C | — | — |
| EHV / SPIFFE (Sharma 2026) | P | P | P | — | C | P | — |
| LDP (Prakash 2026) | P | C | P | — | C | — | — |
| Indirect Prompt Injection (Greshake+ 2023, AISec) | — | — | — | — | — | C | P |
| CaMeL (Debenedetti+ 2026, SaTML) | — | — | C | P | — | C | P |
| AgentDojo (Debenedetti+ 2024, NeurIPS) | — | — | P | — | — | C | P |
| When Context Gets Root (He+ 2026) | — | — | C | P | — | C | P |
| PI Taxonomy (Wang+ 2026) | — | — | — | — | — | C | P |
| MCPTox (Wang+ 2025) | — | — | P | — | — | C | C |
| TRUSTDESC (Ye+ 2026) | — | — | — | — | — | C | C |
| Unicode TAG Concealment (Rashidi 2026) | — | — | — | — | — | C | C |
| No-Box MCP (Zhang+ 2026) | — | — | — | — | — | C | C |
| Coding Assistants (Maloyan+ 2026) | — | — | P | P | — | C | C |
| Interop Survey (Ehtesham+ 2025) | P | P | P | P | P | P | P |
| Glue-Code to Protocols (Li+ 2025) | P | P | P | P | P | P | P |
| Comm Survey (Kong+ 2025) | P | P | P | P | P | C | C |
| Multi-Agent Challenges (de Witt+ 2025) | — | P | P | P | P | C | P |
| SoK Multi-Agent (Yang+ 2026) | — | P | P | P | P | C | P |
| Protocol Comparison (Louck+ 2025) | P | P | P | P | P | P | P |

**Readable gaps.** Two columns are visibly thin. **AudBind** (audience binding / token passthrough) is a central contribution in essentially one research work (Agentic JWT) and the enterprise framework (Narajala & Habler); elsewhere it is partial or absent, because it is treated as a *spec* obligation (RFC 8707, the MCP passthrough prohibition) rather than something the literature tests or enforces. **ConfDep** (confused deputy + dynamic client registration) appears as a named taxonomy entry (MCP-38, MCP-DPT, Enterprise MCP) but almost never as a verified or enforced property. Conversely **PromptInj** and **ToolPoison** are densely covered. The field has thoroughly studied the model-level attacks but under-formalized the token/consent-level authorization guarantees — exactly the seam our SoK occupies.

---

## 3. Where our SoK differs

The surveyed literature splits cleanly into (a) *threat taxonomies* (MCP Landscape, SoK MCP, MCP-38, MCP-DPT, the multi-agent SoKs and surveys) and (b) *point mechanisms* (Progent, Agentic JWT, AIP, SAGA, aiAuthZ, CaMeL, the delegation protocols). None organizes the agent-authorization problem along the three axes our SoK claims:

1. **Threats sorted by pre-deployment testability.** Existing taxonomies sort by attacker type (MCP Landscape), by defense placement (MCP-DPT), or by lifecycle phase, but none sorts threats by *whether they can be caught before deployment by contract/static testing of a server*. This matters because a large subset of authorization threats — audience binding, token passthrough, scope minimization, over-broad dynamic client registration — are decidable from a token, a JWKS document, and a declared tool contract *without running the agent*. No-Box MCP (Zhang+ 2026) hints at this for injection (description-only detection) but does not generalize it to the authorization layer. Our SoK makes testability the primary sort key: statically testable (contract-verifiable) vs. irreducibly runtime.

2. **Threats mapped to their normative source.** Practitioners cannot currently trace a defense to the authority that mandates it. Our SoK maps each threat to the specific normative clause it violates — RFC 8707 (resource indicators / audience), RFC 8693 (token exchange for OBO), RFC 7591 (dynamic client registration), RFC 9700 (OAuth 2.0 security BCP), RFC 8725 (JWT BCP), the MCP authorization spec's passthrough prohibition, OWASP LLM/Agentic Top-10, and CWE-441 (confused deputy). The surveyed works cite these specs in passing but do not systematically bind threat → normative source → test.

3. **The static-vs-runtime enforcement line.** The mechanism papers scatter across a line they never name: *static* enforcement at the trust boundary (CaMeL's control/data-flow extraction, trusted tool-description generation, capability attestation, contract checks on tokens/JWKS) versus *runtime* enforcement during execution (Progent's monotonic confinement, pre-action authorization, MAC reference monitors, out-of-band policy, stateful/residual authorization). Our SoK draws this line explicitly and argues which authorization properties are decidable statically (audience/scope/passthrough, verifiable against a contract — the Keyway thesis) and which are irreducibly runtime (prompt-injection-driven escalation, request-time confused-deputy). This reframes several "runtime-only" defenses as compensating controls for threats that a pre-deployment contract check could have excluded.

In short: the literature has richly characterized *what* can go wrong (especially injection and tool poisoning) and has proposed *individual* delegation/privilege mechanisms, but it has not systematized agent authorization by **testability**, by **normative provenance**, or by the **static/runtime enforcement boundary**. That three-way systematization — and the claim that audience binding, scope minimization, and passthrough are contract-verifiable *before* deployment — is the gap our SoK fills.
