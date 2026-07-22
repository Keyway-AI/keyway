import { Link } from "react-router-dom";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Card, Empty, SeverityBadge, StatTile, Td, Th } from "../components/ui";
import { useAsync } from "../lib/useAsync";
import { relativeTime } from "../lib/format";

export default function Dashboard() {
  const coverage = useAsync(() => api.coverage());
  const snapshot = useAsync(() => api.latestSnapshot());
  const changes = useAsync(() => api.changes());

  const cov = coverage.data;
  const pct = cov && cov.total ? Math.round((cov.resolved / cov.total) * 100) : 0;

  return (
    <Page
      title="Dashboard"
      subtitle="Coverage and recent contract changes across your issuers."
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
          title="Recent changes"
          className="lg:col-span-2"
          action={
            <Link to="/changes" className="text-xs text-brand hover:underline">
              View all →
            </Link>
          }
        >
          {changes.data && changes.data.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full border-collapse">
                <thead>
                  <tr>
                    <Th>Consumer</Th>
                    <Th>Field</Th>
                    <Th>Class</Th>
                    <Th>Severity</Th>
                    <Th>When</Th>
                  </tr>
                </thead>
                <tbody>
                  {changes.data.map((c) => (
                    <tr key={c.id}>
                      <Td className="font-mono text-xs">{c.consumer_id.split("/").pop()}</Td>
                      <Td className="font-mono text-xs text-muted">{c.field}</Td>
                      <Td className="capitalize">{c.class}</Td>
                      <Td>
                        <SeverityBadge severity={c.severity} />
                      </Td>
                      <Td className="text-muted">{relativeTime(c.detected_at)}</Td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <Empty>{changes.loading ? "Loading…" : "No changes since baseline."}</Empty>
          )}
        </Card>
      </div>
    </Page>
  );
}
