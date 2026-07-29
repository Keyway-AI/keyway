// Mirrors the Go types in pkg/apitypes and internal/model (PRD §4, §12).
// Keep in sync with the backend contract.

export type Severity = "critical" | "high" | "medium" | "low" | "info";
export type ChangeClass = "widened" | "narrowed" | "neutral" | "unknown";
export type ConsumerKind = "service" | "gateway_route" | "edge_function" | "client";
export type EdgeState = "verified" | "divergent" | "unverified";

export interface Expectations {
  issuers: string[];
  audiences: string[];
  algorithms: string[];
  required_claims: string[];
  clock_skew_sec: number;
}

export interface JWKSBehavior {
  jwks_uri?: string;
  cache_ttl_sec?: number | null;
  refresh_interval_sec?: number | null;
  refreshes_on_unknown_kid?: boolean | null;
  last_observed_refresh?: string | null;
  source: "config" | "library_default" | "observed" | "probed";
}

export interface LibraryInfo {
  name: string;
  version: string;
  lang: string;
}

export interface Endpoint {
  url: string;
  method: string;
  safe_probe_path: string;
}

export interface ProvenanceRecord {
  source: string;
  locator: string;
  observed_at: string;
  confidence: number;
}

export interface Consumer {
  id: string;
  stable_id: string;
  kind: ConsumerKind;
  name: string;
  namespace?: string;
  owner_team?: string;
  endpoints: Endpoint[];
  expects: Expectations;
  jwks_behavior: JWKSBehavior;
  library?: LibraryInfo | null;
  provenance?: Record<string, ProvenanceRecord[]>;
  confidence: Record<string, number>;
  probeable: boolean;
}

export type KeyStatus = "announced" | "active" | "retiring" | "retired";

export interface Key {
  kid: string;
  alg: string;
  use: string;
  status: KeyStatus;
  public_key_pem?: string;
  first_seen_in_jwks?: string;
  in_signing_use_since?: string | null;
  retired_at?: string | null;
}

export interface IssuerInfo {
  id?: string;
  name: string;
  type?: string;
  issuer_url?: string;
  jwks_uri?: string;
  keys?: Key[] | null;
}

export interface CanaryStatus {
  issuer: string;
  keys: Key[];
  announced_kid: string;
}

export interface ProbeResult {
  id: string;
  probe_id: string;
  consumer_id: string;
  endpoint_url: string;
  status_code: number;
  latency_ms: number;
  passed: boolean;
  raw_response: string;
  run_at: string;
}

export interface Attribution {
  kind: string; // commit | pr | deploy | idp_audit | unattributed
  ref: string;
  actor?: string;
  team?: string;
  timestamp: string;
  confidence: number;
}

export interface ChangeEvent {
  id: string;
  from_version: string;
  to_version: string;
  consumer_id: string;
  field: string;
  old_value: unknown;
  new_value: unknown;
  class: ChangeClass;
  severity: Severity;
  confidence: number;
  evidence: string[];
  attribution?: Attribution | null;
  detected_at: string;
}

export interface SnapshotResponse {
  version_id: string;
  hash: string;
  is_baseline: boolean;
}

export interface CoverageResponse {
  total: number;
  resolved: number;
  low_confidence: number;
  unresolved: number;
}

export interface HealthResponse {
  status: string;
  version: string;
  time: string;
}

export type BlastVerdict = "will_break" | "ready" | "unknown";

export interface AffectedConsumer {
  consumer: Consumer;
  verdict: BlastVerdict;
  reason: string;
  evidence: string[];
  confidence: number;
}

export interface BlastRadiusResult {
  proposal: {
    kind: string;
    issuer_id: string;
    kid?: string;
    claim_name?: string;
    new_issuer_url?: string;
    algorithm?: string;
  };
  affected: AffectedConsumer[];
  unknown: Consumer[];
  recommended_grace_period_seconds: number;
  grace_basis: string;
  generated_at: string;
}

/* ── Threat coverage (keyway threats coverage) ────────────────────────── */

export interface CoverageDomain {
  domain: string;
  covered: number;
  total: number;
  percent: number;
}

export interface CoverageCategory {
  category: string;
  covered: number;
  total: number;
}

export interface CoverageThreat {
  id: string;
  domain: string;
  category: string;
  severity: Severity;
  title: string;
  invariant: string;
  sources: { ref: string; url: string }[];
  detectors: string[]; // e.g. ["analyzer:aud_mismatch"]; empty = gap
}

export interface ThreatCoverage {
  total: number;
  covered: number;
  gaps: number;
  percent: number;
  domains: CoverageDomain[];
  categories: CoverageCategory[];
  threats: CoverageThreat[];
}

/* ── Agent-token inspection (keyway agent inspect) ────────────────────── */

export interface AgentInspectRequest {
  token: string;
  audience?: string;
  require_delegation?: boolean;
  max_lifetime_seconds?: number;
  allowed_scopes?: string[];
}

export interface AgentFinding {
  threat_id: string;
  severity: Severity;
  message: string;
}

export interface AgentInspectResult {
  findings: AgentFinding[];
  count: number;
}
