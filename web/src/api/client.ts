import type {
  ChangeEvent,
  Consumer,
  CoverageResponse,
  HealthResponse,
  SnapshotResponse,
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
 * Many endpoints return 501 until their milestone lands (see PROGRESS.md).
 * `withMock` transparently falls back to sample data so the dashboard is fully
 * navigable during development. Set localStorage `keyway.live=1` to disable.
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

  latestSnapshot: () =>
    withMock<SnapshotResponse>(() => request("/v1/snapshots/latest"), mock.latestSnapshot),

  createSnapshot: () =>
    request<SnapshotResponse>("/v1/snapshots", { method: "POST" }),

  consumers: () =>
    withMock<Consumer[]>(
      async () => (await request<{ consumers: Consumer[] }>("/v1/consumers")).consumers,
      mock.consumers,
    ),

  changes: (params?: { since?: string; severity?: string; class?: string }) =>
    withMock<ChangeEvent[]>(async () => {
      const q = new URLSearchParams(params as Record<string, string>).toString();
      const r = await request<{ changes: ChangeEvent[] }>(`/v1/changes${q ? `?${q}` : ""}`);
      return r.changes;
    }, mock.changes),
};
