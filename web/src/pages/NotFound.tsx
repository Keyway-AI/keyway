import { Link } from "react-router-dom";
import { MarketingShell } from "../components/MarketingChrome";
import { LogoGlyph } from "../components/Logo";

export default function NotFound() {
  return (
    <MarketingShell>
      <section className="mx-auto flex max-w-xl flex-col items-center px-5 py-28 text-center sm:py-36">
        <LogoGlyph width={44} height={44} className="text-accent" />
        <p className="mt-6 text-micro font-semibold uppercase tracking-wider text-faint">404</p>
        <h1 className="display mt-2 text-[2rem] font-semibold tracking-tight sm:text-[2.4rem]">
          This key doesn&apos;t turn.
        </h1>
        <p className="mt-3 max-w-sm text-body text-muted">
          The page you&apos;re looking for isn&apos;t here. It may have moved, or the link is off.
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <Link
            to="/"
            className="glow-accent inline-flex h-11 items-center rounded-md bg-accent px-5 text-body font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
          >
            Back home
          </Link>
          <a
            href="https://github.com/Keyway-AI/keyway"
            className="inline-flex h-11 items-center rounded-md border border-border bg-surface px-5 text-body font-medium text-text shadow-xs transition hover:bg-surface-2 active:scale-[0.98]"
          >
            View on GitHub
          </a>
        </div>
      </section>
    </MarketingShell>
  );
}
