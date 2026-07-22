import { useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Card, Empty, Pill, StatTile } from "../components/ui";
import { useAsync } from "../lib/useAsync";
import { computeRotateKey } from "../lib/blast";
import { humanizeDuration, verdictColor, verdictLabel } from "../lib/format";
import type { AffectedConsumer, BlastRadiusResult } from "../api/types";

function VerdictGroup({
  title,
  verdict,
  rows,
}: {
  title: string;
  verdict: AffectedConsumer["verdict"];
  rows: AffectedConsumer[];
}) {
  if (rows.length === 0) return null;
  return (
    <div className="mb-5">
      <h3 className={`mb-2 text-sm font-semibold ${verdictColor[verdict]}`}>
        {title} ({rows.length})
      </h3>
      <div className="space-y-2">
        {rows.map((a) => (
          <div
            key={a.consumer.stable_id}
            className="flex items-start justify-between gap-4 rounded-lg border border-border bg-surface px-4 py-3"
          >
            <div>
              <div className="font-medium">{a.consumer.name}</div>
              <div className="text-xs text-muted">{a.reason}</div>
              {a.consumer.owner_team && (
                <div className="mt-1 text-xs text-muted">owner: {a.consumer.owner_team}</div>
              )}
            </div>
            <div className="flex flex-col items-end gap-1">
              {a.evidence.map((e) => (
                <Pill key={e} className="font-mono text-[10px]">
                  {e}
                </Pill>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default function BlastRadius() {
  const consumers = useAsync(() => api.consumers());
  const [kid, setKid] = useState("rsa-2026-01");
  const [result, setResult] = useState<BlastRadiusResult>();

  function run() {
    if (!consumers.data) return;
    setResult(computeRotateKey(consumers.data, kid));
  }

  const willBreak = result?.affected.filter((a) => a.verdict === "will_break") ?? [];
  const ready = result?.affected.filter((a) => a.verdict === "ready") ?? [];
  const unknown = result?.affected.filter((a) => a.verdict === "unknown") ?? [];

  return (
    <Page
      title="Blast radius"
      subtitle="Who breaks if you rotate this signing key — and the safe grace period."
    >
      <Card className="mb-6">
        <div className="flex flex-wrap items-end gap-4">
          <label className="text-sm">
            <span className="mb-1 block text-muted">Proposal</span>
            <select className="rounded-lg border border-border bg-surface-2 px-3 py-1.5 text-sm outline-none">
              <option>rotate-key</option>
            </select>
          </label>
          <label className="text-sm">
            <span className="mb-1 block text-muted">Key ID (kid)</span>
            <input
              value={kid}
              onChange={(e) => setKid(e.target.value)}
              className="rounded-lg border border-border bg-surface-2 px-3 py-1.5 font-mono text-sm outline-none focus:border-brand"
            />
          </label>
          <button
            onClick={run}
            disabled={!consumers.data}
            className="rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-strong disabled:opacity-50"
          >
            Compute
          </button>
        </div>
      </Card>

      {result ? (
        <>
          <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-4">
            <StatTile label="Affected" value={result.affected.length} />
            <StatTile label="Will break" value={willBreak.length} accent="text-critical" />
            <StatTile label="Ready" value={ready.length} accent="text-low" />
            <StatTile
              label="Grace period"
              value={humanizeDuration(result.recommended_grace_period_seconds)}
              hint={result.grace_basis ? `bound by ${result.grace_basis.split("/").pop()}` : undefined}
              accent="text-brand"
            />
          </div>

          <Card title={`Rotating ${kid}`}>
            <VerdictGroup title="Will break" verdict="will_break" rows={willBreak} />
            <VerdictGroup title="Ready" verdict="ready" rows={ready} />
            <VerdictGroup title="Unknown" verdict="unknown" rows={unknown} />
            {unknown.length > 0 && (
              <p className="mt-2 text-xs text-medium">
                {unknown.length} consumer(s) unknown — treat the grace period as a lower bound.
              </p>
            )}
          </Card>
        </>
      ) : (
        <Card>
          <Empty>
            {consumers.loading
              ? "Loading consumers…"
              : `Choose a key and press Compute. ${verdictLabel.ready}/${verdictLabel.will_break} verdicts appear here.`}
          </Empty>
        </Card>
      )}
    </Page>
  );
}
