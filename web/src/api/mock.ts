// Sample data mirroring the PRD demo so the dashboard is fully navigable before
// the backend endpoints land. Swap in live data by setting localStorage
// `keyway.live=1` (see client.ts).
import { computeBlast } from "../lib/blast";
import type {
  BlastRadiusResult,
  CanaryStatus,
  ChangeEvent,
  Consumer,
  CoverageResponse,
  IssuerInfo,
  Key,
  ProbeResult,
  SnapshotResponse,
} from "./types";

export const coverage = (): CoverageResponse => ({
  total: 47,
  resolved: 41,
  low_confidence: 3,
  unresolved: 3,
});

export const latestSnapshot = (): SnapshotResponse => ({
  version_id: "9f2c1a7e-8b3d-4f6a-9c1e-2d5b7a0e4c31",
  hash: "3b1f9c2a7e4d5f6089ab12cd34ef5678a9b0c1d2e3f405162738495a6b7c8d9e",
  is_baseline: false,
});

function consumer(partial: Partial<Consumer> & Pick<Consumer, "stable_id" | "name">): Consumer {
  return {
    id: partial.stable_id,
    kind: "service",
    endpoints: [],
    expects: {
      issuers: ["https://kc/realms/main"],
      audiences: [partial.name],
      algorithms: ["RS256"],
      required_claims: [],
      clock_skew_sec: 60,
    },
    jwks_behavior: { source: "config" },
    confidence: { overall: 1 },
    probeable: true,
    ...partial,
  };
}

export const consumers = (): Consumer[] => [
  consumer({
    stable_id: "k8s://prod/default/payments-api",
    name: "payments-api",
    owner_team: "team-payments",
    namespace: "default",
    jwks_behavior: {
      cache_ttl_sec: 172800,
      refreshes_on_unknown_kid: false,
      source: "probed",
    },
    library: { name: "MicahParks/keyfunc", version: "v1.9.0", lang: "go" },
    confidence: {
      overall: 1,
      "expects.issuers": 1,
      "expects.audiences": 1,
      "jwks_behavior.refreshes_on_unknown_kid": 0.8,
    },
    provenance: {
      "expects.issuers": [
        {
          source: "istio:RequestAuthentication/payments",
          locator: "cluster/istio/payments-ra.yaml",
          observed_at: new Date(Date.now() - 86400_000).toISOString(),
          confidence: 1,
        },
      ],
      "jwks_behavior.refreshes_on_unknown_kid": [
        {
          source: "lib:keyfunc",
          locator: "MicahParks/keyfunc v1.9.0",
          observed_at: new Date(Date.now() - 86400_000).toISOString(),
          confidence: 0.8,
        },
      ],
    },
  }),
  consumer({
    stable_id: "k8s://prod/data/legacy-reporting",
    name: "legacy-reporting",
    owner_team: "team-data",
    namespace: "data",
    jwks_behavior: { refreshes_on_unknown_kid: false, source: "library_default" },
    library: { name: "MicahParks/keyfunc", version: "v1.9.0", lang: "go" },
    confidence: { overall: 0.8 },
  }),
  consumer({
    stable_id: "route://mobile-gw/default",
    name: "mobile-gateway",
    kind: "gateway_route",
    owner_team: "team-mobile",
    jwks_behavior: { cache_ttl_sec: 3600, source: "config" },
    confidence: { overall: 1 },
  }),
  consumer({
    stable_id: "k8s://prod/default/orders-api",
    name: "orders-api",
    owner_team: "team-orders",
    namespace: "default",
    jwks_behavior: { cache_ttl_sec: 300, refreshes_on_unknown_kid: true, source: "config" },
    confidence: { overall: 1 },
  }),
  consumer({
    stable_id: "url://analytics.svc/api",
    name: "analytics",
    kind: "edge_function",
    probeable: false,
    jwks_behavior: { source: "config" },
    confidence: { overall: 0.4 },
  }),
];

export const consumerProbes = (stableId: string): ProbeResult[] => {
  // Only the probeable sample services have history; others return none.
  if (!stableId.startsWith("k8s://") && !stableId.startsWith("route://")) return [];
  const probes = [
    { probe_id: "alg_none", passed: true, status_code: 401 },
    { probe_id: "alg_confusion", passed: true, status_code: 401 },
    { probe_id: "expired", passed: true, status_code: 401 },
    { probe_id: "wrong_audience", passed: true, status_code: 403 },
    { probe_id: "valid_token", passed: true, status_code: 200 },
  ];
  return probes.map((p, i) => ({
    id: `${stableId}-${p.probe_id}`,
    probe_id: p.probe_id,
    consumer_id: stableId,
    endpoint_url: "https://svc.internal/healthz",
    status_code: p.status_code,
    latency_ms: 12 + i * 3,
    passed: p.passed,
    raw_response: "",
    run_at: new Date(Date.now() - (i + 1) * 1800_000).toISOString(),
  }));
};

export const blastRadius = (proposal: BlastRadiusResult["proposal"]): BlastRadiusResult =>
  computeBlast(consumers(), proposal);

// --- interactive mocks (so the demo works with no backend) ---

// In-memory canary key state per issuer, mutated by announce/promote so the
// sample UI behaves like the real lifecycle.
const canaryState: Record<string, Key[]> = {
  default: [
    {
      kid: "rsa-2025-06",
      alg: "RS256",
      use: "sig",
      status: "active",
      first_seen_in_jwks: new Date(Date.now() - 40 * 86400_000).toISOString(),
      in_signing_use_since: new Date(Date.now() - 40 * 86400_000).toISOString(),
    },
  ],
};

let mockKidSeq = 100;

export const issuers = (): IssuerInfo[] => [
  {
    id: "default",
    name: "default",
    type: "generic_oidc",
    issuer_url: "https://kc/realms/main",
    jwks_uri: "https://kc/realms/main/protocol/openid-connect/certs",
    keys: canaryState["default"],
  },
];

export const canaryStatus = (issuerId: string): CanaryStatus => {
  const keys = canaryState[issuerId] ?? [];
  const announced = keys.find((k) => k.status === "announced");
  return { issuer: issuerId, keys, announced_kid: announced?.kid ?? "" };
};

export const canaryAnnounce = (issuerId: string, alg: string): Key => {
  const key: Key = {
    kid: `rsa-2026-${mockKidSeq++}`,
    alg: alg || "RS256",
    use: "sig",
    status: "announced",
    first_seen_in_jwks: new Date().toISOString(),
  };
  canaryState[issuerId] = [...(canaryState[issuerId] ?? []), key];
  return key;
};

// Promote flips the announced key to active signing and moves the prior active
// key to retiring — mirroring the real KeyStatus lifecycle so the demo advances.
export const canaryPromote = (issuerId: string, kid: string): { promoted: string } => {
  const keys = canaryState[issuerId] ?? [];
  canaryState[issuerId] = keys.map((k) => {
    if (k.kid === kid) return { ...k, status: "active", in_signing_use_since: new Date().toISOString() };
    if (k.status === "active") return { ...k, status: "retiring" };
    return k;
  });
  return { promoted: kid };
};

export const createSnapshot = (): SnapshotResponse => ({
  version_id: `v-${Date.now()}`,
  hash: Array.from({ length: 64 }, () => "0123456789abcdef"[Math.floor(Math.random() * 16)]).join(""),
  is_baseline: false,
});

export const runProbes = (consumerIds?: string[]): { run_id: string; results: ProbeResult[] } => {
  const targets = (consumers().filter((c) => c.probeable) ?? []).filter(
    (c) => !consumerIds || consumerIds.length === 0 || consumerIds.includes(c.stable_id),
  );
  const results: ProbeResult[] = [];
  const probes = [
    { probe_id: "alg_none", passed: true, status: 401 },
    { probe_id: "alg_confusion", passed: true, status: 401 },
    { probe_id: "tampered_signature", passed: true, status: 401 },
    { probe_id: "expired", passed: true, status: 401 },
    { probe_id: "wrong_audience", passed: true, status: 403 },
    { probe_id: "wrong_issuer", passed: true, status: 401 },
    { probe_id: "valid_token", passed: true, status: 200 },
  ];
  for (const c of targets) {
    for (let i = 0; i < probes.length; i++) {
      const p = probes[i];
      // legacy-reporting fails the alg_none probe (a real finding) in the demo.
      const passed = c.name === "legacy-reporting" && p.probe_id === "alg_none" ? false : p.passed;
      results.push({
        id: `${c.stable_id}-${p.probe_id}`,
        probe_id: p.probe_id,
        consumer_id: c.stable_id,
        endpoint_url: "https://svc.internal/healthz",
        status_code: passed ? p.status : 200,
        latency_ms: 8 + i * 2,
        passed,
        raw_response: "",
        run_at: new Date().toISOString(),
      });
    }
  }
  return { run_id: `run-${Date.now()}`, results };
};

export const changes = (): ChangeEvent[] => [
  {
    id: "ce-1",
    from_version: "v-baseline",
    to_version: "v-2",
    consumer_id: "k8s://prod/default/orders-api",
    field: "expects.audiences",
    old_value: ["orders-api"],
    new_value: ["orders-api", "orders-api-v2"],
    class: "widened",
    severity: "medium",
    confidence: 1,
    evidence: ["istio:RequestAuthentication/orders"],
    attribution: { kind: "commit", ref: "a1b2c3d4e5f6", actor: "alice", timestamp: "", confidence: 0.9 },
    detected_at: new Date(Date.now() - 3600_000).toISOString(),
  },
  {
    id: "ce-2",
    from_version: "v-baseline",
    to_version: "v-2",
    consumer_id: "k8s://prod/default/payments-api",
    field: "jwks_behavior.refreshes_on_unknown_kid",
    old_value: true,
    new_value: false,
    class: "narrowed",
    severity: "high",
    confidence: 0.8,
    evidence: ["lib:keyfunc v1.9.0"],
    detected_at: new Date(Date.now() - 7200_000).toISOString(),
  },
  {
    id: "ce-3",
    from_version: "v-baseline",
    to_version: "v-2",
    consumer_id: "k8s://prod/default/reports-api",
    field: "expects.required_claims",
    old_value: ["dept"],
    new_value: [],
    class: "widened",
    severity: "critical",
    confidence: 1,
    evidence: ["istio:AuthorizationPolicy/reports-dept"],
    attribution: { kind: "deploy", ref: "kubectl apply (PR #482 by bob)", actor: "bob", timestamp: "", confidence: 0.8 },
    detected_at: new Date(Date.now() - 1800_000).toISOString(),
  },
  {
    id: "ce-4",
    from_version: "v-baseline",
    to_version: "v-2",
    consumer_id: "route://mobile-gw/default",
    field: "expects.issuers",
    old_value: ["https://kc/realms/main"],
    new_value: ["https://kc/realms/main", "https://auth.partner.com/realms/ext"],
    class: "widened",
    severity: "high",
    confidence: 1,
    evidence: ["envoy:jwt_authn/mobile"],
    detected_at: new Date(Date.now() - 5400_000).toISOString(),
  },
  {
    id: "ce-5",
    from_version: "v-baseline",
    to_version: "v-2",
    consumer_id: "route://mobile-gw/default",
    field: "expects.algorithms",
    old_value: ["RS256"],
    new_value: ["RS256", "none"],
    class: "widened",
    severity: "critical",
    confidence: 1,
    evidence: ["probe:alg_none"],
    detected_at: new Date(Date.now() - 5400_000).toISOString(),
  },
  {
    id: "ce-6",
    from_version: "v-baseline",
    to_version: "v-2",
    consumer_id: "k8s://prod/default/orders-api",
    field: "jwks_behavior.cache_ttl_sec",
    old_value: 300,
    new_value: 3600,
    class: "narrowed",
    severity: "medium",
    confidence: 1,
    evidence: ["envoy:jwt_authn/orders"],
    detected_at: new Date(Date.now() - 3600_000).toISOString(),
  },
];
