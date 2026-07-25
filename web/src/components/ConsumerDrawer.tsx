import { useEffect } from "react";
import type { ReactNode } from "react";
import { api } from "../api/client";
import { useAsync } from "../lib/useAsync";
import { relativeTime, ttlLabel } from "../lib/format";
import type { Consumer, ProbeResult } from "../api/types";
import { Pill } from "./ui";

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="border-t border-border px-5 py-4">
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-muted">{title}</h3>
      {children}
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 py-1 text-sm">
      <span className="text-muted">{label}</span>
      <span className="text-right">{children}</span>
    </div>
  );
}

function Chips({ items }: { items: string[] }) {
  if (items.length === 0) return <span className="text-muted">—</span>;
  return (
    <span className="flex flex-wrap justify-end gap-1">
      {items.map((v) => (
        <Pill key={v} className="font-mono">
          {v}
        </Pill>
      ))}
    </span>
  );
}

function confidenceTone(v: number): string {
  if (v >= 0.9) return "bg-low";
  if (v >= 0.6) return "bg-medium";
  return "bg-high";
}

function ConfidenceRow({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex items-center gap-3 py-1 text-sm">
      <span className="w-56 shrink-0 truncate font-mono text-xs text-muted">{label}</span>
      <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-2">
        <div className={`h-full ${confidenceTone(value)}`} style={{ width: `${Math.round(value * 100)}%` }} />
      </div>
      <span className="w-10 text-right tabular-nums">{Math.round(value * 100)}%</span>
    </div>
  );
}

function ProbeHistory({ stableId }: { stableId: string }) {
  const { data, loading } = useAsync<ProbeResult[]>(() => api.consumerProbes(stableId), [stableId]);
  const rows = data ?? [];
  if (loading) return <div className="text-sm text-muted">Loading…</div>;
  if (rows.length === 0)
    return <div className="text-sm text-muted">No probe runs recorded for this consumer yet.</div>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="text-xs uppercase tracking-wide text-muted">
            <th className="py-1.5 text-left font-medium">Probe</th>
            <th className="py-1.5 text-left font-medium">Result</th>
            <th className="py-1.5 text-right font-medium">Status</th>
            <th className="py-1.5 text-right font-medium">Latency</th>
            <th className="py-1.5 text-right font-medium">When</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id} className="border-t border-border/60">
              <td className="py-1.5 font-mono text-xs">{r.probe_id}</td>
              <td className="py-1.5">
                {r.passed ? (
                  <span className="text-low">pass</span>
                ) : (
                  <span className="text-high">fail</span>
                )}
              </td>
              <td className="py-1.5 text-right tabular-nums">{r.status_code || "—"}</td>
              <td className="py-1.5 text-right tabular-nums">{r.latency_ms}ms</td>
              <td className="py-1.5 text-right text-muted">{relativeTime(r.run_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function ConsumerDrawer({ consumer, onClose }: { consumer: Consumer | null; onClose: () => void }) {
  // Close on Escape while open.
  useEffect(() => {
    if (!consumer) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [consumer, onClose]);

  if (!consumer) return null;

  const c = consumer;
  const confidenceKeys = Object.keys(c.confidence).sort((a) =>
    a === "overall" ? -1 : 1,
  );
  const provenanceEntries = Object.entries(c.provenance ?? {});

  return (
    <div className="fixed inset-0 z-40 flex justify-end">
      {/* Scrim */}
      <button
        aria-label="Close"
        onClick={onClose}
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px]"
      />
      {/* Panel */}
      <aside className="relative z-50 flex h-full w-full max-w-xl flex-col overflow-y-auto border-l border-border bg-surface shadow-2xl">
        <header className="sticky top-0 flex items-start justify-between gap-4 border-b border-border bg-surface px-5 py-4">
          <div>
            <div className="text-lg font-semibold">{c.name}</div>
            <div className="mt-0.5 font-mono text-xs text-muted">{c.stable_id}</div>
            <div className="mt-2 flex flex-wrap gap-1.5">
              <Pill>{c.kind}</Pill>
              {c.namespace && <Pill>ns: {c.namespace}</Pill>}
              {c.owner_team && <Pill>{c.owner_team}</Pill>}
              {c.probeable ? (
                <Pill className="text-low">probeable</Pill>
              ) : (
                <Pill className="text-muted">not probeable</Pill>
              )}
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-2 py-1 text-sm text-muted hover:bg-surface-2"
          >
            Close
          </button>
        </header>

        <Section title="Token contract">
          <Field label="Issuers">
            <Chips items={c.expects.issuers} />
          </Field>
          <Field label="Audiences">
            <Chips items={c.expects.audiences} />
          </Field>
          <Field label="Algorithms">
            <Chips items={c.expects.algorithms} />
          </Field>
          <Field label="Required claims">
            <Chips items={c.expects.required_claims} />
          </Field>
          <Field label="Clock skew">{c.expects.clock_skew_sec}s</Field>
        </Section>

        <Section title="JWKS behavior">
          <Field label="Cache TTL">{ttlLabel(c.jwks_behavior.cache_ttl_sec)}</Field>
          <Field label="Refreshes on unknown kid">
            {c.jwks_behavior.refreshes_on_unknown_kid == null ? (
              <span className="text-muted">unknown</span>
            ) : c.jwks_behavior.refreshes_on_unknown_kid ? (
              <span className="text-low">yes</span>
            ) : (
              <span className="text-high">no</span>
            )}
          </Field>
          <Field label="Source">{c.jwks_behavior.source}</Field>
          {c.library && (
            <Field label="Library">
              {c.library.name} {c.library.version}
            </Field>
          )}
        </Section>

        <Section title="Confidence">
          {confidenceKeys.map((k) => (
            <ConfidenceRow key={k} label={k} value={c.confidence[k]} />
          ))}
        </Section>

        <Section title="Provenance">
          {provenanceEntries.length === 0 ? (
            <div className="text-sm text-muted">No provenance recorded (sample data may omit it).</div>
          ) : (
            <div className="space-y-3">
              {provenanceEntries.map(([field, recs]) => (
                <div key={field}>
                  <div className="font-mono text-xs text-text">{field}</div>
                  {recs.map((r, i) => (
                    <div key={i} className="mt-1 rounded-lg border border-border bg-surface-2 px-3 py-2 text-xs">
                      <div className="flex justify-between gap-3">
                        <span className="font-medium">{r.source}</span>
                        <span className="text-muted">{Math.round(r.confidence * 100)}% · {relativeTime(r.observed_at)}</span>
                      </div>
                      <div className="mt-0.5 truncate font-mono text-muted">{r.locator}</div>
                    </div>
                  ))}
                </div>
              ))}
            </div>
          )}
        </Section>

        <Section title="Probe history">
          <ProbeHistory stableId={c.stable_id} />
        </Section>
      </aside>
    </div>
  );
}
