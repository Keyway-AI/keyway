import { useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Card, Empty, Pill, Td, Th } from "../components/ui";
import { ConsumerDrawer } from "../components/ConsumerDrawer";
import { useAsync } from "../lib/useAsync";
import { ttlLabel } from "../lib/format";
import type { Consumer } from "../api/types";

function RefreshFlag({ c }: { c: Consumer }) {
  const v = c.jwks_behavior.refreshes_on_unknown_kid;
  if (v == null) return <span className="text-muted">unknown</span>;
  return v ? (
    <span className="text-low">refreshes</span>
  ) : (
    <span className="text-high">no refresh</span>
  );
}

export default function Consumers() {
  const { data, loading } = useAsync(() => api.consumers());
  const [q, setQ] = useState("");
  const [selected, setSelected] = useState<Consumer | null>(null);

  const rows = (data ?? []).filter(
    (c) =>
      !q ||
      c.name.toLowerCase().includes(q.toLowerCase()) ||
      c.stable_id.toLowerCase().includes(q.toLowerCase()) ||
      (c.owner_team ?? "").toLowerCase().includes(q.toLowerCase()),
  );

  return (
    <Page
      title="Consumers"
      subtitle="Every component that validates tokens, derived from your cluster and config. Select a row for provenance, confidence, and probe history."
      actions={
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Filter…"
          className="rounded-lg border border-border bg-surface px-3 py-1.5 text-sm outline-none focus:border-brand"
        />
      }
    >
      <Card>
        {rows.length > 0 ? (
          <div className="-mx-5 overflow-x-auto px-5">
            <table className="w-full min-w-[52rem] border-collapse">
              <thead>
                <tr>
                  <Th>Name</Th>
                  <Th>Stable ID</Th>
                  <Th>Owner</Th>
                  <Th>Library</Th>
                  <Th>JWKS cache</Th>
                  <Th>Unknown kid</Th>
                  <Th>Probeable</Th>
                </tr>
              </thead>
              <tbody>
                {rows.map((c) => (
                  <tr
                    key={c.stable_id}
                    onClick={() => setSelected(c)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        setSelected(c);
                      }
                    }}
                    role="button"
                    tabIndex={0}
                    aria-label={`Open details for ${c.name}`}
                    className="cursor-pointer hover:bg-surface-2/50 focus-visible:bg-surface-2/50"
                  >
                    <Td className="font-medium">{c.name}</Td>
                    <Td className="font-mono text-xs text-muted">{c.stable_id}</Td>
                    <Td>{c.owner_team ? <Pill>{c.owner_team}</Pill> : "—"}</Td>
                    <Td className="text-xs">
                      {c.library ? `${c.library.name} ${c.library.version}` : "—"}
                    </Td>
                    <Td>{ttlLabel(c.jwks_behavior.cache_ttl_sec)}</Td>
                    <Td>
                      <RefreshFlag c={c} />
                    </Td>
                    <Td>
                      {c.probeable ? (
                        <span className="text-low">yes</span>
                      ) : (
                        <span className="text-muted">no</span>
                      )}
                    </Td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <Empty>{loading ? "Loading…" : "No consumers discovered yet."}</Empty>
        )}
      </Card>
      <ConsumerDrawer consumer={selected} onClose={() => setSelected(null)} />
    </Page>
  );
}
