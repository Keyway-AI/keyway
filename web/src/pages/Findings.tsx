import { useMemo, useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Card, Empty, Pill, SeverityBadge } from "../components/ui";
import { useAsync } from "../lib/useAsync";
import { relativeTime, severityColor } from "../lib/format";
import { countBySeverity, groupByService, toFinding } from "../lib/findings";
import type { Finding } from "../lib/findings";
import type { Severity } from "../api/types";

const order: Severity[] = ["critical", "high", "medium", "low", "info"];

function DirectionTag({ d }: { d: Finding["direction"] }) {
  const map: Record<Finding["direction"], { label: string; cls: string }> = {
    loosening: { label: "Accepts more", cls: "text-critical" },
    tightening: { label: "Accepts less", cls: "text-medium" },
    rotation: { label: "Rotation risk", cls: "text-medium" },
    neutral: { label: "Inventory", cls: "text-muted" },
    unknown: { label: "Low confidence", cls: "text-muted" },
  };
  const m = map[d];
  return <Pill className={m.cls}>{m.label}</Pill>;
}

function FindingRow({ f }: { f: Finding }) {
  const attr = f.event.attribution;
  return (
    <div className="border-t border-border/60 py-4 first:border-t-0 first:pt-0">
      <div className="flex flex-wrap items-center gap-2">
        <SeverityBadge severity={f.severity} />
        <DirectionTag d={f.direction} />
        <Pill>{f.kind}</Pill>
        <span className="ml-auto text-xs text-muted">{relativeTime(f.event.detected_at)}</span>
      </div>
      <h4 className={`mt-2 text-sm font-semibold ${severityColor[f.severity]}`}>{f.headline}</h4>
      <p className="mt-1 text-sm text-text/90">{f.meaning}</p>
      <div className="mt-2 grid gap-2 sm:grid-cols-2">
        <div className="rounded-lg border border-border bg-surface-2/50 px-3 py-2">
          <div className="text-[11px] uppercase tracking-wide text-muted">Why it matters</div>
          <div className="mt-0.5 text-xs text-text/90">{f.impact}</div>
        </div>
        <div className="rounded-lg border border-border bg-surface-2/50 px-3 py-2">
          <div className="text-[11px] uppercase tracking-wide text-muted">What to do</div>
          <div className="mt-0.5 text-xs text-text/90">{f.action}</div>
        </div>
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-1.5 text-xs text-muted">
        <span className="font-mono">{f.event.field}</span>
        {attr && attr.kind !== "unattributed" && (
          <Pill className="font-mono text-[10px]">
            by {attr.actor || attr.kind}
            {attr.ref ? ` · ${attr.ref.slice(0, 10)}` : ""}
          </Pill>
        )}
        {f.event.evidence.slice(0, 2).map((e) => (
          <Pill key={e} className="font-mono text-[10px]">
            {e}
          </Pill>
        ))}
      </div>
    </div>
  );
}

export default function Findings() {
  const { data, loading } = useAsync(() => api.changes());
  const [sev, setSev] = useState<Severity | "all">("all");

  const findings = useMemo(() => (data ?? []).map(toFinding), [data]);
  const counts = useMemo(() => countBySeverity(findings), [findings]);
  const filtered = sev === "all" ? findings : findings.filter((f) => f.severity === sev);
  const groups = useMemo(() => groupByService(filtered), [filtered]);

  return (
    <Page
      title="Findings"
      subtitle="What changed about who can log into your services — in plain English, grouped by service and prioritised by risk."
    >
      {/* Severity overview — click to filter. */}
      <div className="mb-5 grid grid-cols-2 gap-3 sm:grid-cols-5">
        {order.map((s) => (
          <button
            key={s}
            onClick={() => setSev(sev === s ? "all" : s)}
            className={`rounded-xl border bg-surface p-4 text-left transition-colors ${
              sev === s ? "border-brand" : "border-border hover:border-border/80"
            }`}
          >
            <div className={`text-2xl font-semibold ${severityColor[s]}`}>{counts[s]}</div>
            <div className="mt-0.5 text-xs capitalize text-muted">{s}</div>
          </button>
        ))}
      </div>

      {sev !== "all" && (
        <button onClick={() => setSev("all")} className="mb-3 text-xs text-brand hover:underline">
          ← clear “{sev}” filter
        </button>
      )}

      {groups.length > 0 ? (
        <div className="space-y-4">
          {groups.map((g) => (
            <Card
              key={g.serviceId}
              title={
                <span className="flex items-center gap-2">
                  <SeverityBadge severity={g.worst} />
                  <span>{g.service}</span>
                  <span className="font-mono text-xs font-normal text-muted">{g.serviceId}</span>
                </span>
              }
              action={
                <span className="text-xs text-muted">
                  {g.findings.length} finding{g.findings.length === 1 ? "" : "s"}
                </span>
              }
            >
              {g.findings.map((f) => (
                <FindingRow key={f.id} f={f} />
              ))}
            </Card>
          ))}
        </div>
      ) : (
        <Card>
          <Empty>
            {loading
              ? "Analysing changes…"
              : findings.length === 0
                ? "No findings — every service's token contract is unchanged since the baseline. 🎉"
                : "No findings match this severity."}
          </Empty>
        </Card>
      )}
    </Page>
  );
}
