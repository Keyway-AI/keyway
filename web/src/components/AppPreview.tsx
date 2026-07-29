import { IconAgent, IconBlast, IconCoverage, IconDashboard, IconFindings, IconProbes } from "./icons";
import { LogoBadge } from "./Logo";

/**
 * AppPreview is a framed, detailed render of the product — a "screenshot" built
 * from the real design tokens rather than a captured image, so it stays crisp and
 * theme-aware. Shows the real dashboard surface: stat tiles, top findings, a live
 * coverage panel, and recent harness activity.
 */
export function AppPreview() {
  const stats = [
    { label: "Consumers", value: "47", hint: "auto-discovered", accent: "text-text" },
    { label: "JWT coverage", value: "60%", hint: "21 / 35 threats", accent: "text-accent" },
    { label: "Agent auth", value: "40%", hint: "6 / 15 threats", accent: "text-medium" },
    { label: "Open gaps", value: "23", hint: "named + cited", accent: "text-high" },
  ];
  const findings = [
    { sev: "critical", label: "Now accepts UNSIGNED tokens (alg=none)", svc: "mobile-gw" },
    { sev: "high", label: "Won't pick up rotated signing keys", svc: "payments-api" },
    { sev: "medium", label: "Caches signing keys 6× longer", svc: "orders-api" },
    { sev: "low", label: "Audience list widened to include staging", svc: "search-svc" },
  ];
  const sevColor: Record<string, string> = {
    critical: "bg-critical",
    high: "bg-high",
    medium: "bg-medium",
    low: "bg-low",
  };
  const harness = [
    { a: "alg=none", ok: true },
    { a: "key confusion", ok: true },
    { a: "jku injection", ok: false },
  ];
  const nav = [
    { icon: IconDashboard, label: "Dashboard", active: true },
    { icon: IconFindings, label: "Findings" },
    { icon: IconProbes, label: "Probes" },
    { icon: IconBlast, label: "Blast radius" },
    { icon: IconCoverage, label: "Coverage" },
    { icon: IconAgent, label: "Agent auth" },
  ];

  return (
    <div className="overflow-hidden rounded-2xl border border-border bg-surface shadow-lg">
      {/* window chrome */}
      <div className="flex items-center gap-2 border-b border-border bg-surface-2/60 px-4 py-2.5">
        <span className="h-2.5 w-2.5 rounded-full bg-border-strong" />
        <span className="h-2.5 w-2.5 rounded-full bg-border-strong" />
        <span className="h-2.5 w-2.5 rounded-full bg-border-strong" />
        <div className="ml-3 hidden h-5 flex-1 items-center gap-1.5 rounded-md bg-surface px-2.5 text-[10px] text-faint sm:flex">
          <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" className="text-low">
            <path d="M7 11V8a5 5 0 0 1 10 0v3" strokeLinecap="round" />
            <rect x="5" y="11" width="14" height="9" rx="2" />
          </svg>
          app.keyway.dev
        </div>
      </div>
      {/* body */}
      <div className="flex min-h-[340px] text-left">
        <aside className="hidden w-44 shrink-0 flex-col border-r border-border p-3 sm:flex">
          <div className="flex items-center gap-2 px-2 pb-3">
            <LogoBadge size={18} />
            <span className="text-caption font-semibold tracking-tight">Keyway</span>
          </div>
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
          <div className="mt-auto flex items-center gap-2 px-2 pt-3 text-[10px] text-muted">
            <span className="h-1.5 w-1.5 rounded-full bg-low" />
            Connected
          </div>
        </aside>
        <div className="min-w-0 flex-1 p-4">
          <div className="flex items-center justify-between">
            <div className="text-[1.15rem] font-semibold tracking-tight">Dashboard</div>
            <span className="hidden rounded-md bg-accent px-2.5 py-1 text-[10px] font-semibold text-accent-fg sm:block">
              Take snapshot
            </span>
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
            {stats.map((s) => (
              <div key={s.label} className="rounded-lg border border-border bg-surface p-2.5">
                <div className="text-[9px] font-semibold uppercase tracking-wider text-faint">{s.label}</div>
                <div className={`mt-1 text-lg font-semibold tabular-nums ${s.accent}`}>{s.value}</div>
                <div className="text-[9px] text-faint">{s.hint}</div>
              </div>
            ))}
          </div>
          <div className="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-5">
            <div className="rounded-lg border border-border bg-surface lg:col-span-3">
              <div className="flex items-center justify-between border-b border-border px-3 py-2">
                <span className="text-caption font-semibold">Top findings</span>
                <span className="rounded-pill bg-critical/10 px-2 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-critical">
                  2 need attention
                </span>
              </div>
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
            <div className="space-y-3 lg:col-span-2">
              <div className="rounded-lg border border-border bg-surface p-3">
                <div className="eyebrow mb-2">Coverage</div>
                {[
                  { n: "JWT", pct: 60, tone: "bg-accent" },
                  { n: "Agent", pct: 40, tone: "bg-medium" },
                ].map((d) => (
                  <div key={d.n} className="mb-2 last:mb-0">
                    <div className="flex justify-between text-[10px] text-muted">
                      <span>{d.n}</span>
                      <span className="tabular-nums">{d.pct}%</span>
                    </div>
                    <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-surface-2">
                      <div className={`h-full rounded-full ${d.tone}`} style={{ width: `${d.pct}%` }} />
                    </div>
                  </div>
                ))}
              </div>
              <div className="rounded-lg border border-border bg-surface p-3">
                <div className="eyebrow mb-2">Latest harness run</div>
                <div className="space-y-1">
                  {harness.map((h) => (
                    <div key={h.a} className="flex items-center gap-2 text-[10px]">
                      <span className={h.ok ? "text-low" : "text-critical"}>{h.ok ? "✓" : "✗"}</span>
                      <span className="flex-1 truncate text-muted">{h.a}</span>
                      <span className={h.ok ? "text-low" : "text-critical"}>{h.ok ? "rejected" : "accepted"}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
