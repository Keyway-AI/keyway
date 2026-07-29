import { useMemo, useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Card, Empty, Pill, SeverityBadge, StatTile, Table, Td, Th } from "../components/ui";
import { useAsync } from "../lib/useAsync";
import type { CoverageThreat, Severity } from "../api/types";

const severityRank: Record<Severity, number> = { critical: 0, high: 1, medium: 2, low: 3, info: 4 };

function Bar({ covered, total }: { covered: number; total: number }) {
  const pct = total ? Math.round((covered / total) * 100) : 0;
  return (
    <div className="h-1.5 overflow-hidden rounded-pill bg-surface-2">
      <div className="h-full rounded-pill bg-accent transition-all" style={{ width: `${pct}%` }} />
    </div>
  );
}

function DomainCard({ domain, covered, total, percent }: { domain: string; covered: number; total: number; percent: number }) {
  const label = domain === "jwt" ? "JWT / JWKS / OIDC" : "AI agent auth";
  return (
    <Card>
      <div className="flex items-baseline justify-between">
        <div>
          <div className="eyebrow">{domain}</div>
          <div className="mt-1 text-[1.05rem] font-semibold tracking-tight">{label}</div>
        </div>
        <div className="text-h1 font-semibold tabular-nums text-accent">{percent}%</div>
      </div>
      <div className="mt-4">
        <Bar covered={covered} total={total} />
        <div className="mt-2 text-caption text-muted">
          {covered} of {total} threats detected · {total - covered} gaps
        </div>
      </div>
    </Card>
  );
}

export default function Coverage() {
  const cov = useAsync(() => api.threatCoverage());
  const [domain, setDomain] = useState<"all" | "jwt" | "agent">("all");

  const threats = useMemo(() => {
    const list = cov.data?.threats ?? [];
    const filtered = domain === "all" ? list : list.filter((t) => t.domain === domain);
    return [...filtered].sort(
      (a, b) => severityRank[a.severity] - severityRank[b.severity] || a.id.localeCompare(b.id),
    );
  }, [cov.data, domain]);

  const d = cov.data;

  return (
    <Page
      title="Threat coverage"
      subtitle="Detection measured against the documented universe of JWT and agent-auth threats — RFC 8725, the MCP spec, OWASP, CVEs — not a corpus we wrote. Every gap is named."
    >
      {!d ? (
        <Card>
          <Empty>{cov.loading ? "Loading coverage…" : cov.error ?? "No coverage data."}</Empty>
        </Card>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatTile label="Documented" value={d.total} hint="cited threats" />
            <StatTile label="Detected" value={d.covered} accent="text-low" hint={`${d.percent}% overall`} />
            <StatTile label="Gaps" value={d.gaps} accent="text-high" hint="named + on the roadmap" />
            <StatTile label="Domains" value={d.domains.length} hint="jwt · agent" />
          </div>

          <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
            {d.domains.map((dm) => (
              <DomainCard key={dm.domain} {...dm} />
            ))}
          </div>

          <Card title="By category" className="mt-4">
            <div className="grid grid-cols-1 gap-x-8 gap-y-3 sm:grid-cols-2">
              {d.categories.map((c) => (
                <div key={c.category} className="flex items-center gap-3">
                  <span className="w-40 shrink-0 truncate text-body text-muted" title={c.category}>
                    {c.category.replace(/_/g, " ")}
                  </span>
                  <div className="min-w-0 flex-1">
                    <Bar covered={c.covered} total={c.total} />
                  </div>
                  <span className="w-12 shrink-0 text-right text-caption tabular-nums text-faint">
                    {c.covered}/{c.total}
                  </span>
                </div>
              ))}
            </div>
          </Card>

          <Card
            title="Threats"
            className="mt-4"
            bodyClassName="p-5 pt-4"
            action={
              <div className="flex items-center gap-1 rounded-md border border-border bg-surface-2 p-0.5">
                {(["all", "jwt", "agent"] as const).map((k) => (
                  <button
                    key={k}
                    onClick={() => setDomain(k)}
                    className={`rounded-[7px] px-2.5 py-1 text-caption font-medium capitalize transition ${
                      domain === k ? "bg-surface text-text shadow-xs" : "text-muted hover:text-text"
                    }`}
                  >
                    {k}
                  </button>
                ))}
              </div>
            }
          >
            <Table>
              <thead>
                <tr>
                  <Th className="w-24">ID</Th>
                  <Th className="w-24">Severity</Th>
                  <Th>Threat</Th>
                  <Th>Detection</Th>
                </tr>
              </thead>
              <tbody>
                {threats.map((t: CoverageThreat) => (
                  <tr key={t.id}>
                    <Td className="font-mono text-caption text-muted">{t.id}</Td>
                    <Td>
                      <SeverityBadge severity={t.severity} />
                    </Td>
                    <Td>
                      <div className="font-medium text-text">{t.title}</div>
                      <div className="mt-0.5 text-caption text-muted">{t.invariant}</div>
                    </Td>
                    <Td>
                      {t.detectors.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {t.detectors.map((det) => (
                            <Pill key={det} className="font-mono text-[10px] text-low">
                              {det}
                            </Pill>
                          ))}
                        </div>
                      ) : (
                        <span className="text-caption text-faint">— gap</span>
                      )}
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </Card>
        </>
      )}
    </Page>
  );
}
