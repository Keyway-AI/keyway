import { useState } from "react";
import { Link } from "react-router-dom";
import { cloud } from "./api";
import { useCloudAuth } from "./CloudAuth";
import { LogoBadge, LogoGlyph } from "../components/Logo";
import { Button } from "../components/ui";
import { ThemeToggle } from "../components/ThemeToggle";

/**
 * The Cloud front door. GitHub OAuth is a full-page redirect (so the session
 * cookie is set first-party on the API origin); dev-login is a passwordless local
 * shortcut the server only enables with KEYWAY_CLOUD_DEV_LOGIN=1. The luminous
 * ground is the swishX-inspired premium treatment.
 */
export default function CloudSignIn() {
  const { config, error, refresh } = useCloudAuth();
  const [busy, setBusy] = useState(false);
  const [devErr, setDevErr] = useState("");

  const apiReachable = config !== null;

  async function devLogin() {
    setBusy(true);
    setDevErr("");
    try {
      await cloud.devLogin();
      await refresh();
    } catch {
      setDevErr("Dev login failed. Is the Cloud API running with KEYWAY_CLOUD_DEV_LOGIN=1?");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="aurora flex min-h-screen flex-col">
      <header className="mx-auto flex w-full max-w-6xl items-center justify-between px-5 py-5 sm:px-8">
        <Link to="/" className="flex items-center gap-2.5">
          <LogoBadge size={28} />
          <span className="text-[1.05rem] font-semibold tracking-[-0.02em]">Keyway</span>
          <span className="rounded-pill border border-border bg-surface/70 px-2 py-0.5 text-micro font-semibold uppercase tracking-wider text-muted">
            Cloud
          </span>
        </Link>
        <ThemeToggle />
      </header>

      <main className="flex flex-1 items-center justify-center px-5 py-10">
        <div className="w-full max-w-md">
          <div className="glass rounded-2xl p-8 shadow-lg">
            <div className="mb-6 flex items-center gap-3">
              <span className="grid h-11 w-11 place-items-center rounded-xl bg-accent text-accent-fg glow-accent">
                <LogoGlyph width="24" height="24" />
              </span>
              <div>
                <h1 className="text-h2 font-semibold tracking-tight">Keyway Cloud</h1>
                <p className="text-caption text-muted">Hosted auth-contract analysis for your repos.</p>
              </div>
            </div>

            {!apiReachable && (
              <div className="mb-4 rounded-md border border-high/30 bg-high/8 px-3 py-2.5 text-caption text-high">
                Can't reach the Cloud API{error ? ` (${error})` : ""}. Start it with{" "}
                <code className="rounded bg-surface-2 px-1 font-mono">make cloud</code> or set{" "}
                <code className="rounded bg-surface-2 px-1 font-mono">VITE_CLOUD_API_URL</code>.
              </div>
            )}

            <div className="space-y-3">
              {config?.github_login && (
                <a
                  href={cloud.githubLoginUrl()}
                  className="glow-accent inline-flex h-11 w-full items-center justify-center gap-2.5 rounded-md bg-accent px-4 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.99]"
                >
                  <GitHubMark />
                  Continue with GitHub
                </a>
              )}

              {config?.dev_login && (
                <Button
                  variant={config?.github_login ? "secondary" : "primary"}
                  className={`h-11 w-full ${config?.github_login ? "" : "glow-accent"}`}
                  onClick={devLogin}
                  loading={busy}
                >
                  Continue with dev login
                </Button>
              )}

              {apiReachable && !config?.github_login && !config?.dev_login && (
                <div className="rounded-md border border-border bg-surface-2 px-3 py-2.5 text-caption text-muted">
                  No sign-in method is enabled on this server. Configure{" "}
                  <code className="font-mono">GITHUB_CLIENT_ID</code>/<code className="font-mono">SECRET</code>,
                  or set <code className="font-mono">KEYWAY_CLOUD_DEV_LOGIN=1</code> for local use.
                </div>
              )}

              {devErr && <p className="text-caption text-critical">{devErr}</p>}
            </div>

            <p className="mt-6 text-caption leading-relaxed text-muted">
              Cloud runs the <span className="font-medium text-text">static</span> half of Keyway — discovery,
              contract drift, and threat coverage on the repos you connect. Live probing and canary keys stay in
              your own environment, so Keyway never handles your signing keys.
            </p>
          </div>

          <p className="mt-5 text-center text-caption text-muted">
            Prefer self-hosting?{" "}
            <Link to="/app" className="font-medium text-accent hover:underline">
              Open the local app
            </Link>
          </p>
        </div>
      </main>
    </div>
  );
}

function GitHubMark() {
  return (
    <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor" aria-hidden>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
    </svg>
  );
}
