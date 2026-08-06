/**
 * Client for the Keyway Cloud API (`cmd/keyway-cloud`, package `cloud`) — the
 * multi-tenant hosted layer. It is a *separate* backend from the self-hosted
 * `keyway serve` API the rest of the app talks to: cookie-authenticated, on its
 * own origin. Point the frontend at it with `VITE_CLOUD_API_URL`; in local dev it
 * defaults to the `make cloud` port (:8090).
 *
 * Every request sends `credentials: "include"` so the session cookie flows on the
 * cross-site calls from the Vercel frontend to the API host.
 */

const RAW_BASE = (import.meta.env.VITE_CLOUD_API_URL as string | undefined) ?? "";
// Dev fallback: talk to `make cloud` on localhost. In production the env var must
// be set to the deployed API origin.
const DEV_FALLBACK = "http://localhost:8090";
export const CLOUD_BASE =
  RAW_BASE || (import.meta.env.DEV ? DEV_FALLBACK : "");

/* ── Wire types (mirror the Go JSON in cloud/*.go) ──────────────────────── */

export interface CloudConfig {
  github_login: boolean;
  dev_login: boolean;
}

export interface CloudUser {
  id: string;
  login: string;
  name?: string;
  avatar_url?: string;
  email?: string;
  created_at: string;
}

export type SourceKind = "upload" | "github";

export interface Source {
  kind: SourceKind;
  repo?: string;
  ref?: string;
  path?: string;
}

export interface Project {
  id: string;
  owner_id: string;
  name: string;
  source: Source;
  created_at: string;
  latest_analysis_id?: string;
}

export interface AnalysisSummary {
  id: string;
  created_at: string;
  trigger_kind: string;
  trigger_ref?: string;
  hash: string;
  is_baseline: boolean;
  consumer_count: number;
  issuer_count: number;
  change_count: number;
}

/** Contract + drift shapes reuse the engine's model types (loosely typed here). */
export interface Analysis extends AnalysisSummary {
  project_id: string;
  version: ContractVersion;
  changes: ChangeEvent[];
}

export interface ContractVersion {
  id: string;
  hash: string;
  created_at: string;
  is_baseline: boolean;
  issuers?: Issuer[];
  consumers?: ContractConsumer[];
  edges?: unknown[];
}

export interface Issuer {
  url?: string;
  name?: string;
  algorithms?: string[];
  [k: string]: unknown;
}

export interface ContractConsumer {
  name?: string;
  service?: string;
  kind?: string;
  expects?: {
    issuers?: string[];
    audiences?: string[];
    algorithms?: string[];
    required_claims?: string[];
  };
  [k: string]: unknown;
}

/** Drift event emitted by `internal/diff` (see model.ChangeEvent). */
export interface ChangeEvent {
  id?: string;
  severity?: string;
  class?: string; // widened | narrowed | added | removed | ...
  field?: string; // e.g. "expects.audiences"
  consumer_id?: string;
  old_value?: unknown;
  new_value?: unknown;
  [k: string]: unknown;
}

/* ── Error ──────────────────────────────────────────────────────────────── */

export class CloudError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "CloudError";
  }
}

/** Distinguishes "not signed in" from other failures so the UI can route to sign-in. */
export function isUnauthorized(err: unknown): boolean {
  return err instanceof CloudError && err.status === 401;
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(CLOUD_BASE + path, {
      credentials: "include",
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers ?? {}),
      },
    });
  } catch {
    throw new CloudError(0, "Can't reach the Keyway Cloud API. Is it running?");
  }
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      /* ignore */
    }
    throw new CloudError(res.status, msg);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

/* ── Endpoints ──────────────────────────────────────────────────────────── */

export const cloud = {
  base: CLOUD_BASE,

  config: () => req<CloudConfig>("/v1/config"),
  me: () => req<CloudUser>("/v1/me"),

  /** GitHub OAuth is a full-page redirect (not fetch) so cookies are set first-party. */
  githubLoginUrl: () => CLOUD_BASE + "/v1/auth/github/login",

  devLogin: () => req<CloudUser>("/v1/auth/dev-login", { method: "POST" }),
  logout: () => req<void>("/v1/auth/logout", { method: "POST" }),

  listProjects: () =>
    req<{ projects: Project[] }>("/v1/projects").then((r) => r.projects ?? []),

  createProject: (name: string, source: Source) =>
    req<Project>("/v1/projects", {
      method: "POST",
      body: JSON.stringify({ name, source }),
    }),

  /** Returns the project plus its latest analysis (if any) in one round-trip. */
  getProject: (id: string) =>
    req<{ project: Project; latest?: Analysis }>(`/v1/projects/${encodeURIComponent(id)}`),

  deleteProject: (id: string) =>
    req<void>(`/v1/projects/${encodeURIComponent(id)}`, { method: "DELETE" }),

  /** Run the engine: `manifests` for upload sources, empty body to sync a repo. */
  analyze: (id: string, manifests?: Record<string, string>) =>
    req<Analysis>(`/v1/projects/${encodeURIComponent(id)}/analyze`, {
      method: "POST",
      body: JSON.stringify(manifests ? { manifests } : {}),
    }),

  listAnalyses: (id: string) =>
    req<{ analyses: AnalysisSummary[] }>(
      `/v1/projects/${encodeURIComponent(id)}/analyses`,
    ).then((r) => r.analyses ?? []),

  getAnalysis: (id: string) => req<Analysis>(`/v1/analyses/${encodeURIComponent(id)}`),
};
