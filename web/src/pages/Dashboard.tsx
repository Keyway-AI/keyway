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
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
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
    </Page>
  );
}
