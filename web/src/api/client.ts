import type {
  AgentInspectRequest,
  AgentInspectResult,
  BlastRadiusResult,
  CanaryStatus,
  ChangeEvent,
  Consumer,
  CoverageResponse,
  HealthResponse,
  IssuerInfo,
  Key,
  ProbeResult,
  SnapshotResponse,
  ThreatCoverage,
} from "./types";
import * as mock from "./mock";

const TOKEN_KEY = "keyway.token";

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

/** Thrown for non-2xx responses; `notImplemented` flags the 501 milestone stubs. */
export class ApiError extends Error {
  status: number;
  notImplemented: boolean;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.notImplemented = status === 501;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${getToken()}`,
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      /* ignore parse errors */
    }
    throw new ApiError(res.status, msg);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

/**
 * `withMock` transparently falls back to built-in sample data whenever the live
 * API is unavailable — no backend running, a missing/invalid API token, or an
 * endpoint that errors — so the dashboard is fully explorable out of the box.
 * Set localStorage `keyway.live=1` to force the live API and surface errors.
 */
async function withMock<T>(live: () => Promise<T>, fallback: () => T): Promise<T> {
  const forceLive = localStorage.getItem("keyway.live") === "1";
  try {
    return await live();
  } catch (err) {
    // In non-live mode, fall back to sample data for milestone stubs (501),
    // missing auth (401), and network failures (no backend running).
    if (!forceLive) {
      return fallback();
    }
    throw err;
  }
}

export const api = {
  health: () => request<HealthResponse>("/v1/health"),

  coverage: () =>
    withMock<CoverageResponse>(() => request("/v1/coverage"), mock.coverage),

  // GET /v1/snapshots/latest returns the full contract version; map to the
  // compact SnapshotResponse the dashboard shows.
  latestSnapshot: () =>
    withMock<SnapshotResponse>(async () => {
      const v = await request<{ id: string; hash: string; is_baseline: boolean }>("/v1/snapshots/latest");
      return { version_id: v.id, hash: v.hash, is_baseline: v.is_baseline };
    }, mock.latestSnapshot),

  createSnapshot: () =>
    withMock<SnapshotResponse>(
      () => request<SnapshotResponse>("/v1/snapshots", { method: "POST" }),
      mock.createSnapshot,
    ),

  consumers: () =>
    withMock<Consumer[]>(
      async () => (await request<{ consumers: Consumer[] }>("/v1/consumers")).consumers ?? [],
      mock.consumers,
    ),

  consumerProbes: (stableId: string) =>
    withMock<ProbeResult[]>(
      async () =>
        (
          await request<{ results: ProbeResult[] }>(
            `/v1/consumers/${encodeURIComponent(stableId)}/probes`,
          )
        ).results ?? [],
      () => mock.consumerProbes(stableId),
    ),

  changes: (params?: { since?: string; severity?: string; class?: string }) =>
    withMock<ChangeEvent[]>(async () => {
      const q = new URLSearchParams(params as Record<string, string>).toString();
      const r = await request<{ changes: ChangeEvent[] }>(`/v1/changes${q ? `?${q}` : ""}`);
      return r.changes ?? [];
    }, mock.changes),

  // POST /v1/blast-radius — live resolution. Callers fall back to the local
  // resolver when the API is unavailable.
  blastRadius: (proposal: BlastRadiusResult["proposal"]) =>
    withMock<BlastRadiusResult>(
      () =>
        request<BlastRadiusResult>("/v1/blast-radius", {
          method: "POST",
          body: JSON.stringify({ proposal }),
        }),
      () => mock.blastRadius(proposal),
    ),

  issuers: () =>
    withMock<IssuerInfo[]>(
      async () => (await request<{ issuers: IssuerInfo[] }>("/v1/issuers")).issuers ?? [],
      mock.issuers,
    ),

  canaryStatus: (issuerId: string) =>
    withMock<CanaryStatus>(
      () => request<CanaryStatus>(`/v1/canary/status?issuer_id=${encodeURIComponent(issuerId)}`),
      () => mock.canaryStatus(issuerId),
    ),

  canaryAnnounce: (issuerId: string, alg: string) =>
    withMock<Key>(
      () =>
        request<Key>("/v1/canary/announce", {
          method: "POST",
          body: JSON.stringify({ issuer_id: issuerId, alg }),
        }),
      () => mock.canaryAnnounce(issuerId, alg),
    ),

  canaryPromote: (issuerId: string, kid: string) =>
    withMock<{ promoted: string }>(
      () =>
        request<{ promoted: string }>("/v1/canary/promote", {
          method: "POST",
          body: JSON.stringify({ issuer_id: issuerId, kid }),
        }),
      () => mock.canaryPromote(issuerId, kid),
    ),

  runProbes: (consumerIds?: string[]) =>
    withMock<{ run_id: string; results: ProbeResult[] }>(
      () =>
        request<{ run_id: string; results: ProbeResult[] }>("/v1/probes/run", {
          method: "POST",
          body: JSON.stringify({ consumer_ids: consumerIds ?? [] }),
        }),
      () => mock.runProbes(consumerIds),
    ),

  threatCoverage: () =>
    withMock<ThreatCoverage>(() => request("/v1/threats/coverage"), mock.threatCoverage),

  agentInspect: (req: AgentInspectRequest) =>
    withMock<AgentInspectResult>(
      () => request("/v1/agent/inspect", { method: "POST", body: JSON.stringify(req) }),
      () => mock.agentInspect(req),
    ),
};
