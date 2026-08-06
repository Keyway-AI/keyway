import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { cloud, isUnauthorized } from "./api";
import type { Analysis, AnalysisSummary, Project } from "./api";
import { useCloudAuth } from "./CloudAuth";
import { Button, Card, Empty, Pill, StatTile, Table, Td, Th } from "../components/ui";
import { readManifestFiles } from "./manifests";

/**
 * Project detail: run the engine on the project's config and show the derived
 * contract (issuers + consumers) plus drift versus the previous run, with the
 * project's analysis history alongside.
 */
export default function CloudProject() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { refresh } = useCloudAuth();

  const [project, setProject] = useState<Project | null>(null);
  const [history, setHistory] = useState<AnalysisSummary[]>([]);
  const [current, setCurrent] = useState<Analysis | null>(null);
  const [error, setError] = useState("");
  const [notFound, setNotFound] = useState(false);
  const [busy, setBusy] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const handle = useCallback(
    (e: unknown) => {
      if (isUnauthorized(e)) return void refresh();
      const err = e as { status?: number };
      if (err.status === 404) return setNotFound(true);
      setError(e instanceof Error ? e.message : "Something went wrong");
    },
    [refresh],
  );

  const load = useCallback(async () => {
    try {
      const [{ project: p, latest }, h] = await Promise.all([
        cloud.getProject(id),
        cloud.listAnalyses(id),
      ]);
      setProject(p);
      setHistory(h);
      if (latest) setCurrent(latest);
    } catch (e) {
      handle(e);
    }
  }, [id, handle]);

  useEffect(() => {
    void load();
  }, [load]);

  async function runAnalysis(manifests?: Record<string, string>) {
    setBusy(true);
    setError("");
    try {
      const a = await cloud.analyze(id, manifests);
      setCurrent(a);
      setHistory(await cloud.listAnalyses(id));
      setProject((p) => (p ? { ...p, latest_analysis_id: a.id } : p));
    } catch (e) {
      handle(e);
    } finally {
      setBusy(false);
    }
  }

  async function onDelete() {
    if (!confirm("Delete this project and its analysis history? This cannot be undone.")) return;
    try {
      await cloud.deleteProject(id);
      navigate("/cloud");
    } catch (e) {
      handle(e);
    }
  }

  async function viewAnalysis(analysisId: string) {
    try {
      setCurrent(await cloud.getAnalysis(analysisId));
    } catch (e) {
      handle(e);
    }
  }

  if (notFound) {
    return (
      <Card>
        <Empty>
          <p className="text-body">Project not found.</p>
          <Link to="/cloud" className="mt-2 inline-block text-caption font-medium text-accent hover:underline">
            ← Back to projects
          </Link>
        </Empty>
      </Card>
    );
  }
  if (!project) {
    return <div className="h-40 animate-pulse rounded-xl border border-border bg-surface-2/60" />;
  }

  const isUpload = project.source.kind === "upload";

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <Link to="/cloud" className="text-caption font-medium text-muted transition hover:text-text">
          ← Projects
        </Link>
        <div className="mt-2 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-h1 font-semibold tracking-tight display">{project.name}</h1>
            <p className="mt-1 flex items-center gap-2 text-caption text-muted">
              <Pill className="capitalize">{project.source.kind}</Pill>
              {project.source.kind === "github"
                ? `${project.source.repo}${project.source.ref ? ` · ${project.source.ref}` : ""}`
                : "Uploaded configuration"}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" onClick={onDelete}>
              Delete
            </Button>
            {isUpload ? (
              <>
                <input
                  ref={fileInput}
                  type="file"
                  multiple
                  accept=".yaml,.yml"
                  className="hidden"
                  onChange={async (e) => {
                    if (!e.target.files) return;
                    try {
                      const m = await readManifestFiles(e.target.files);
                      void runAnalysis(m);
                    } catch (err) {
                      setError(err instanceof Error ? err.message : "Could not read files");
                    }
                  }}
                />
                <Button variant="primary" className="glow-accent" loading={busy} onClick={() => fileInput.current?.click()}>
                  Upload &amp; analyze
                </Button>
              </>
            ) : (
              <Button variant="primary" className="glow-accent" loading={busy} onClick={() => runAnalysis()}>
                Sync &amp; analyze
              </Button>
            )}
          </div>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-md border border-critical/30 bg-critical/8 px-3 py-2.5 text-caption text-critical">
          {error}
        </div>
      )}

      {!current ? (
        <Card>
          <Empty>
            <p className="text-body">No analysis yet.</p>
            <p className="mt-1 text-caption">
              {isUpload
                ? "Upload your config manifests to derive the token contract."
                : "Sync the connected repository to derive the token contract."}
            </p>
          </Empty>
        </Card>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[1fr_20rem]">
          <div className="space-y-6">
            <AnalysisView analysis={current} />
          </div>
          <HistoryPanel history={history} currentId={current.id} onSelect={viewAnalysis} />
        </div>
      )}
    </div>
  );
}

/* ── The analysis itself ────────────────────────────────────────────────── */

function AnalysisView({ analysis }: { analysis: Analysis }) {
  const consumers = analysis.version?.consumers ?? [];
  const issuers = analysis.version?.issuers ?? [];
  const changes = analysis.changes ?? [];

  return (
    <>
      <div className="grid grid-cols-3 gap-4">
        <StatTile label="Consumers" value={analysis.consumer_count} hint="services validating tokens" />
        <StatTile label="Issuers" value={analysis.issuer_count} hint="token signers in contract" />
        <StatTile
          label="Drift"
          value={analysis.change_count}
          hint={analysis.is_baseline ? "baseline — nothing to compare" : "changes vs previous"}
          accent={analysis.change_count > 0 ? "text-high" : "text-text"}
        />
      </div>

      {!analysis.is_baseline && (
        <Card title="Drift since previous analysis">
          {changes.length === 0 ? (
            <Empty>No changes — the contract held.</Empty>
          ) : (
            <ul className="divide-y divide-border/70">
              {changes.map((c, i) => (
                <li key={c.id ?? i} className="flex items-start gap-3 py-3 first:pt-0 last:pb-0">
                  <SeverityDot severity={String(c.severity ?? "info")} />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-body font-medium text-text">
                        {c.field || "field"}{" "}
                        <span className="font-normal text-muted">{c.class || "changed"}</span>
                      </span>
                      {c.severity && (
                        <span className="rounded-pill bg-surface-2 px-2 py-0.5 text-micro font-medium capitalize text-muted">
                          {String(c.severity)}
                        </span>
                      )}
                    </div>
                    <div className="mt-0.5 text-caption text-muted">
                      {fmtVal(c.old_value)} <span className="text-faint">→</span> {fmtVal(c.new_value)}
                      {c.consumer_id ? <span className="text-faint"> · {shortConsumer(String(c.consumer_id))}</span> : null}
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      )}

      <Card title={`Consumers (${consumers.length})`} bodyClassName="p-0">
        {consumers.length === 0 ? (
          <div className="p-5">
            <Empty>No token-validating consumers discovered in this config.</Empty>
          </div>
        ) : (
          <Table minWidth="min-w-[34rem]">
            <thead>
              <tr>
                <Th>Consumer</Th>
                <Th>Issuers</Th>
                <Th>Audiences</Th>
                <Th>Algorithms</Th>
              </tr>
            </thead>
            <tbody>
              {consumers.map((c, i) => {
                const ex = c.expects ?? {};
                return (
                  <tr key={i}>
                    <Td className="font-medium">{c.name || c.service || `consumer ${i + 1}`}</Td>
                    <Td className="text-muted">{join(ex.issuers)}</Td>
                    <Td className="text-muted">{join(ex.audiences)}</Td>
                    <Td className="text-muted">{join(ex.algorithms)}</Td>
                  </tr>
                );
              })}
            </tbody>
          </Table>
        )}
      </Card>

      {issuers.length > 0 && (
        <Card title={`Issuers (${issuers.length})`} bodyClassName="p-0">
          <Table minWidth="min-w-[28rem]">
            <thead>
              <tr>
                <Th>Issuer</Th>
                <Th>Algorithms</Th>
              </tr>
            </thead>
            <tbody>
              {issuers.map((s, i) => (
                <tr key={i}>
                  <Td className="font-medium">{s.name || s.url || `issuer ${i + 1}`}</Td>
                  <Td className="text-muted">{join(s.algorithms)}</Td>
                </tr>
              ))}
            </tbody>
          </Table>
        </Card>
      )}

      <p className="text-micro text-faint">
        Contract <code className="font-mono">{analysis.hash.slice(0, 12)}</code> · analyzed{" "}
        {new Date(analysis.created_at).toLocaleString()}
      </p>
    </>
  );
}

function HistoryPanel({
  history,
  currentId,
  onSelect,
}: {
  history: AnalysisSummary[];
  currentId: string;
  onSelect: (id: string) => void;
}) {
  return (
    <Card title="History" bodyClassName="p-2">
      {history.length === 0 ? (
        <div className="p-3">
          <Empty>No runs yet.</Empty>
        </div>
      ) : (
        <ul className="space-y-1">
          {history.map((h) => {
            const active = h.id === currentId;
            return (
              <li key={h.id}>
                <button
                  onClick={() => onSelect(h.id)}
                  className={`flex w-full items-center justify-between gap-2 rounded-lg px-3 py-2.5 text-left transition ${
                    active ? "bg-accent-soft" : "hover:bg-surface-2"
                  }`}
                >
                  <div className="min-w-0">
                    <div className="truncate text-caption font-medium text-text">
                      {new Date(h.created_at).toLocaleString([], {
                        month: "short",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                      })}
                    </div>
                    <div className="text-micro text-muted">
                      {h.is_baseline ? "Baseline" : `${h.change_count} change${h.change_count === 1 ? "" : "s"}`}
                      {" · "}
                      {h.consumer_count} consumer{h.consumer_count === 1 ? "" : "s"}
                    </div>
                  </div>
                  {!h.is_baseline && h.change_count > 0 && (
                    <span className="shrink-0 rounded-pill bg-high/12 px-2 py-0.5 text-micro font-semibold text-high">
                      {h.change_count}
                    </span>
                  )}
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </Card>
  );
}

/* ── bits ───────────────────────────────────────────────────────────────── */

function join(xs?: string[]): string {
  return xs && xs.length ? xs.join(", ") : "—";
}

/** Render a drift value: arrays as comma lists, empty/null as ∅. */
function fmtVal(v: unknown): string {
  if (v == null || v === "") return "∅";
  if (Array.isArray(v)) return v.length ? v.join(", ") : "∅";
  return String(v);
}

/** Trim a consumer stable id ("k8s://local/foo/httpbin") to its tail for display. */
function shortConsumer(id: string): string {
  const parts = id.split("/").filter(Boolean);
  return parts.slice(-2).join("/") || id;
}

const dotColor: Record<string, string> = {
  critical: "bg-critical",
  high: "bg-high",
  medium: "bg-medium",
  low: "bg-low",
  info: "bg-info",
};

function SeverityDot({ severity }: { severity: string }) {
  const s = severity.toLowerCase();
  return <span className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${dotColor[s] ?? "bg-info"}`} title={severity} />;
}
