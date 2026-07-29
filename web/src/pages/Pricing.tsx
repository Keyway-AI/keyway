import { Link } from "react-router-dom";
import { MarketingShell } from "../components/MarketingChrome";

const tiers = [
  {
    name: "Open source",
    price: "Free",
    unit: "self-hosted, forever",
    cta: { label: "Get started", to: "/signup", primary: true },
    highlight: false,
    features: [
      "Contract discovery & versioning",
      "Drift detection & attribution",
      "The full generative attack harness",
      "Blast-radius & canary flows",
      "JWT + agent-auth coverage",
      "Single binary or container",
    ],
  },
  {
    name: "Cloud",
    price: "Coming soon",
    unit: "hosted, with history & alerts",
    cta: { label: "Join the waitlist", href: "https://github.com/nometria/keyway", primary: false },
    highlight: true,
    features: [
      "Everything in open source",
      "Managed, always-on scanning",
      "Historical coverage trends",
      "Slack / PagerDuty alerting",
      "SSO & team roles",
      "Priority support",
    ],
  },
];

function Check() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" className="mt-0.5 shrink-0 text-low">
      <path d="m5 12 5 5L20 7" />
    </svg>
  );
}

export default function Pricing() {
  return (
    <MarketingShell>
      <section className="mx-auto max-w-5xl px-5 py-16 text-center sm:px-8 sm:py-24">
        <span className="eyebrow">Pricing</span>
        <h1 className="mt-2 text-h1 font-semibold tracking-tight sm:text-[2.75rem]">
          Free and open source. Cloud when you want it.
        </h1>
        <p className="mx-auto mt-3 max-w-lg text-body-lg text-muted">
          The whole tool is open source and self-hosted at no cost. A managed cloud with history and alerting is on the way.
        </p>

        <div className="mx-auto mt-12 grid max-w-3xl grid-cols-1 gap-4 text-left md:grid-cols-2">
          {tiers.map((t) => (
            <div
              key={t.name}
              className={`flex flex-col rounded-2xl border bg-surface p-7 shadow-xs ${
                t.highlight ? "border-accent/40 ring-1 ring-accent/20" : "border-border"
              }`}
            >
              <div className="flex items-center justify-between">
                <h3 className="text-[1.05rem] font-semibold tracking-tight">{t.name}</h3>
                {t.highlight && (
                  <span className="rounded-pill bg-accent-soft px-2.5 py-0.5 text-micro font-semibold uppercase tracking-wider text-accent">
                    Preview
                  </span>
                )}
              </div>
              <div className="mt-4 flex items-baseline gap-2">
                <span className="text-h1 font-semibold tracking-tight">{t.price}</span>
              </div>
              <div className="mt-1 text-caption text-muted">{t.unit}</div>

              {t.cta.primary ? (
                <Link
                  to={t.cta.to!}
                  className="mt-6 inline-flex h-10 items-center justify-center rounded-md bg-accent px-4 text-body font-medium text-accent-fg shadow-xs transition hover:bg-accent-strong active:scale-[0.98]"
                >
                  {t.cta.label}
                </Link>
              ) : (
                <a
                  href={t.cta.href}
                  className="mt-6 inline-flex h-10 items-center justify-center rounded-md border border-border bg-surface px-4 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
                >
                  {t.cta.label}
                </a>
              )}

              <ul className="mt-6 space-y-2.5 border-t border-border pt-6">
                {t.features.map((f) => (
                  <li key={f} className="flex items-start gap-2.5 text-body text-text">
                    <Check />
                    {f}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </section>
    </MarketingShell>
  );
}
