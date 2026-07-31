import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { BrandMark } from "../components/MarketingChrome";
import { Button } from "../components/ui";

const steps = [
  {
    n: "1",
    title: "Run it — zero config",
    body: "One container with the web app embedded. Boots on an in-memory store, no database needed. From source, use “make demo”.",
    code: "docker run -p 8080:8080 ghcr.io/nometria/keyway",
  },
  {
    n: "2",
    title: "Open the dashboard",
    body: "The UI loads on sample data — explore findings, coverage, blast radius and the agent inspector.",
    code: "open http://localhost:8080",
  },
  {
    n: "3",
    title: "Connect your own services",
    body: "Point Keyway at your Istio / Envoy / Kubernetes configs and a Postgres store. See the README quickstart for the full flow.",
    code: "keyway discover --path ./manifests   # set KEYWAY_DB_URL to persist",
  },
];

export default function Signup() {
  const navigate = useNavigate();
  const [copied, setCopied] = useState("");

  function copy(text: string) {
    navigator.clipboard?.writeText(text).then(() => {
      setCopied(text);
      window.setTimeout(() => setCopied(""), 1500);
    });
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-3xl flex-col px-5 py-10 sm:py-16">
      <BrandMark />
      <div className="mt-10">
        <span className="eyebrow">Get started</span>
        <h1 className="mt-2 text-h1 font-semibold tracking-tight">Keyway is self-hosted, and free.</h1>
        <p className="mt-2 max-w-xl text-body-lg text-muted">
          No account to create — run your own instance and connect to it. It takes about a minute.
        </p>
      </div>

      <ol className="mt-10 space-y-4">
        {steps.map((s) => (
          <li key={s.n} className="rounded-xl border border-border bg-surface p-5 shadow-xs">
            <div className="flex items-start gap-4">
              <span className="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-accent-soft text-body font-semibold text-accent">
                {s.n}
              </span>
              <div className="min-w-0 flex-1">
                <h3 className="text-[1.05rem] font-semibold tracking-tight">{s.title}</h3>
                <p className="mt-1 text-body text-muted">{s.body}</p>
                <button
                  onClick={() => copy(s.code)}
                  className="group mt-3 flex w-full items-center justify-between gap-3 rounded-md border border-border bg-surface-2 px-3 py-2 text-left font-mono text-caption text-text transition hover:border-border-strong"
                >
                  <span className="truncate">{s.code}</span>
                  <span className="shrink-0 text-faint transition group-hover:text-muted">
                    {copied === s.code ? "copied" : "copy"}
                  </span>
                </button>
              </div>
            </div>
          </li>
        ))}
      </ol>

      <div className="mt-10 flex flex-col items-center gap-3 rounded-xl border border-border bg-surface-2/50 p-6 text-center">
        <p className="text-body text-muted">Want to look around first?</p>
        <Button
          variant="primary"
          onClick={() => {
            localStorage.removeItem("keyway.live");
            navigate("/app");
          }}
        >
          Explore the demo
        </Button>
        <p className="text-caption text-muted">
          Already running one?{" "}
          <Link to="/login" className="font-medium text-accent hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
