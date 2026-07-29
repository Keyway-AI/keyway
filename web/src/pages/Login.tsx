import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { getToken, setToken as persistToken } from "../api/client";
import { BrandMark } from "../components/MarketingChrome";
import { Button, Field, Input } from "../components/ui";

/**
 * Keyway is self-hosted and authenticates with a bearer API token against your
 * instance. "Signing in" is connecting to that instance (or exploring the built-in
 * demo). No accounts, no passwords to store — honest for an open-source tool.
 */
function AuthPanel({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen">
      {/* brand panel */}
      <div className="relative hidden w-1/2 flex-col justify-between overflow-hidden bg-accent p-10 text-accent-fg lg:flex">
        <div
          aria-hidden
          className="pointer-events-none absolute -right-24 -top-24 h-96 w-96 rounded-full opacity-30 blur-3xl"
          style={{ background: "radial-gradient(closest-side, #fff, transparent)" }}
        />
        <Link to="/" className="flex items-center gap-2.5">
          <span className="grid h-7 w-7 place-items-center rounded-lg bg-white/15 text-[0.95rem]">🔑</span>
          <span className="text-[1.05rem] font-semibold tracking-tight">Keyway</span>
        </Link>
        <div className="relative">
          <h2 className="max-w-sm text-h2 font-semibold leading-tight tracking-tight">
            Know your token contracts hold — before they break.
          </h2>
          <p className="mt-3 max-w-sm text-body-lg text-accent-fg/80">
            Verify and adversarially test JWT and agent-auth, for classic services and AI agents alike.
          </p>
        </div>
        <div className="relative text-caption text-accent-fg/70">Open source · MIT licensed</div>
      </div>

      {/* form panel */}
      <div className="flex flex-1 flex-col items-center justify-center px-5 py-10">
        <div className="w-full max-w-sm">
          <div className="lg:hidden">
            <BrandMark />
          </div>
          <h1 className="mt-6 text-h1 font-semibold tracking-tight lg:mt-0">{title}</h1>
          <p className="mt-1.5 text-body text-muted">{subtitle}</p>
          <div className="mt-8">{children}</div>
        </div>
      </div>
    </div>
  );
}

export default function Login() {
  const navigate = useNavigate();
  const [token, setToken] = useState(getToken());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function connect() {
    setBusy(true);
    setError("");
    persistToken(token.trim());
    localStorage.setItem("keyway.live", "1");
    try {
      const res = await fetch("/v1/health", { headers: { Authorization: `Bearer ${token.trim()}` } });
      if (!res.ok) throw new Error(res.status === 401 ? "That token was rejected." : `Instance returned ${res.status}.`);
      navigate("/app");
    } catch {
      setError("Couldn't reach a Keyway instance with that token. Check the token, or explore the demo below.");
    } finally {
      setBusy(false);
    }
  }

  function demo() {
    localStorage.removeItem("keyway.live");
    navigate("/app");
  }

  return (
    <AuthPanel title="Sign in" subtitle="Connect to your Keyway instance with its API token.">
      <div className="space-y-4">
        <Field label="API token" hint="Set on your instance via KEYWAY_API_TOKEN.">
          <Input
            type="password"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="paste your API token"
            autoComplete="off"
            onKeyDown={(e) => e.key === "Enter" && token.trim() && connect()}
          />
        </Field>
        {error && (
          <div className="rounded-md border border-critical/30 bg-critical/8 px-3 py-2 text-caption text-critical">
            {error}
          </div>
        )}
        <Button variant="primary" className="w-full" onClick={connect} loading={busy} disabled={!token.trim()}>
          Connect
        </Button>
        <div className="flex items-center gap-3 py-1 text-caption text-faint">
          <span className="h-px flex-1 bg-border" />
          or
          <span className="h-px flex-1 bg-border" />
        </div>
        <Button variant="secondary" className="w-full" onClick={demo}>
          Explore the demo
        </Button>
        <p className="pt-2 text-center text-caption text-muted">
          No instance yet?{" "}
          <Link to="/signup" className="font-medium text-accent hover:underline">
            Get started
          </Link>
        </p>
      </div>
    </AuthPanel>
  );
}
