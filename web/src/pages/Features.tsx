import { Link } from "react-router-dom";
import { MarketingShell, PageHero } from "../components/MarketingChrome";
import { buttonClass } from "../components/ui";
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
            <Link to="/signup" className={buttonClass("primary", "lg")}>
              Get started
            </Link>
            <Link to="/app" className={buttonClass("secondary", "lg")}>
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
