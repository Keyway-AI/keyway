import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { agentInspect } from "../api/mock";
import { severityColor } from "../lib/format";
import type { AgentFinding } from "../api/types";

/**
 * Interactive hero widget: paste a JWT / agent token and see what a correct
 * verifier should reject. The analysis is the real agent-auth check set
 * (audience binding, delegation, scope, expiry — each cited to its spec) and
 * runs **client-side** on the marketing site, so nothing leaves the browser.
 * It is the same logic `keyway agent inspect` / `POST /v1/agent/inspect` run.
 */

const AUDIENCE = "https://mcp.example/api";

const now = () => Math.floor(Date.now() / 1000);
const b64u = (o: unknown) =>
  btoa(JSON.stringify(o)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
const jwt = (header: unknown, claims: unknown) => `${b64u(header)}.${b64u(claims)}.`;

/**
 * Three real agent-token scenarios the visitor can click through and watch the
 * checks react to — the tool does the talking. Each is a genuine JWT the
 * client-side analysis reads; the tokens are crafted so the findings are exact.
 */
type Scenario = { key: string; label: string; note: string; build: () => string };
const SCENARIOS: Scenario[] = [
  {
    key: "overscoped",
    label: "Over-scoped",
    note: "4 findings",
    // alg=none, omnibus scope, no audience, no expiry — shows its teeth on arrival.
    build: () =>
      jwt(
        { alg: "none", typ: "JWT" },
        { iss: "https://issuer.example", sub: "agent-7", scope: "admin:* files:read", iat: now() },
      ),
  },
  {
    key: "passthrough",
    label: "Passthrough",
    note: "1 critical",
    // Well-formed, but bound to a different resource — the marquee MCP bug: a
    // token minted for payments, replayed against the MCP server.
    build: () =>
      jwt(
        { alg: "RS256", typ: "JWT", kid: "k1" },
        {
          iss: "https://issuer.example",
          sub: "agent-7",
          aud: "https://payments.internal/api",
          act: { sub: "svc-gateway" },
          scope: "files:read",
          iat: now(),
          exp: now() + 3600,
        },
      ),
  },
  {
    key: "clean",
    label: "Clean",
    note: "passes",
    // Bound to this resource, a verifiable delegation chain, least-privilege
    // scope, a real expiry — the green state, so it isn't just a red-flag machine.
    build: () =>
      jwt(
        { alg: "RS256", typ: "JWT", kid: "k1" },
        {
          iss: "https://issuer.example",
          sub: "agent-7",
          aud: [AUDIENCE],
          act: { sub: "user-42" },
          scope: "files:read calendar:read",
          iat: now(),
          exp: now() + 900,
        },
      ),
  },
];

export function TokenInspector() {
  const [token, setToken] = useState(SCENARIOS[0].build);
  const [active, setActive] = useState<string>(SCENARIOS[0].key);
  const [findings, setFindings] = useState<AgentFinding[] | null>(null);
  const [scanning, setScanning] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // The analysis runs entirely in the browser (the same check `keyway agent
  // inspect` runs). Live typing resolves instantly; an explicit inspect (mount,
  // the button, "load sample") plays a brief scan so you see the tool work.
  const inspect = useCallback((value: string, animate = false) => {
    if (timer.current) clearTimeout(timer.current);
    const v = value.trim();
    if (!v) {
      setScanning(false);
      setFindings(null);
      return;
    }
    const compute = () => setFindings(agentInspect({ token: v, audience: AUDIENCE, require_delegation: true }).findings);
    if (!animate) {
      setScanning(false);
      compute();
      return;
    }
    setScanning(true);
    setFindings(null);
    timer.current = setTimeout(() => {
      compute();
      setScanning(false);
    }, 620);
  }, []);

  // Analyze the sample on mount, with the scan, so the widget is alive on arrival.
  useEffect(() => {
    inspect(token, true);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
    // run once
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="glass rounded-2xl p-4 shadow-lg sm:p-5">
      <div className="mb-2.5 flex items-center justify-between">
        <div className="flex items-center gap-2 text-caption font-medium text-muted">
          <span className="grid h-6 w-6 place-items-center rounded-md bg-accent-soft text-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="7" />
              <path d="m20 20-3.5-3.5" />
            </svg>
          </span>
          Token inspector
        </div>
        <span className="rounded-pill bg-surface-2 px-2 py-0.5 text-micro text-faint">runs in your browser</span>
      </div>

      <textarea
        value={token}
        onChange={(e) => {
          setActive("");
          setToken(e.target.value);
          inspect(e.target.value);
        }}
        spellCheck={false}
        rows={3}
        placeholder="Paste a JWT or agent token — header.payload.signature"
        className="w-full resize-none rounded-lg border border-border bg-surface-2 p-3 font-mono text-[0.72rem] leading-relaxed text-text outline-none transition placeholder:text-faint focus:border-accent"
      />

      <div className="mt-2.5 flex items-center gap-1.5">
        <span className="mr-0.5 text-micro text-faint">Try</span>
        {SCENARIOS.map((s) => (
          <button
            key={s.key}
            onClick={() => {
              const t = s.build();
              setActive(s.key);
              setToken(t);
              inspect(t, true);
            }}
            title={`${s.label} — ${s.note}`}
            className={`rounded-pill border px-2.5 py-1 text-micro font-medium transition ${
              active === s.key
                ? "border-accent bg-accent-soft text-accent"
                : "border-border bg-surface text-muted hover:bg-surface-2 hover:text-text"
            }`}
          >
            {s.label}
          </button>
        ))}
        <button
          onClick={() => inspect(token, true)}
          disabled={!token.trim() || scanning}
          className="ml-auto inline-flex h-7 items-center justify-center rounded-md bg-accent px-3 text-micro font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98] disabled:opacity-50"
        >
          {scanning ? "Analyzing…" : "Inspect"}
        </button>
      </div>

      {scanning && (
        <div className="mt-3 border-t border-border/70 pt-3">
          <div className="h-1 w-full overflow-hidden rounded-full bg-surface-2">
            <div className="scan-bar h-full w-1/3 rounded-full bg-accent" />
          </div>
          <div className="mt-2.5 text-micro font-medium text-faint">Analyzing token against the agent-auth checks…</div>
        </div>
      )}

      {!scanning && findings !== null && (
        <div className="mt-3 border-t border-border/70 pt-3">
          {findings.length === 0 ? (
            <div className="finding-in flex items-center gap-2 rounded-lg bg-low/10 px-2.5 py-2 text-caption font-medium text-low">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
                <path d="m5 12 5 5L20 7" />
              </svg>
              No findings — this token passes every agent-auth check.
            </div>
          ) : (
            <>
              <div className="mb-2 text-micro font-medium text-faint">
                {findings.length} finding{findings.length === 1 ? "" : "s"} · checked against{" "}
                <span className="font-mono">{AUDIENCE}</span>
              </div>
              <ul className="space-y-1.5">
                {findings.map((f, i) => (
                  <li
                    key={i}
                    className="finding-in flex items-start gap-2.5 rounded-lg bg-surface-2/70 px-2.5 py-2"
                    style={{ animationDelay: `${i * 70}ms` }}
                  >
                    <span
                      className={`mt-0.5 shrink-0 rounded-pill border px-1.5 py-0.5 text-[11px] font-semibold uppercase ${severityColor[f.severity]}`}
                    >
                      {f.severity}
                    </span>
                    <div className="min-w-0">
                      <span className="text-caption text-text">{f.message}</span>
                      {f.threat_id !== "—" && (
                        <span className="ml-1.5 font-mono text-[11px] text-faint">{f.threat_id}</span>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            </>
          )}
          <p className="mt-2.5 text-micro text-faint">
            Same checks as{" "}
            <code className="font-mono">keyway agent inspect</code> ·{" "}
            <Link to="/app/agent" className="text-accent hover:underline">
              open the full inspector
            </Link>
          </p>
        </div>
      )}
    </div>
  );
}
