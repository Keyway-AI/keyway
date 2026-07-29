import { IconBlast, IconCoverage, IconDashboard, IconFindings, IconProbes } from "./icons";

/**
 * AppPreview is a framed, static render of the product — a "screenshot" built
 * from the real design tokens rather than a captured image, so it stays crisp and
 * theme-aware. Purely decorative on the marketing page.
 */
export function AppPreview() {
  const stats = [
    { label: "Consumers", value: "47", accent: "text-text" },
    { label: "Coverage", value: "60%", accent: "text-low" },
    { label: "Agent auth", value: "40%", accent: "text-accent" },
    { label: "Gaps", value: "23", accent: "text-high" },
  ];
  const findings = [
    { sev: "critical", label: "Now accepts UNSIGNED tokens (alg=none)", svc: "mobile-gw" },
    { sev: "high", label: "Won't pick up rotated signing keys", svc: "payments-api" },
    { sev: "medium", label: "Caches signing keys longer", svc: "orders-api" },
  ];
  const sevColor: Record<string, string> = { critical: "bg-critical", high: "bg-high", medium: "bg-medium" };
  const nav = [
    { icon: IconDashboard, label: "Dashboard", active: true },
    { icon: IconFindings, label: "Findings" },
    { icon: IconProbes, label: "Probes" },
    { icon: IconBlast, label: "Blast radius" },
    { icon: IconCoverage, label: "Coverage" },
  ];

  return (
    <div className="overflow-hidden rounded-2xl border border-border bg-surface shadow-lg">
      {/* window chrome */}
      <div className="flex items-center gap-2 border-b border-border bg-surface-2/60 px-4 py-2.5">
        <span className="h-2.5 w-2.5 rounded-full bg-border-strong" />
        <span className="h-2.5 w-2.5 rounded-full bg-border-strong" />
        <span className="h-2.5 w-2.5 rounded-full bg-border-strong" />
        <div className="ml-3 hidden h-5 flex-1 items-center rounded-md bg-surface px-2.5 text-[10px] text-faint sm:flex">
          app.keyway.dev
        </div>
      </div>
      {/* body */}
      <div className="flex min-h-[300px]">
        <aside className="hidden w-40 shrink-0 flex-col gap-0.5 border-r border-border p-3 sm:flex">
          <div className="eyebrow px-2 pb-1">Overview</div>
          {nav.map((n) => (
            <div
              key={n.label}
              className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-caption font-medium ${
                n.active ? "bg-accent-soft text-accent" : "text-muted"
              }`}
            >
              <n.icon width="14" height="14" />
              {n.label}
            </div>
          ))}
        </aside>
        <div className="min-w-0 flex-1 p-4">
          <div className="text-[1.15rem] font-semibold tracking-tight">Dashboard</div>
          <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
            {stats.map((s) => (
              <div key={s.label} className="rounded-lg border border-border bg-surface p-2.5">
                <div className="text-[9px] font-semibold uppercase tracking-wider text-faint">{s.label}</div>
                <div className={`mt-1 text-lg font-semibold tabular-nums ${s.accent}`}>{s.value}</div>
              </div>
            ))}
          </div>
          <div className="mt-3 rounded-lg border border-border bg-surface">
            <div className="border-b border-border px-3 py-2 text-caption font-semibold">Top findings</div>
            <ul>
              {findings.map((f) => (
                <li key={f.label} className="flex items-center gap-2.5 border-b border-border/60 px-3 py-2 last:border-0">
                  <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${sevColor[f.sev]}`} />
                  <span className="min-w-0 flex-1 truncate text-caption text-text">{f.label}</span>
                  <span className="hidden shrink-0 text-[10px] text-faint sm:block">{f.svc}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
