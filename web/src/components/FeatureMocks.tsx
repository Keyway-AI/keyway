import type { ReactNode } from "react";

/**
 * Detailed, theme-aware product mockups used as the "screenshots" beside each
 * marketing feature. Rendered from real design tokens rather than raster images —
 * they show the actual product surfaces (diff, harness, blast radius, coverage,
 * inspector) so the landing page demonstrates the tool instead of decorating it.
 */

function MockFrame({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-surface shadow-lg">
      <div className="flex items-center gap-2 border-b border-border bg-surface-2/60 px-3.5 py-2.5">
        <span className="h-2 w-2 rounded-full bg-border-strong" />
        <span className="h-2 w-2 rounded-full bg-border-strong" />
        <span className="h-2 w-2 rounded-full bg-border-strong" />
        <span className="ml-2 text-[11px] font-medium text-faint">{label}</span>
      </div>
      <div className="p-4">{children}</div>
    </div>
  );
}

const sevDot: Record<string, string> = {
  critical: "bg-critical",
  high: "bg-high",
  medium: "bg-medium",
  low: "bg-low",
};

/* ── Contract & drift: a claim-level diff ─────────────────────────────── */
export function DiffMock() {
  const rows = [
    { sign: "-", text: "aud: [\"payments-api\"]", tone: "del" },
    { sign: "+", text: "aud: [\"payments-api\", \"*\"]", tone: "add", note: "audience widened" },
    { sign: "-", text: "alg: [\"RS256\"]", tone: "del" },
    { sign: "+", text: "alg: [\"RS256\", \"none\"]", tone: "add", note: "unsigned allowed" },
    { sign: " ", text: "require: [\"sub\", \"exp\", \"iss\"]", tone: "ctx" },
  ];
  return (
    <MockFrame label="payments-api · contract diff">
      <div className="flex items-center justify-between">
        <span className="text-caption font-semibold">v14 → v15</span>
        <span className="rounded-pill bg-critical/10 px-2 py-0.5 text-micro font-semibold uppercase tracking-wider text-critical">
          2 breaking
        </span>
      </div>
      <div className="mt-3 overflow-hidden rounded-lg border border-border font-mono text-[11px] leading-relaxed">
        {rows.map((r, i) => (
          <div
            key={i}
            className={`flex items-center gap-2 px-3 py-1 ${
              r.tone === "add"
                ? "bg-low/8 text-text"
                : r.tone === "del"
                  ? "bg-critical/8 text-muted line-through decoration-critical/40"
                  : "text-faint"
            }`}
          >
            <span className={`w-2 ${r.tone === "add" ? "text-low" : r.tone === "del" ? "text-critical" : "text-faint"}`}>
              {r.sign}
            </span>
            <span className="flex-1 truncate">{r.text}</span>
            {r.note && <span className="text-[10px] font-sans text-muted">{r.note}</span>}
          </div>
        ))}
      </div>
    </MockFrame>
  );
}

/* ── Generative attack harness: pass/fail probe table ─────────────────── */
export function HarnessMock() {
  const rows = [
    { attack: "alg=none (unsigned)", threat: "JWT-01", ok: true },
    { attack: "HS/RS key confusion", threat: "JWT-03", ok: true },
    { attack: "jku header injection", threat: "JWT-07", ok: false },
    { attack: "expired token replay", threat: "JWT-11", ok: true },
    { attack: "forged kid path traversal", threat: "JWT-09", ok: true },
  ];
  return (
    <MockFrame label="orders-api · harness run">
      <div className="flex items-center justify-between text-caption">
        <span className="font-semibold">42 tokens fired</span>
        <span className="text-muted">41 rejected · <span className="font-semibold text-critical">1 accepted</span></span>
      </div>
      <div className="mt-3 space-y-1">
        {rows.map((r) => (
          <div
            key={r.attack}
            className="flex items-center gap-2.5 rounded-md border border-border/70 bg-surface px-2.5 py-1.5"
          >
            <span
              className={`grid h-4 w-4 shrink-0 place-items-center rounded-full text-[9px] font-bold ${
                r.ok ? "bg-low/15 text-low" : "bg-critical/15 text-critical"
              }`}
            >
              {r.ok ? "✓" : "✗"}
            </span>
            <span className="min-w-0 flex-1 truncate text-caption text-text">{r.attack}</span>
            <span className="shrink-0 font-mono text-[10px] text-faint">{r.threat}</span>
            <span className={`shrink-0 text-[10px] font-medium ${r.ok ? "text-low" : "text-critical"}`}>
              {r.ok ? "rejected" : "accepted"}
            </span>
          </div>
        ))}
      </div>
    </MockFrame>
  );
}

/* ── Blast radius: who breaks if you rotate ───────────────────────────── */
export function BlastMock() {
  const rows = [
    { svc: "mobile-gw", impact: "breaks", tone: "critical", why: "pins old kid" },
    { svc: "payments-api", impact: "breaks", tone: "high", why: "no JWKS refresh" },
    { svc: "orders-api", impact: "safe", tone: "low", why: "refreshes hourly" },
    { svc: "search-svc", impact: "safe", tone: "low", why: "refreshes hourly" },
  ];
  return (
    <MockFrame label="simulate · rotate signing key">
      <div className="rounded-lg bg-surface-2/60 px-3 py-2 text-caption">
        <span className="text-muted">Proposal</span>{" "}
        <span className="font-medium text-text">retire kid <span className="font-mono text-[11px]">2024-06</span></span>
      </div>
      <div className="mt-3 space-y-1.5">
        {rows.map((r) => (
          <div key={r.svc} className="flex items-center gap-2.5">
            <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${sevDot[r.tone]}`} />
            <span className="w-24 shrink-0 truncate text-caption font-medium text-text">{r.svc}</span>
            <span
              className={`rounded-pill px-2 py-0.5 text-micro font-semibold uppercase tracking-wide ${
                r.impact === "breaks" ? "bg-critical/10 text-critical" : "bg-low/12 text-low"
              }`}
            >
              {r.impact}
            </span>
            <span className="truncate text-[10px] text-faint">{r.why}</span>
          </div>
        ))}
      </div>
      <div className="mt-3 flex items-center gap-2 border-t border-border pt-2.5 text-caption">
        <span className="font-semibold text-high">2 consumers break</span>
        <span className="text-muted">· safe grace period 26h</span>
      </div>
    </MockFrame>
  );
}

/* ── Coverage, kept honest: domain bars ───────────────────────────────── */
function Bar({ pct, tone }: { pct: number; tone: string }) {
  return (
    <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-2">
      <div className={`h-full rounded-full ${tone}`} style={{ width: `${pct}%` }} />
    </div>
  );
}
export function CoverageMock() {
  const domains = [
    { name: "JWT / JWKS", pct: 60, n: "21 / 35", tone: "bg-accent" },
    { name: "Agent auth", pct: 40, n: "6 / 15", tone: "bg-medium" },
  ];
  const cats = [
    { name: "Signature forgery", pct: 83 },
    { name: "Key management", pct: 67 },
    { name: "Token binding", pct: 50 },
    { name: "Delegation", pct: 25 },
  ];
  return (
    <MockFrame label="coverage · threat taxonomy">
      <div className="space-y-3">
        {domains.map((d) => (
          <div key={d.name}>
            <div className="flex items-center justify-between text-caption">
              <span className="font-medium text-text">{d.name}</span>
              <span className="tabular-nums text-muted">{d.n}</span>
            </div>
            <div className="mt-1.5 flex items-center gap-2">
              <Bar pct={d.pct} tone={d.tone} />
              <span className="w-8 text-right text-[10px] font-semibold tabular-nums text-muted">{d.pct}%</span>
            </div>
          </div>
        ))}
      </div>
      <div className="mt-3.5 border-t border-border pt-3">
        <div className="eyebrow mb-2">By category</div>
        <div className="grid grid-cols-2 gap-x-4 gap-y-2">
          {cats.map((c) => (
            <div key={c.name} className="flex items-center gap-2">
              <span className="w-24 shrink-0 truncate text-[10px] text-muted">{c.name}</span>
              <Bar pct={c.pct} tone="bg-border-strong" />
            </div>
          ))}
        </div>
      </div>
    </MockFrame>
  );
}

/* ── Agent auth: token inspector findings ─────────────────────────────── */
export function AgentMock() {
  const findings = [
    { id: "MCP-01", tone: "critical", title: "Token audience is not this server", detail: "aud: auth0-mgmt" },
    { id: "DEL-01", tone: "high", title: "No act claim — delegation unverifiable", detail: "on-behalf-of" },
    { id: "SCOPE-01", tone: "high", title: "Over-scoped: files:* granted", detail: "uses files:read" },
    { id: "SCOPE-02", tone: "medium", title: "Token never expires", detail: "no exp" },
  ];
  return (
    <MockFrame label="agent inspect · mcp-token">
      <div className="rounded-lg border border-border bg-surface-2/50 px-3 py-2 font-mono text-[10px] text-faint">
        eyJhbGciOiJSUzI1NiIsImtpZCI6…<span className="text-muted">.claims.</span>…sig
      </div>
      <div className="mt-3 space-y-1.5">
        {findings.map((f) => (
          <div key={f.id} className="flex items-start gap-2.5 rounded-md border border-border/70 px-2.5 py-1.5">
            <span className={`mt-1 h-1.5 w-1.5 shrink-0 rounded-full ${sevDot[f.tone]}`} />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="font-mono text-[10px] font-semibold text-muted">{f.id}</span>
                <span className="truncate text-caption text-text">{f.title}</span>
              </div>
              <div className="truncate font-mono text-[10px] text-faint">{f.detail}</div>
            </div>
          </div>
        ))}
      </div>
    </MockFrame>
  );
}
