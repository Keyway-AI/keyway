import { MarketingShell } from "../components/MarketingChrome";

/**
 * The research & methods hub. Surfaces the whitepaper, the research-note series,
 * and the reproducible evidence. Every number shown here is measured and
 * reproduced by a named command (see the "Reproduce" block); the docs it links to
 * live in the repo and render on GitHub.
 */

const REPO = "https://github.com/Keyway-AI/keyway/blob/main";

const notes = [
  {
    n: "01",
    title: "Deriving auth contracts from configuration",
    blurb: "How discovery recovers the consumer inventory — with per-field provenance and a library-defaults DB — from Istio, Envoy, K8s, and OIDC config.",
    href: `${REPO}/docs/research/01-contract-discovery.md`,
  },
  {
    n: "02",
    title: "Zero-false-alarm drift classification",
    blurb: "A fixed rule table classifies semantic drift (widened / narrowed). 100% recall / 0% false alarms on 800 pairs — and how mutation testing keeps that honest.",
    href: `${REPO}/docs/research/02-drift-classification.md`,
  },
  {
    n: "03",
    title: "Adversarially verifying JWT verifiers",
    blurb: "Minting real attack tokens (alg=none, key confusion, jku/kid injection) and firing them at a staging verifier — deny-by-default, with a hard production guard.",
    href: `${REPO}/docs/research/03-adversarial-verification.md`,
  },
  {
    n: "04",
    title: "Verifying AI-agent authorization",
    blurb: "Applying the contract model to MCP/OAuth agent tokens: audience binding, delegation, scope, expiry — cited to spec, with coverage stated honestly.",
    href: `${REPO}/docs/research/04-agent-auth.md`,
  },
];

const supporting = [
  { label: "Benchmark — how accurate is it?", href: `${REPO}/BENCHMARK.md` },
  { label: "Benchmark methodology", href: `${REPO}/docs/benchmark.md` },
  { label: "Integrity — is 100% overfit?", href: `${REPO}/docs/benchmark-integrity.md` },
  { label: "Threat coverage (the denominator)", href: `${REPO}/docs/threat-coverage.md` },
  { label: "Real-world CVE / incident validation", href: `${REPO}/docs/realworld-validation.md` },
  { label: "Architecture", href: `${REPO}/ARCHITECTURE.md` },
];

const stats = [
  { n: "100% / 0%", l: "recall / false-alarm rate — gated corpus (J = 1.0)" },
  { n: "0.75", l: "Youden J on held-out adversarial cases" },
  { n: "24", l: "mutants, 100% killed (2 provably equivalent)" },
  { n: "27 / 50", l: "cited threats covered — every gap named" },
];

export default function Research() {
  return (
    <MarketingShell>
      {/* hero */}
      <section className="aurora border-b border-border px-5 py-16 sm:px-8 sm:py-24">
        <div className="mx-auto max-w-3xl text-center">
          <span className="eyebrow">Research &amp; methods</span>
          <h1 className="display mt-2 text-[2.4rem] sm:text-[3rem]">
            We publish the stress tests, not just the wins.
          </h1>
          <p className="mx-auto mt-4 max-w-xl text-body-lg text-muted">
            Keyway is measured against a cited taxonomy of documented threats, and every accuracy
            number ships with its counter-evidence. Nothing here is asserted that the repo can’t
            regenerate.
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <a
              href={`${REPO}/docs/whitepaper.md`}
              className="glow-accent inline-flex h-11 items-center rounded-md bg-accent px-5 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
            >
              Read the whitepaper
            </a>
            <a
              href={`${REPO}/BENCHMARK.md`}
              className="inline-flex h-11 items-center rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
            >
              How accurate is it?
            </a>
          </div>
        </div>
      </section>

      {/* reproduced results */}
      <section className="border-b border-border bg-surface-2/30">
        <div className="mx-auto max-w-6xl px-5 py-14 sm:px-8">
          <div className="grid grid-cols-2 gap-6 sm:grid-cols-4">
            {stats.map((s) => (
              <div key={s.l} className="text-center">
                <div className="text-h2 font-semibold tracking-tight tabular-nums text-accent">{s.n}</div>
                <div className="mt-1 text-caption text-muted">{s.l}</div>
              </div>
            ))}
          </div>
          <div className="mx-auto mt-10 max-w-2xl rounded-xl border border-border bg-surface p-5 shadow-xs">
            <div className="eyebrow mb-2">Reproduce every number</div>
            <pre className="overflow-x-auto text-[0.78rem] leading-relaxed text-text">
              <code className="font-mono">{`make bench       # accuracy scorecard → bench/out/scorecard.json
make mutation    # mutation score (installs gremlins if absent)
keyway threats coverage   # the 27/50 coverage table`}</code>
            </pre>
          </div>
        </div>
      </section>

      {/* whitepaper feature */}
      <section className="border-b border-border">
        <div className="mx-auto max-w-6xl px-5 py-16 sm:px-8 sm:py-20">
          <a
            href={`${REPO}/docs/whitepaper.md`}
            className="group block rounded-2xl border border-border bg-surface p-8 shadow-xs transition hover:border-border-strong hover:shadow-md sm:p-10"
          >
            <div className="eyebrow">Whitepaper</div>
            <h2 className="mt-2 text-h1 font-semibold tracking-tight display">
              Deriving, versioning, and adversarially verifying token-auth contracts
            </h2>
            <p className="mt-3 max-w-2xl text-body-lg text-muted">
              The anchor document: the problem, the five-stage approach, the benchmark design, the
              results with their limitations, and how to reproduce all of it.
            </p>
            <span className="mt-5 inline-flex items-center gap-1.5 text-body font-medium text-accent transition group-hover:translate-x-0.5">
              Read the whitepaper →
            </span>
          </a>
        </div>
      </section>

      {/* research notes */}
      <section className="border-b border-border bg-surface-2/30">
        <div className="mx-auto max-w-6xl px-5 py-16 sm:px-8 sm:py-20">
          <div className="mx-auto max-w-2xl text-center">
            <div className="eyebrow">Research notes</div>
            <h2 className="mt-2 text-h1 font-semibold tracking-tight display">One approach per note.</h2>
            <p className="mt-3 text-body-lg text-muted">
              arXiv-style: abstract, method, results, threats to validity, and a reproduce block.
            </p>
          </div>
          <div className="mt-12 grid gap-4 sm:grid-cols-2">
            {notes.map((note) => (
              <a
                key={note.n}
                href={note.href}
                className="group flex gap-4 rounded-xl border border-border bg-surface p-6 shadow-xs transition hover:border-border-strong hover:shadow-md"
              >
                <span className="text-h2 font-semibold tabular-nums text-accent/40 transition group-hover:text-accent">
                  {note.n}
                </span>
                <div>
                  <h3 className="text-[1.05rem] font-semibold tracking-tight">{note.title}</h3>
                  <p className="mt-1.5 text-caption text-muted">{note.blurb}</p>
                </div>
              </a>
            ))}
          </div>
        </div>
      </section>

      {/* supporting docs */}
      <section>
        <div className="mx-auto max-w-6xl px-5 py-16 sm:px-8 sm:py-20">
          <div className="mx-auto max-w-2xl text-center">
            <div className="eyebrow">Reference</div>
            <h2 className="mt-2 text-h1 font-semibold tracking-tight display">The receipts.</h2>
            <p className="mt-3 text-body-lg text-muted">
              The methodology, the integrity stress-tests, the cited threat taxonomy, and the
              real-world CVE validation — all in the repo.
            </p>
          </div>
          <div className="mx-auto mt-10 max-w-2xl divide-y divide-border rounded-xl border border-border bg-surface shadow-xs">
            {supporting.map((d) => (
              <a
                key={d.label}
                href={d.href}
                className="flex items-center justify-between gap-3 px-5 py-3.5 text-body text-text transition hover:bg-surface-2"
              >
                <span>{d.label}</span>
                <span className="text-faint">↗</span>
              </a>
            ))}
          </div>
        </div>
      </section>
    </MarketingShell>
  );
}
