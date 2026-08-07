import { Link } from "react-router-dom";
import type { ReactNode } from "react";
import { ThemeToggle } from "./ThemeToggle";
import { LogoBadge } from "./Logo";

export function BrandMark({ size = "md" }: { size?: "sm" | "md" }) {
  const badge = size === "sm" ? 24 : 28;
  const text = size === "sm" ? "text-[0.95rem]" : "text-[1.05rem]";
  return (
    <Link to="/" className="flex items-center gap-2.5">
      <LogoBadge size={badge} />
      <span className={`font-semibold tracking-[-0.02em] ${text}`}>Keyway</span>
    </Link>
  );
}

const navLinks = [
  { to: "/features", label: "Features" },
  { to: "/pricing", label: "Pricing" },
  { to: "/cloud", label: "Cloud" },
  { to: "/app/coverage", label: "Coverage" },
  { to: "/contact", label: "Contact" },
];

export function MarketingNav() {
  return (
    <header className="sticky top-0 z-30 px-3 pt-3 sm:px-5 sm:pt-4">
      <div className="glass mx-auto flex max-w-5xl items-center justify-between gap-4 rounded-pill py-2 pl-4 pr-2 shadow-md">
        <div className="flex items-center gap-7">
          <BrandMark />
          <nav className="hidden items-center gap-6 md:flex">
            {navLinks.map((l) => (
              <Link key={l.to} to={l.to} className="text-caption font-medium text-muted transition hover:text-text">
                {l.label}
              </Link>
            ))}
            <a
              href="https://github.com/Keyway-AI/keyway"
              className="text-caption font-medium text-muted transition hover:text-text"
            >
              GitHub
            </a>
          </nav>
        </div>
        <div className="flex items-center gap-2">
          <div className="hidden sm:block">
            <ThemeToggle />
          </div>
          <Link
            to="/login"
            className="hidden h-9 items-center rounded-pill px-3 text-caption font-medium text-muted transition hover:text-text sm:inline-flex"
          >
            Sign in
          </Link>
          <Link
            to="/signup"
            className="glow-accent inline-flex h-9 items-center rounded-pill bg-accent px-4 text-caption font-semibold text-accent-fg transition hover:bg-accent-strong active:scale-[0.98]"
          >
            Get started
          </Link>
        </div>
      </div>
    </header>
  );
}

const footerCols: { title: string; links: { label: string; to?: string; href?: string }[] }[] = [
  {
    title: "Product",
    links: [
      { label: "Features", to: "/features" },
      { label: "Pricing", to: "/pricing" },
      { label: "Coverage", to: "/app/coverage" },
      { label: "Open the app", to: "/app" },
    ],
  },
  {
    title: "Resources",
    links: [
      { label: "GitHub", href: "https://github.com/Keyway-AI/keyway" },
      { label: "Documentation", href: "https://github.com/Keyway-AI/keyway#readme" },
      { label: "Security", href: "https://github.com/Keyway-AI/keyway/blob/main/SECURITY.md" },
      { label: "Changelog", href: "https://github.com/Keyway-AI/keyway/releases" },
    ],
  },
  {
    title: "Company",
    links: [
      { label: "Contact", to: "/contact" },
      { label: "Sign in", to: "/login" },
      { label: "Get started", to: "/signup" },
      { label: "Report an issue", href: "https://github.com/Keyway-AI/keyway/issues" },
    ],
  },
];

export function MarketingFooter() {
  return (
    <footer className="border-t border-border bg-surface-2/40">
      <div className="mx-auto max-w-6xl px-5 py-12 sm:px-8">
        <div className="grid grid-cols-2 gap-8 sm:grid-cols-4">
          <div className="col-span-2 sm:col-span-1">
            <BrandMark />
            <p className="mt-3 max-w-[26ch] text-caption text-muted">
              Contract verification for JWT &amp; AI-agent auth. Open source.
            </p>
          </div>
          {footerCols.map((col) => (
            <div key={col.title}>
              <div className="eyebrow mb-3">{col.title}</div>
              <ul className="space-y-2">
                {col.links.map((l) => (
                  <li key={l.label}>
                    {l.to ? (
                      <Link to={l.to} className="text-caption text-muted transition hover:text-text">
                        {l.label}
                      </Link>
                    ) : (
                      <a href={l.href} className="text-caption text-muted transition hover:text-text">
                        {l.label}
                      </a>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <div className="mt-10 flex flex-col items-center justify-between gap-3 border-t border-border pt-6 text-caption text-faint sm:flex-row">
          <span>© {"2026"} Keyway — open source, MIT licensed.</span>
          <span>Built for a world where machines authenticate machines.</span>
        </div>
      </div>
    </footer>
  );
}

export function MarketingShell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen">
      <MarketingNav />
      {children}
      <MarketingFooter />
    </div>
  );
}
