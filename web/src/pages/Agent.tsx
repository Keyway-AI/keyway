import { useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Button, Card, Empty, Field, Input } from "../components/ui";
import { useToast } from "../components/toast";
import { severityColor } from "../lib/format";
import type { AgentFinding } from "../api/types";

// A deliberately-flawed sample: no audience, an omnibus scope, and no expiry.
function sampleToken(): string {
  const b64 = (o: unknown) => btoa(JSON.stringify(o)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  const header = { alg: "none", typ: "JWT" };
  const claims = {
    iss: "https://issuer.example",
    sub: "agent-7",
    scope: "admin:* files:read",
    iat: Math.floor(Date.now() / 1000),
  };
  return `${b64(header)}.${b64(claims)}.`;
}

export default function Agent() {
  const toast = useToast();
  const [token, setToken] = useState("");
  const [audience, setAudience] = useState("https://mcp.example/api");
  const [requireDelegation, setRequireDelegation] = useState(true);
  const [maxLifetime, setMaxLifetime] = useState("1h");
  const [allowedScopes, setAllowedScopes] = useState("");
  const [busy, setBusy] = useState(false);
  const [findings, setFindings] = useState<AgentFinding[] | null>(null);

  function parseDurationSec(s: string): number | undefined {
    const m = s.trim().match(/^(\d+)\s*([smhd])$/);
    if (!m) return undefined;
    const n = Number(m[1]);
    return n * { s: 1, m: 60, h: 3600, d: 86400 }[m[2] as "s" | "m" | "h" | "d"];
  }

  async function inspect() {
    if (!token.trim()) {
      toast.error("Paste a token to inspect");
      return;
    }
    setBusy(true);
    try {
      const res = await api.agentInspect({
        token: token.trim(),
        audience: audience.trim() || undefined,
        require_delegation: requireDelegation,
        max_lifetime_seconds: parseDurationSec(maxLifetime),
        allowed_scopes: allowedScopes
          .split(/[,\s]+/)
          .map((s) => s.trim())
          .filter(Boolean),
      });
      setFindings(res.findings);
      if (res.count === 0) toast.success("No agent-auth findings");
      else toast.error(`${res.count} finding(s)`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Inspection failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Page
      title="Agent token inspector"
      subtitle="Statically check an agent / MCP / on-behalf-of token against the agent-auth invariants — audience binding, delegation, scope minimization, and expiry. No signature verification; just the token's own hygiene."
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1.1fr_1fr]">
        <Card title="Token">
          <div className="space-y-4">
            <div>
              <div className="mb-1.5 flex items-center justify-between">
                <span className="text-caption font-medium text-muted">JWT</span>
                <button
                  onClick={() => setToken(sampleToken())}
                  className="text-caption font-medium text-accent hover:underline"
                >
                  Load sample
                </button>
              </div>
              <textarea
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="eyJhbGciOi…"
                spellCheck={false}
                rows={5}
                className="w-full resize-y rounded-md border border-border bg-surface-2 px-3.5 py-2.5 font-mono text-caption text-text outline-none transition placeholder:text-faint hover:border-border-strong focus:border-accent"
              />
            </div>
            <Field label="Expected audience (resource URI)" hint="RFC 8707/9728 — the token's aud must be bound to this.">
              <Input value={audience} onChange={(e) => setAudience(e.target.value)} className="font-mono" />
            </Field>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="Max lifetime" hint="e.g. 1h, 30m">
                <Input value={maxLifetime} onChange={(e) => setMaxLifetime(e.target.value)} />
              </Field>
              <Field label="Allowed scopes" hint="comma-separated; else flagged">
                <Input value={allowedScopes} onChange={(e) => setAllowedScopes(e.target.value)} className="font-mono" />
              </Field>
            </div>
            <label className="flex cursor-pointer items-center gap-2.5">
              <input
                type="checkbox"
                checked={requireDelegation}
                onChange={(e) => setRequireDelegation(e.target.checked)}
                className="h-4 w-4 accent-[var(--color-accent)]"
              />
              <span className="text-body">Require delegation (on-behalf-of tokens must carry <code className="font-mono text-caption">act</code>)</span>
            </label>
            <Button variant="primary" onClick={inspect} loading={busy}>
              Inspect token
            </Button>
          </div>
        </Card>

        <Card title="Findings">
          {findings === null ? (
            <Empty>Paste a token and press Inspect. Findings map to the agent threats in Coverage.</Empty>
          ) : findings.length === 0 ? (
            <div className="flex items-center gap-2 py-6 text-body text-low">
              <span className="grid h-6 w-6 place-items-center rounded-full bg-low/12 text-low">✓</span>
              No agent-auth findings for this token.
            </div>
          ) : (
            <ul className="space-y-2.5">
              {findings.map((f, i) => (
                <li key={i} className="rounded-lg border border-border bg-surface-2 p-3.5">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-caption text-muted">{f.threat_id}</span>
                    <span className={`text-caption font-semibold capitalize ${severityColor[f.severity]}`}>
                      {f.severity}
                    </span>
                  </div>
                  <div className="mt-1 text-body text-text">{f.message}</div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </Page>
  );
}
