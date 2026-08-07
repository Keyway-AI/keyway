import { Link } from "react-router-dom";
import { MarketingShell } from "../components/MarketingChrome";
import { FeatureGrid } from "./Marketing";

export default function Features() {
  return (
    <MarketingShell>
      <section className="aurora border-b border-border px-5 py-16 sm:px-8 sm:py-24">
        <div className="mx-auto max-w-6xl">
          <div className="max-w-2xl">
            <span className="eyebrow">Features</span>
            <h1 className="display mt-2 text-[2.4rem] sm:text-[3rem]">
              Everything you need to prove your auth is correct.
            </h1>
            <p className="mt-4 text-body-lg text-muted">
              From discovering token contracts to adversarially attacking your endpoints — for classic services and AI agents.
            </p>
          </div>
          <div className="mt-12">
            <FeatureGrid />
          </div>
          <div className="mt-14 flex flex-wrap items-center gap-3">
            <Link
              to="/signup"
              className="glow-accent inline-flex h-11 items-center rounded-md bg-accent px-5 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
            >
              Get started — free
            </Link>
            <Link
              to="/app"
              className="inline-flex h-11 items-center rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
            >
              Explore the demo
            </Link>
          </div>
        </div>
      </section>
    </MarketingShell>
  );
}
