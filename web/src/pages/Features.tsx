import { Link } from "react-router-dom";
import { MarketingShell, PageHero } from "../components/MarketingChrome";
import { CTA, FeatureSections } from "./Marketing";

export default function Features() {
  return (
    <MarketingShell>
      <PageHero
        eyebrow="Features"
        title={
          <>
            Prove your auth is correct.<br className="hidden sm:block" /> Everything else assumes it.
          </>
        }
        line="From discovering token contracts to attacking your own endpoints — for services and AI agents."
        actions={
          <>
            <Link
              to="/signup"
              className="glow-accent inline-flex h-11 items-center rounded-md bg-accent px-5 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
            >
              Get started
            </Link>
            <Link
              to="/app"
              className="inline-flex h-11 items-center rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
            >
              Explore the demo →
            </Link>
          </>
        }
      />
      <FeatureSections />
      <CTA />
    </MarketingShell>
  );
}
