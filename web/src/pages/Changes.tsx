import { useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Card, Empty, Pill, SeverityBadge, Td, Th } from "../components/ui";
import { useAsync } from "../lib/useAsync";
import { relativeTime } from "../lib/format";
import type { Severity } from "../api/types";

const severities: (Severity | "all")[] = ["all", "critical", "high", "medium", "low", "info"];

function fmtValue(v: unknown): string {
  if (v == null) return "∅";
  if (Array.isArray(v)) return `[${v.join(", ")}]`;
  return String(v);
}

export default function Changes() {
  const [sev, setSev] = useState<Severity | "all">("all");
  const { data, loading } = useAsync(() => api.changes());

  const rows = (data ?? []).filter((c) => sev === "all" || c.severity === sev);

  return (
    <Page
      title="Changes"
      subtitle="Classified diffs between contract versions. Unknown-class changes never page."
      actions={
        <div className="flex gap-1">
          {severities.map((s) => (
            <button
              key={s}
              onClick={() => setSev(s)}
              className={`rounded-lg px-2.5 py-1 text-xs capitalize transition-colors ${
                sev === s ? "bg-surface-2 text-text" : "text-muted hover:text-text"
              }`}
            >
              {s}
            </button>
          ))}
        </div>
      }
    >
      <Card>
        {rows.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                <tr>
                  <Th>Consumer</Th>
                  <Th>Field</Th>
                  <Th>Change</Th>
                  <Th>Class</Th>
                  <Th>Severity</Th>
                  <Th>Evidence</Th>
                  <Th>When</Th>
                </tr>
              </thead>
              <tbody>
                {rows.map((c) => (
                  <tr key={c.id} className="hover:bg-surface-2/50">
                    <Td className="font-mono text-xs">{c.consumer_id.split("/").pop()}</Td>
                    <Td className="font-mono text-xs text-muted">{c.field}</Td>
                    <Td className="font-mono text-xs">
                      <span className="text-muted">{fmtValue(c.old_value)}</span>
                      <span className="mx-1.5 text-brand">→</span>
                      <span>{fmtValue(c.new_value)}</span>
                    </Td>
                    <Td className="capitalize">{c.class}</Td>
                    <Td>
                      <SeverityBadge severity={c.severity} />
                    </Td>
                    <Td>
                      {c.evidence.map((e) => (
                        <Pill key={e} className="mr-1 font-mono text-[10px]">
                          {e}
                        </Pill>
                      ))}
                    </Td>
                    <Td className="text-muted">{relativeTime(c.detected_at)}</Td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <Empty>{loading ? "Loading…" : "No changes match this filter."}</Empty>
        )}
      </Card>
    </Page>
  );
}
