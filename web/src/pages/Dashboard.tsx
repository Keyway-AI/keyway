import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Button, Card, Empty, Pill, SeverityBadge, StatTile } from "../components/ui";
import { useToast } from "../components/toast";
import { useAsync } from "../lib/useAsync";
import { relativeTime, severityColor } from "../lib/format";
import { countBySeverity, severityRank, toFinding } from "../lib/findings";

export default function Dashboard() {
  const toast = useToast();
  const [snapshotting, setSnapshotting] = useState(false);
  const coverage = useAsync(() => api.coverage());
  const snapshot = useAsync(() => api.latestSnapshot());
  const changes = useAsync(() => api.changes());
  const threats = useAsync(() => api.threatCoverage());

  async function takeSnapshot() {
    setSnapshotting(true);
    try {
      const v = await api.createSnapshot();
      toast.success(`Snapshot ${v.is_baseline ? "captured as baseline" : "captured"}`);
      snapshot.reload();
      coverage.reload();
      changes.reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Snapshot failed");
    } finally {
      setSnapshotting(false);
    }
  }

  const cov = coverage.data;
  const pct = cov && cov.total ? Math.round((cov.resolved / cov.total) * 100) : 0;

  const findings = useMemo(() => (changes.data ?? []).map(toFinding), [changes.data]);
  const counts = useMemo(() => countBySeverity(findings), [findings]);
  const topFindings = useMemo(
    () => [...findings].sort((a, b) => severityRank[a.severity] - severityRank[b.severity]).slice(0, 6),
    [findings],
  );
  const actionable = counts.critical + counts.high;

  return (
    <Page
      title="Dashboard"
      subtitle="Coverage and recent contract changes across your issuers."
      actions={
        <Button variant="primary" onClick={takeSnapshot} loading={snapshotting}>
          Take snapshot
        </Button>
      }
    >
      <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
        <StatTile label="Consumers" value={cov?.total ?? "—"} hint="derived automatically" />
        <StatTile
          label="Resolved"
          value={cov ? `${pct}%` : "—"}
          hint={cov ? `${cov.resolved} of ${cov.total}` : undefined}
          accent="text-low"
        />
        <StatTile
          label="Low confidence"
          value={cov?.low_confidence ?? "—"}
          hint="< 0.6 confidence"
          accent="text-medium"
        />
        <StatTile
          label="Unresolved"
          value={cov?.unresolved ?? "—"}
          hint="not probeable"
          accent="text-muted"
        />
      </div>

      <div className="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card title="Latest snapshot" className="lg:col-span-1">
          {snapshot.data ? (
            <dl className="space-y-3 text-sm">
              <div>
                <dt className="text-muted">Version</dt>
                <dd className="font-mono text-xs break-all">{snapshot.data.version_id}</dd>
              </div>
              <div>
                <dt className="text-muted">Hash</dt>
                <dd className="font-mono text-xs break-all">
                  {snapshot.data.hash.slice(0, 24)}…
                </dd>
              </div>
              <div>
                <dt className="text-muted">Baseline</dt>
                <dd>{snapshot.data.is_baseline ? "yes" : "no"}</dd>
              </div>
            </dl>
          ) : (
            <Empty>{snapshot.loading ? "Loading…" : "No snapshot yet."}</Empty>
          )}
        </Card>

        <Card
          title={
            <span className="flex items-center gap-2">
              Top findings
              {actionable > 0 && (
                <Pill className="text-critical">{actionable} need attention</Pill>
              )}
            </span>
          }
          className="lg:col-span-2"
          action={
            <Link to="/app/findings" className="text-xs text-brand hover:underline">
              View all →
            </Link>
          }
        >
          {topFindings.length > 0 ? (
            <ul className="divide-y divide-border/60">
              {topFindings.map((f) => (
                <li key={f.id}>
                  <Link
                    to="/app/findings"
                    className="flex items-start gap-3 py-2.5 -mx-2 px-2 rounded-lg hover:bg-surface-2/50"
                  >
                    <SeverityBadge severity={f.severity} />
                    <span className="min-w-0 flex-1">
                      <span className={`block text-sm font-medium ${severityColor[f.severity]}`}>
                        {f.headline}
                      </span>
                      <span className="block truncate text-xs text-muted">
                        {f.service} · {relativeTime(f.event.detected_at)}
                      </span>
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          ) : (
            <Empty>
              {changes.loading ? "Loading…" : "No findings — token contracts unchanged since baseline. 🎉"}
            </Empty>
          )}
        </Card>
      </div>

      <div className="mt-6">
        <Card
          title="Verification coverage"
          action={
            <Link to="/app/coverage" className="text-xs text-brand hover:underline">
              Full report →
            </Link>
          }
        >
          {threats.data ? (
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
              <div className="flex flex-col justify-center gap-1 lg:border-r lg:border-border/60 lg:pr-6">
                <div className="text-3xl font-semibold tabular-nums tracking-tight">
                  {threats.data.percent}%
                </div>
                <div className="text-caption text-muted">
                  {threats.data.covered} of {threats.data.total} documented threats detected
                </div>
                <div className="mt-1 text-caption text-high">
                  {threats.data.gaps} named gaps on the roadmap
                </div>
              </div>
              <div className="space-y-4 lg:col-span-2">
                {threats.data.domains.map((d) => (
                  <div key={d.domain}>
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium capitalize">
                        {d.domain === "jwt" ? "JWT / JWKS" : "Agent auth"}
                      </span>
                      <span className="tabular-nums text-muted">
                        {d.covered} / {d.total} · {d.percent}%
                      </span>
                    </div>
                    <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-surface-2">
                      <div
                        className={`h-full rounded-full ${d.domain === "jwt" ? "bg-accent" : "bg-medium"}`}
                        style={{ width: `${d.percent}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <Empty>{threats.loading ? "Loading coverage…" : "Coverage unavailable."}</Empty>
          )}
        </Card>
      </div>
    </Page>
  );
}
