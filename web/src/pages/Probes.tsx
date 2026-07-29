import { useMemo, useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Button, Card, Empty, Pill, StatTile, Table, Td, Th } from "../components/ui";
import { useToast } from "../components/toast";
import { useAsync } from "../lib/useAsync";
import { relativeTime } from "../lib/format";
import type { ProbeResult } from "../api/types";

// Human labels for the safety probes (PRD §5). Keep in sync with the probe IDs
// the backend emits; unknown IDs fall back to the raw id.
const PROBE_LABELS: Record<string, string> = {
  alg_none: "alg=none accepted",
  alg_confusion: "RS/HS key confusion",
  tampered_signature: "Tampered signature",
  expired: "Expired token accepted",
  not_yet_valid: "nbf in the future",
  wrong_audience: "Wrong audience",
  wrong_issuer: "Wrong issuer",
  unknown_kid: "Unknown kid",
  missing_kid: "Missing kid",
  valid_token: "Valid token (control)",
};

function probeLabel(id: string): string {
  return PROBE_LABELS[id] ?? id;
}

export default function Probes() {
  const toast = useToast();
  const consumers = useAsync(() => api.consumers());
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [results, setResults] = useState<ProbeResult[]>();
  const [runId, setRunId] = useState<string>();
  const [running, setRunning] = useState(false);

  const probeable = useMemo(() => (consumers.data ?? []).filter((c) => c.probeable), [consumers.data]);

  function toggle(id: string) {
    setSelected((s) => {
      const next = new Set(s);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function run() {
    setRunning(true);
    try {
      const ids = selected.size > 0 ? [...selected] : undefined;
      const res = await api.runProbes(ids);
      setResults(res.results);
      setRunId(res.run_id);
      const failed = res.results.filter((r) => !r.passed).length;
      if (failed > 0) toast.error(`${failed} probe(s) failed — see findings below`);
      else toast.success(`All ${res.results.length} probes passed`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Probe run failed");
    } finally {
      setRunning(false);
    }
  }

  const passed = results?.filter((r) => r.passed).length ?? 0;
  const failed = results?.filter((r) => !r.passed).length ?? 0;

  // Group results by consumer for a readable report.
  const byConsumer = useMemo(() => {
    const m = new Map<string, ProbeResult[]>();
    for (const r of results ?? []) {
      const arr = m.get(r.consumer_id) ?? [];
      arr.push(r);
      m.set(r.consumer_id, arr);
    }
    return [...m.entries()];
  }, [results]);

  const nameFor = (id: string) => probeable.find((c) => c.stable_id === id)?.name ?? id;

  return (
    <Page
      title="Probes"
      subtitle="Safely test each consumer against forged tokens (alg=none, key confusion, expiry, wrong audience…). Only benign, expected-to-be-rejected tokens are sent."
      actions={
        <Button variant="primary" onClick={run} loading={running} disabled={probeable.length === 0}>
          {selected.size > 0 ? `Run probes (${selected.size})` : "Run all probes"}
        </Button>
      }
    >
      <Card
        title="Targets"
        action={
          probeable.length > 0 && (
            <button
              onClick={() => setSelected((s) => (s.size === probeable.length ? new Set() : new Set(probeable.map((c) => c.stable_id))))}
              className="text-xs text-brand hover:underline"
            >
              {selected.size === probeable.length ? "Clear" : "Select all"}
            </button>
          )
        }
      >
        {probeable.length > 0 ? (
          <ul className="space-y-2">
            {probeable.map((c) => (
              <li key={c.stable_id}>
                <label className="flex cursor-pointer items-center gap-3 rounded-lg border border-border bg-surface px-4 py-3 hover:bg-surface-2/40">
                  <input
                    type="checkbox"
                    checked={selected.has(c.stable_id)}
                    onChange={() => toggle(c.stable_id)}
                    className="h-4 w-4 accent-[var(--color-brand)]"
                  />
                  <span className="flex-1">
                    <span className="block text-sm font-medium">{c.name}</span>
                    <span className="block text-xs text-muted">{c.stable_id}</span>
                  </span>
                  {c.owner_team && <Pill className="text-muted">{c.owner_team}</Pill>}
                </label>
              </li>
            ))}
          </ul>
        ) : (
          <Empty>
            {consumers.loading
              ? "Loading consumers…"
              : "No probeable consumers. Take a snapshot first, or check that endpoints were discovered."}
          </Empty>
        )}
      </Card>

      {results && (
        <>
          <div className="mt-6 grid grid-cols-3 gap-3 sm:gap-4">
            <StatTile label="Probes run" value={results.length} hint={runId ? `run ${runId.slice(0, 8)}` : undefined} />
            <StatTile label="Passed" value={passed} accent="text-low" />
            <StatTile label="Failed" value={failed} accent={failed > 0 ? "text-critical" : "text-muted"} />
          </div>

          <div className="mt-6 space-y-6">
            {byConsumer.map(([cid, rows]) => {
              const cfailed = rows.filter((r) => !r.passed).length;
              return (
                <Card
                  key={cid}
                  title={
                    <span className="flex items-center gap-2">
                      {nameFor(cid)}
                      {cfailed > 0 ? (
                        <Pill className="text-critical">{cfailed} failed</Pill>
                      ) : (
                        <Pill className="text-low">all passed</Pill>
                      )}
                    </span>
                  }
                >
                  <Table minWidth="min-w-[40rem]">
                    <thead>
                      <tr>
                        <Th>Probe</Th>
                        <Th>Result</Th>
                        <Th>Status</Th>
                        <Th>Latency</Th>
                        <Th>When</Th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map((r) => (
                        <tr key={r.id}>
                          <Td>{probeLabel(r.probe_id)}</Td>
                          <Td>
                            {r.passed ? (
                              <span className="text-low">✓ rejected as expected</span>
                            ) : (
                              <span className="text-critical">✕ accepted — vulnerable</span>
                            )}
                          </Td>
                          <Td className="font-mono text-xs">{r.status_code}</Td>
                          <Td className="text-muted">{r.latency_ms}ms</Td>
                          <Td className="text-muted">{r.run_at ? relativeTime(r.run_at) : "—"}</Td>
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                </Card>
              );
            })}
          </div>
        </>
      )}
    </Page>
  );
}
