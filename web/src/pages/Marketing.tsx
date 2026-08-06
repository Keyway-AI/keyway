import { Link } from "react-router-dom";
import type { ComponentType } from "react";
import { MarketingShell } from "../components/MarketingChrome";
import { AppPreview } from "../components/AppPreview";
import { AgentMock, BlastMock, CoverageMock, DiffMock, HarnessMock } from "../components/FeatureMocks";
import { IconAgent, IconBlast, IconCoverage, IconFindings, IconProbes, IconChanges } from "../components/icons";

const GITHUB = "https://github.com/Keyway-AI/keyway";

/* ── Hero ─────────────────────────────────────────────────────────────── */
function Hero() {
  return (
    <section className="aurora relative overflow-hidden border-b border-border">
      <div className="mx-auto max-w-6xl px-5 pt-16 text-center sm:px-8 sm:pt-24">
        <a
          href={GITHUB}
          className="glass inline-flex items-center gap-2 rounded-pill px-3.5 py-1.5 text-caption font-medium text-muted shadow-sm transition hover:text-text"
        >
          <span className="h-1.5 w-1.5 rounded-full bg-low" />
          Open-source · JWT &amp; agent-auth verification
          <span className="text-faint">→</span>
        </a>
        <h1 className="display mx-auto mt-6 max-w-3xl text-[2.75rem] leading-[1.02] sm:text-[3.5rem]">
          Know your token contracts hold — <span className="text-accent">before</span> they break.
        </h1>
        <p className="mx-auto mt-5 max-w-xl text-body-lg text-muted">
          Keyway discovers what your services expect from JWTs, versions it as a contract, and adversarially tests that a correct verifier is enforced — for classic services and AI agents alike.
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <Link
            to="/signup"
            className="glow-accent inline-flex h-11 items-center rounded-md bg-accent px-5 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
          >
            Get started — free
          </Link>
          <a
            href={GITHUB}
            className="inline-flex h-11 items-center gap-2 rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
          >
            <GitHubIcon />
            Star on GitHub
          </a>
        </div>
        <div className="mx-auto mt-14 max-w-4xl pb-16 sm:mt-16">
          <AppPreview />
        </div>
      </div>
    </section>
  );
}

/* ── Standards strip: the honest denominator, not fake customer logos ──── */
function StandardsStrip() {
  const standards = ["RFC 8725 · JWT BCP", "RFC 8693 · Token Exchange", "RFC 8707 · Resource Indicators", "MCP Authorization", "OWASP LLM & Agentic Top 10", "OWASP JWT"];
  return (
    <section className="border-b border-border bg-surface-2/40">
      <div className="mx-auto max-w-6xl px-5 py-8 sm:px-8">
        <p className="text-center text-caption text-muted">
          Grounded in the specs that define correct auth — every detection cites its source
        </p>
        <div className="mt-4 flex flex-wrap items-center justify-center gap-x-6 gap-y-2">
          {standards.map((s) => (
            <span key={s} className="text-caption font-medium text-faint">
              {s}
            </span>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ── Stats band ───────────────────────────────────────────────────────── */
function Stats() {
  const stats = [
    { n: "50", l: "documented threats, all cited", ref: "1" },
    { n: "60%", l: "JWT coverage, gaps named" },
    { n: "40%", l: "agent-auth coverage, and rising" },
    { n: "0", l: "corpus we grade ourselves on" },
  ];
  return (
    <section className="border-b border-border">
      <div className="mx-auto grid max-w-6xl grid-cols-2 gap-8 px-5 py-12 sm:grid-cols-4 sm:px-8">
        {stats.map((s) => (
          <div key={s.l} className="text-center">
            <div className="text-h1 font-semibold tracking-tight tabular-nums text-accent">
              {s.n}
              {s.ref && <sup className="footnote-ref">{s.ref}</sup>}
            </div>
            <div className="mt-1 text-caption text-muted">{s.l}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

/* ── Alternating feature sections ─────────────────────────────────────── */
type Section = {
  eyebrow: string;
  title: string;
  body: string;
  points: string[];
  mock: ComponentType;
  flip?: boolean;
};

const sections: Section[] = [
  {
    eyebrow: "Contract",
    title: "Every token contract, versioned and diffed.",
    body: "Keyway discovers what each service expects from a JWT — issuers, audiences, algorithms, required claims, JWKS behavior — and snapshots it as a hashed contract. Every change is classified, attributed, and graded by severity.",
    points: ["Auto-discovered from configs and live behavior", "Hashed snapshots you can diff across versions", "Breaking changes flagged before they ship"],
    mock: DiffMock,
  },
  {
    eyebrow: "Adversarial testing",
    title: "Attack your own endpoints, generatively.",
    body: "A taxonomy-driven harness mints real tokens for alg=none, key confusion, jku/kid injection, expiry and forged-claim attacks, and fires them at your staging endpoints — proving a correct verifier rejects every one.",
    points: ["Generated from invariants, not a fixed checklist", "Deny-by-default: staging allowlist, never production", "An accepted attack is a confirmed live vulnerability"],
    mock: HarnessMock,
    flip: true,
  },
  {
    eyebrow: "Change safety",
    title: "Know exactly who breaks before you rotate.",
    body: "Model a key rotation, a retired claim, or a new issuer and Keyway shows precisely which consumers would fail — with a safe grace period derived from their observed JWKS-refresh behavior.",
    points: ["Simulate rotations without touching production", "Per-consumer break / safe verdict with reasons", "Grace windows measured, not guessed"],
    mock: BlastMock,
  },
  {
    eyebrow: "Honest coverage",
    title: "Measured against every known threat — gaps and all.",
    body: "Detection is scored against the documented universe of threats — RFC 8725, OWASP, published CVEs — not a corpus we wrote ourselves. Every gap is named, cited, and on the roadmap. No security theater.",
    points: ["The denominator is the real threat taxonomy", "Per-domain and per-category breakdowns", "Each threat links to its normative source"],
    mock: CoverageMock,
    flip: true,
  },
  {
    eyebrow: "Agent auth",
    title: "The auth layer nobody else verifies.",
    body: "The same model pointed at AI agents: MCP token passthrough, missing or forged delegation, over-scoped and non-expiring agent credentials. Paste an agent token and get cited findings against the MCP spec and OAuth RFCs.",
    points: ["MCP audience binding & token passthrough", "On-behalf-of delegation (act / may_act) checks", "Least-privilege and expiry enforcement"],
    mock: AgentMock,
  },
];

function FeatureSection({ s }: { s: Section }) {
  const Mock = s.mock;
  return (
    <div className="grid grid-cols-1 items-center gap-10 lg:grid-cols-2 lg:gap-16">
      <div className={s.flip ? "lg:order-2" : ""}>
        <span className="eyebrow">{s.eyebrow}</span>
        <h3 className="mt-2 text-h2 font-semibold tracking-tight">{s.title}</h3>
        <p className="mt-3 text-body-lg text-muted">{s.body}</p>
        <ul className="mt-5 space-y-2.5">
          {s.points.map((p) => (
            <li key={p} className="flex items-start gap-2.5 text-body text-text">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" className="mt-0.5 shrink-0 text-accent">
                <path d="m5 12 5 5L20 7" />
              </svg>
              {p}
            </li>
          ))}
        </ul>
      </div>
      <div className={s.flip ? "lg:order-1" : ""}>
        <Mock />
      </div>
    </div>
  );
}

function FeatureSections() {
  return (
    <section className="mx-auto max-w-6xl px-5 py-20 sm:px-8">
      <div className="mx-auto max-w-2xl text-center">
        <div className="eyebrow">What it does</div>
        <h2 className="mt-2 text-h1 font-semibold tracking-tight">Verification, not another gateway.</h2>
        <p className="mt-3 text-body-lg text-muted">
          The market issues and enforces auth. Keyway proves it&apos;s correct — the part everyone else assumes.
        </p>
      </div>
      <div className="mt-16 space-y-20 sm:mt-20 sm:space-y-28">
        {sections.map((s) => (
          <FeatureSection key={s.title} s={s} />
        ))}
      </div>
    </section>
  );
}

/* ── Compact grid (reused by /features) ───────────────────────────────── */
const gridFeatures = [
  { icon: IconChanges, title: "Contract, versioned", body: "Discover what every service expects from a JWT and snapshot it as a hashed contract you can diff over time." },
  { icon: IconFindings, title: "Drift, caught early", body: "Every change is classified and attributed — a widened audience, a dropped claim — before it becomes an incident." },
  { icon: IconProbes, title: "Attacked, generatively", body: "A taxonomy-driven harness fires alg=none, key confusion, jku injection and forged claims at your endpoints." },
  { icon: IconBlast, title: "Blast radius", body: "Model a rotation and see exactly which consumers break — with a safe grace period derived from real behavior." },
  { icon: IconCoverage, title: "Coverage, kept honest", body: "Detection is measured against the documented universe of threats — RFC 8725, OWASP, CVEs. Every gap is cited." },
  { icon: IconAgent, title: "Agent auth, verified", body: "MCP token passthrough, missing delegation, over-scoped and non-expiring agent credentials — the layer nobody tests." },
];

export function FeatureGrid() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {gridFeatures.map((f) => (
        <div key={f.title} className="rounded-xl border border-border bg-surface p-6 shadow-xs">
          <span className="grid h-10 w-10 place-items-center rounded-lg bg-accent-soft text-accent">
            <f.icon />
          </span>
          <h3 className="mt-4 text-[1.05rem] font-semibold tracking-tight">{f.title}</h3>
          <p className="mt-2 text-body text-muted">{f.body}</p>
        </div>
      ))}
    </div>
  );
}

/* ── CTA ──────────────────────────────────────────────────────────────── */
function CTA() {
  return (
    <section className="border-t border-border">
      <div className="mx-auto max-w-6xl px-5 py-20 text-center sm:px-8">
        <h2 className="mx-auto max-w-xl text-h1 font-semibold tracking-tight">See it on your own contracts.</h2>
        <p className="mx-auto mt-3 max-w-md text-body-lg text-muted">
          The app runs on sample data out of the box — no backend required. Self-host the whole thing in one binary.
        </p>
        <div className="mt-7 flex flex-wrap items-center justify-center gap-3">
          <Link
            to="/signup"
            className="glow-accent inline-flex h-11 items-center rounded-md bg-accent px-6 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
          >
            Get started
          </Link>
          <a
            href={GITHUB}
            className="inline-flex h-11 items-center gap-2 rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
          >
            <GitHubIcon />
            View source
          </a>
          <Link
            to="/contact"
            className="inline-flex h-11 items-center rounded-md px-4 text-body font-medium text-muted transition hover:text-text"
          >
            Talk to us →
          </Link>
        </div>
      </div>
    </section>
  );
}

function GitHubIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
      <path d="M12 2C6.48 2 2 6.48 2 12c0 4.42 2.87 8.17 6.84 9.5.5.09.68-.22.68-.48v-1.7c-2.78.6-3.37-1.34-3.37-1.34-.45-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.6.07-.6 1 .07 1.53 1.03 1.53 1.03.89 1.53 2.34 1.09 2.91.83.09-.65.35-1.09.63-1.34-2.22-.25-4.55-1.11-4.55-4.94 0-1.09.39-1.98 1.03-2.68-.1-.25-.45-1.27.1-2.65 0 0 .84-.27 2.75 1.02a9.6 9.6 0 0 1 5 0c1.91-1.29 2.75-1.02 2.75-1.02.55 1.38.2 2.4.1 2.65.64.7 1.03 1.59 1.03 2.68 0 3.84-2.34 4.68-4.57 4.93.36.31.68.92.68 1.85v2.74c0 .27.18.58.69.48A10 10 0 0 0 22 12c0-5.52-4.48-10-10-10Z" />
    </svg>
  );
}

export default function Marketing() {
  return (
    <MarketingShell>
      <Hero />
      <StandardsStrip />
      <Stats />
      <FeatureSections />
      <CTA />
    </MarketingShell>
  );
}
