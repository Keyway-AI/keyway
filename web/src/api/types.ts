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
  cache_ttl_sec?: number | null;
  refresh_interval_sec?: number | null;
  refreshes_on_unknown_kid?: boolean | null;
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
  confidence: Record<string, number>;
  probeable: boolean;
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
  };
  affected: AffectedConsumer[];
  unknown: Consumer[];
  recommended_grace_period_seconds: number;
  grace_basis: string;
  generated_at: string;
}
