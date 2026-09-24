import { Link } from "react-router-dom";
import type { ComponentType } from "react";
import { MarketingShell } from "../components/MarketingChrome";
import { AppPreview } from "../components/AppPreview";
import { TokenInspector } from "../components/TokenInspector";
import { Reveal } from "../components/Reveal";
import { AgentMock, BlastMock, CoverageMock, DiffMock, HarnessMock } from "../components/FeatureMocks";
import { IconAgent, IconBlast, IconCoverage, IconFindings, IconProbes, IconChanges } from "../components/icons";

const GITHUB = "https://github.com/Keyway-AI/keyway";

/* ── Hero ─────────────────────────────────────────────────────────────── */
function Hero() {
  return (
    <section className="aurora grid-ground relative overflow-hidden border-b border-border">
      <div className="mx-auto grid max-w-6xl items-center gap-10 px-5 pb-16 pt-16 sm:px-8 sm:pt-24 lg:grid-cols-[1.05fr_0.95fr] lg:gap-14 lg:pb-24">
        <div className="text-center lg:text-left">
          <Reveal>
            <a
              href={GITHUB}
              className="glass inline-flex items-center gap-2 rounded-pill px-3.5 py-1.5 text-caption font-medium text-muted shadow-sm transition hover:text-text"
            >
              <span className="h-1.5 w-1.5 rounded-full bg-low" />
              Open source · MIT
              <span className="text-faint">→</span>
            </a>
          </Reveal>
          <Reveal delay={60}>
            <h1 className="display mt-6 text-[2.7rem] leading-[1.02] sm:text-[3.4rem]">
              Verify the auth<br className="hidden sm:block" /> everyone else <span className="text-accent">assumes.</span>
            </h1>
          </Reveal>
          <Reveal delay={120}>
            <p className="mx-auto mt-5 max-w-md text-body-lg text-muted lg:mx-0">
              Contract testing and adversarial verification for JWT and AI-agent auth.
            </p>
          </Reveal>
          <Reveal delay={180}>
            <div className="mt-8 flex flex-wrap items-center justify-center gap-3 lg:justify-start">
              <Link
                to="/signup"
                className="glow-accent inline-flex h-11 items-center rounded-md bg-accent px-5 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
              >
                Get started
              </Link>
              <a
                href={GITHUB}
                className="inline-flex h-11 items-center gap-2 rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
              >
                <GitHubIcon />
                GitHub
              </a>
            </div>
          </Reveal>
        </div>

        {/* interactive centerpiece — the product does the talking */}
        <Reveal delay={140} className="relative">
          <div
            aria-hidden
            className="drift pointer-events-none absolute -inset-6 -z-10 rounded-[2rem] bg-accent/10 blur-3xl"
          />
          <TokenInspector />
        </Reveal>
      </div>
    </section>
  );
}

/* ── Standards strip: the honest denominator, not fake customer logos ──── */
function StandardsStrip() {
  const standards = ["RFC 8725", "RFC 8693", "RFC 8707", "MCP Authorization", "OWASP Agentic Top 10", "OWASP JWT"];
  return (
    <section className="border-b border-border bg-surface-2/40">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-center gap-x-7 gap-y-2 px-5 py-6 sm:px-8">
        {standards.map((s) => (
          <span key={s} className="text-caption font-medium text-faint">
            {s}
          </span>
        ))}
      </div>
    </section>
  );
}

/* ── Proof band ───────────────────────────────────────────────────────────
 * Every number is measured and links to its source. The honesty is the point. */
const DOCS = {
  benchmark: `${GITHUB}/blob/main/BENCHMARK.md`,
  integrity: `${GITHUB}/blob/main/docs/benchmark-integrity.md`,
  coverage: `${GITHUB}/blob/main/docs/threat-coverage.md`,
};

function Proof() {
  const stats = [
    { n: "1,226", l: "config changes benchmarked", href: DOCS.benchmark },
    { n: "100%", l: "real changes caught", href: DOCS.benchmark },
    { n: "0%", l: "false alarms on redeploys", href: DOCS.benchmark },
    { n: "54%", l: "of 50 threats, gaps named", href: DOCS.coverage },
  ];
  return (
    <section className="border-b border-border bg-surface-2/30">
      <div className="mx-auto max-w-6xl px-5 py-14 sm:px-8">
        <div className="grid grid-cols-2 gap-6 sm:grid-cols-4">
          {stats.map((s, i) => (
            <Reveal key={s.l} delay={i * 70}>
              <a href={s.href} className="group block text-center">
                <div className="text-h1 font-semibold tracking-tight tabular-nums text-accent">{s.n}</div>
                <div className="mt-1 text-caption text-muted transition group-hover:text-text">{s.l}</div>
              </a>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ── Product showcase: real output, not a description ─────────────────────── */
function ProductShowcase() {
  return (
    <section className="border-b border-border">
      <div className="mx-auto max-w-6xl px-5 py-16 sm:px-8 sm:py-24">
        <Reveal>
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="display text-h1 font-semibold tracking-tight">Your whole auth surface, on one screen.</h2>
          </div>
        </Reveal>
        <Reveal delay={100}>
          <div className="relative mx-auto mt-12 max-w-4xl">
            <div
              aria-hidden
              className="pointer-events-none absolute -inset-x-8 -bottom-8 top-8 -z-10 rounded-[2rem] bg-gradient-to-b from-accent/10 to-transparent blur-2xl"
            />
            <div className="lift">
              <AppPreview />
            </div>
          </div>
        </Reveal>
      </div>
    </section>
  );
}

/* ── One sharp line of positioning ───────────────────────────────────────── */
function Thesis() {
  return (
    <section className="border-b border-border bg-surface-2/30">
      <div className="mx-auto max-w-3xl px-5 py-20 text-center sm:px-8 sm:py-28">
        <Reveal>
          <h2 className="display text-[2rem] font-semibold tracking-tight sm:text-[2.6rem]">
            A scanner guesses.<br />
            Keyway <span className="text-accent">proves</span> it.
          </h2>
          <p className="mx-auto mt-4 max-w-md text-body-lg text-muted">
            Auth correctness is a property of what&apos;s running — the part static tools can&apos;t see.
          </p>
        </Reveal>
      </div>
    </section>
  );
}

/* ── Alternating feature sections: a line + the real output ────────────── */
type Section = {
  eyebrow: string;
  title: string;
  line: string;
  mock: ComponentType;
  flip?: boolean;
};

const sections: Section[] = [
  {
    eyebrow: "Contract",
    title: "Versioned and diffed.",
    line: "Discovered from your configs, snapshotted as a hash, diffed on every change.",
    mock: DiffMock,
  },
  {
    eyebrow: "Adversarial testing",
    title: "Attack your own endpoints.",
    line: "Real forged tokens — alg=none, key confusion, jku injection — fired at staging.",
    mock: HarnessMock,
    flip: true,
  },
  {
    eyebrow: "Change safety",
    title: "Know who breaks before you rotate.",
    line: "Per-consumer blast radius, with a safe grace window measured from real behavior.",
    mock: BlastMock,
  },
  {
    eyebrow: "Coverage",
    title: "Measured against every known threat.",
    line: "Scored on real CVEs and RFCs, not a checklist we wrote. Every gap named.",
    mock: CoverageMock,
    flip: true,
  },
  {
    eyebrow: "Agent auth",
    title: "The layer nobody else verifies.",
    line: "MCP and OAuth agent tokens, checked against the spec they cite.",
    mock: AgentMock,
  },
];

function FeatureSection({ s }: { s: Section }) {
  const Mock = s.mock;
  return (
    <div className="grid grid-cols-1 items-center gap-8 lg:grid-cols-2 lg:gap-16">
      <Reveal className={s.flip ? "lg:order-2" : ""}>
        <span className="eyebrow">{s.eyebrow}</span>
        <h3 className="mt-2 text-h2 font-semibold tracking-tight">{s.title}</h3>
        <p className="mt-2.5 max-w-sm text-body text-muted">{s.line}</p>
      </Reveal>
      <Reveal delay={90} className={s.flip ? "lg:order-1" : ""}>
        <Mock />
      </Reveal>
    </div>
  );
}

export function FeatureSections() {
  return (
    <section className="mx-auto max-w-6xl px-5 py-20 sm:px-8 sm:py-24">
      <div className="space-y-20 sm:space-y-28">
        {sections.map((s) => (
          <FeatureSection key={s.title} s={s} />
        ))}
      </div>
    </section>
  );
}

/* ── Compact grid (reused by /features) ───────────────────────────────── */
const gridFeatures = [
  { icon: IconChanges, title: "Contract, versioned", body: "Snapshot what every service expects from a JWT as a hash you can diff." },
  { icon: IconFindings, title: "Drift, caught early", body: "Every change classified and attributed before it becomes an incident." },
  { icon: IconProbes, title: "Attacked, generatively", body: "alg=none, key confusion, jku injection and forged claims, fired at your endpoints." },
  { icon: IconBlast, title: "Blast radius", body: "See exactly which consumers break on a rotation, with a safe grace window." },
  { icon: IconCoverage, title: "Coverage, kept honest", body: "Measured against RFC 8725, OWASP and real CVEs. Every gap cited." },
  { icon: IconAgent, title: "Agent auth, verified", body: "MCP passthrough, missing delegation, over-scoped tokens — the layer nobody tests." },
];

export function FeatureGrid() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {gridFeatures.map((f, i) => (
        <Reveal key={f.title} delay={(i % 3) * 70}>
          <div className="lift h-full rounded-xl border border-border bg-surface p-6 shadow-xs">
            <span className="grid h-10 w-10 place-items-center rounded-lg bg-accent-soft text-accent">
              <f.icon />
            </span>
            <h3 className="mt-4 text-[1.05rem] font-semibold tracking-tight">{f.title}</h3>
            <p className="mt-2 text-body text-muted">{f.body}</p>
          </div>
        </Reveal>
      ))}
    </div>
  );
}

/* ── CTA ──────────────────────────────────────────────────────────────── */
export function CTA() {
  return (
    <section className="border-t border-border">
      <div className="mx-auto max-w-6xl px-5 py-24 text-center sm:px-8">
        <Reveal>
          <h2 className="display mx-auto max-w-xl text-h1 font-semibold tracking-tight">
            See it on your own contracts.
          </h2>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
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
          </div>
        </Reveal>
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
      <Proof />
      <ProductShowcase />
      <Thesis />
      <FeatureSections />
      <CTA />
    </MarketingShell>
  );
}
