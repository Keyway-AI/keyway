import { Link } from "react-router-dom";
import { ThemeToggle } from "../components/ThemeToggle";
import { IconAgent, IconBlast, IconCoverage, IconFindings, IconProbes, IconChanges } from "../components/icons";

const features = [
  {
    icon: IconChanges,
    title: "Contract, versioned",
    body: "Discover what every service expects from a JWT — issuers, audiences, algorithms, claims — and snapshot it as a hashed contract you can diff over time.",
  },
  {
    icon: IconFindings,
    title: "Drift, caught early",
    body: "Every change is classified and attributed: a widened audience, a dropped required claim, a narrowed JWKS cache — before it becomes an incident.",
  },
  {
    icon: IconProbes,
    title: "Attacked, generatively",
    body: "A taxonomy-driven harness fires alg=none, key confusion, jku injection, and forged claims at your endpoints — and proves a correct verifier rejects them.",
  },
  {
    icon: IconBlast,
    title: "Blast radius, before you rotate",
    body: "Model a key rotation, a retired claim, or a new issuer and see exactly which consumers break — with a safe grace period derived from real behavior.",
  },
  {
    icon: IconCoverage,
    title: "Coverage, kept honest",
    body: "Detection is measured against the documented universe of threats — RFC 8725, OWASP, CVEs — not a corpus we wrote. Every gap is named and cited.",
  },
  {
    icon: IconAgent,
    title: "Agent auth, verified",
    body: "The same model, pointed at AI agents: MCP token passthrough, missing delegation, over-scoped and non-expiring agent credentials — the layer nobody else tests.",
  },
];

function Nav() {
  return (
    <header className="sticky top-0 z-30 border-b border-border/70 bg-bg/80 backdrop-blur-md">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-5 py-3.5 sm:px-8">
        <div className="flex items-center gap-2.5">
          <span className="grid h-7 w-7 place-items-center rounded-lg bg-accent text-[0.95rem] text-accent-fg shadow-xs">🔑</span>
          <span className="text-[1.05rem] font-semibold tracking-tight">Keyway</span>
        </div>
        <div className="flex items-center gap-3">
          <ThemeToggle />
          <Link
            to="/app"
            className="inline-flex h-9 items-center rounded-md bg-accent px-4 text-body font-medium text-accent-fg shadow-xs transition hover:bg-accent-strong active:scale-[0.98]"
          >
            Open app
          </Link>
        </div>
      </div>
    </header>
  );
}

function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-border">
      {/* soft ambient accent wash */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 -top-40 mx-auto h-80 max-w-3xl rounded-full opacity-60 blur-3xl"
        style={{ background: "radial-gradient(closest-side, color-mix(in srgb, var(--color-accent) 22%, transparent), transparent)" }}
      />
      <div className="mx-auto max-w-6xl px-5 py-20 text-center sm:px-8 sm:py-28">
        <span className="inline-flex items-center gap-2 rounded-pill border border-border bg-surface px-3 py-1 text-caption font-medium text-muted shadow-xs">
          <span className="h-1.5 w-1.5 rounded-full bg-low" />
          Open-source · JWT &amp; agent-auth verification
        </span>
        <h1 className="mx-auto mt-6 max-w-3xl text-[2.5rem] font-semibold leading-[1.05] tracking-tight sm:text-display">
          Know your token contracts hold —{" "}
          <span className="text-accent">before</span> they break.
        </h1>
        <p className="mx-auto mt-5 max-w-xl text-body-lg text-muted">
          Keyway discovers what your services expect from JWTs, versions it as a contract, and adversarially tests that a correct verifier is enforced — for classic services and AI agents alike.
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <Link
            to="/app"
            className="inline-flex h-11 items-center rounded-md bg-accent px-5 text-body font-medium text-accent-fg shadow-sm transition hover:bg-accent-strong active:scale-[0.98]"
          >
            Open the app
          </Link>
          <a
            href="https://github.com/nometria/keyway"
            className="inline-flex h-11 items-center rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
          >
            View on GitHub
          </a>
        </div>
      </div>
    </section>
  );
}

function Stats() {
  const stats = [
    { n: "50", l: "documented threats, all cited" },
    { n: "60%", l: "JWT coverage, gaps named" },
    { n: "40%", l: "agent-auth coverage, and rising" },
    { n: "0", l: "corpus we grade ourselves on" },
  ];
  return (
    <section className="border-b border-border bg-surface-2">
      <div className="mx-auto grid max-w-6xl grid-cols-2 gap-px overflow-hidden px-5 py-12 sm:grid-cols-4 sm:px-8">
        {stats.map((s) => (
          <div key={s.l} className="px-2 text-center">
            <div className="text-h1 font-semibold tracking-tight tabular-nums text-accent">{s.n}</div>
            <div className="mt-1 text-caption text-muted">{s.l}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

function Features() {
  return (
    <section className="mx-auto max-w-6xl px-5 py-20 sm:px-8">
      <div className="max-w-2xl">
        <div className="eyebrow">What it does</div>
        <h2 className="mt-2 text-h2 font-semibold tracking-tight">
          Verification, not another gateway.
        </h2>
        <p className="mt-3 text-body-lg text-muted">
          The market issues and enforces auth. Keyway proves it&apos;s correct — the part everyone else assumes.
        </p>
      </div>
      <div className="mt-10 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {features.map((f) => (
          <div key={f.title} className="rounded-xl border border-border bg-surface p-6 shadow-xs">
            <span className="grid h-10 w-10 place-items-center rounded-lg bg-accent-soft text-accent">
              <f.icon />
            </span>
            <h3 className="mt-4 text-[1.05rem] font-semibold tracking-tight">{f.title}</h3>
            <p className="mt-2 text-body text-muted">{f.body}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

function CTA() {
  return (
    <section className="border-t border-border">
      <div className="mx-auto max-w-6xl px-5 py-20 text-center sm:px-8">
        <h2 className="mx-auto max-w-xl text-h1 font-semibold tracking-tight">
          See it on your own contracts.
        </h2>
        <p className="mx-auto mt-3 max-w-md text-body-lg text-muted">
          The app runs on sample data out of the box — no backend required.
        </p>
        <Link
          to="/app"
          className="mt-7 inline-flex h-11 items-center rounded-md bg-accent px-6 text-body font-medium text-accent-fg shadow-sm transition hover:bg-accent-strong active:scale-[0.98]"
        >
          Open the app
        </Link>
      </div>
    </section>
  );
}

export default function Marketing() {
  return (
    <div className="min-h-screen">
      <Nav />
      <Hero />
      <Stats />
      <Features />
      <CTA />
      <footer className="border-t border-border">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-3 px-5 py-8 text-caption text-faint sm:flex-row sm:px-8">
          <span>Keyway — contract verification for JWT &amp; agent auth.</span>
          <span>Open source · MIT</span>
        </div>
      </footer>
    </div>
  );
}
