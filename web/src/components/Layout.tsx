import { NavLink, Outlet } from "react-router-dom";
import { useEffect, useState } from "react";
import type { ComponentType, ReactNode, SVGProps } from "react";
import { SettingsModal, useHealthProbe } from "./Settings";
import { ThemeToggle } from "./ThemeToggle";
import { LogoBadge } from "./Logo";
import {
  IconAgent,
  IconBlast,
  IconCanary,
  IconChanges,
  IconClose,
  IconConsumers,
  IconCoverage,
  IconDashboard,
  IconFindings,
  IconMenu,
  IconProbes,
  IconSettings,
} from "./icons";

type NavEntry = { to: string; label: string; end?: boolean; icon: ComponentType<SVGProps<SVGSVGElement>> };

const nav: { section: string; items: NavEntry[] }[] = [
  {
    section: "Overview",
    items: [
      { to: "/app", label: "Dashboard", end: true, icon: IconDashboard },
      { to: "/app/findings", label: "Findings", icon: IconFindings },
    ],
  },
  {
    section: "Contract",
    items: [
      { to: "/app/consumers", label: "Consumers", icon: IconConsumers },
      { to: "/app/changes", label: "Changes", icon: IconChanges },
    ],
  },
  {
    section: "Verify",
    items: [
      { to: "/app/probes", label: "Probes", icon: IconProbes },
      { to: "/app/blast-radius", label: "Blast radius", icon: IconBlast },
      { to: "/app/canary", label: "Canary", icon: IconCanary },
      { to: "/app/coverage", label: "Coverage", icon: IconCoverage },
      { to: "/app/agent", label: "Agent auth", icon: IconAgent },
    ],
  },
];

const healthDot: Record<string, { cls: string; label: string }> = {
  checking: { cls: "bg-faint", label: "Checking connection…" },
  ok: { cls: "bg-low", label: "Connected to backend" },
  down: { cls: "bg-critical", label: "Backend unreachable" },
  mock: { cls: "bg-high", label: "Sample data" },
};

function NavItem({ to, label, end, icon: Icon, onNavigate }: NavEntry & { onNavigate?: () => void }) {
  return (
    <NavLink
      to={to}
      end={end}
      onClick={onNavigate}
      className={({ isActive }) =>
        `flex items-center gap-3 rounded-md px-3 py-2 text-body font-medium tracking-tight transition ${
          isActive
            ? "bg-accent-soft text-accent"
            : "text-muted hover:bg-surface-2 hover:text-text"
        }`
      }
    >
      <Icon className="shrink-0" />
      {label}
    </NavLink>
  );
}

function Brand() {
  return (
    <NavLink to="/" className="flex items-center gap-2.5">
      <LogoBadge size={28} />
      <span className="text-[1.05rem] font-semibold tracking-[-0.02em] text-text">Keyway</span>
    </NavLink>
  );
}

function SidebarBody({ onNavigate, onSettings }: { onNavigate?: () => void; onSettings: () => void }) {
  const health = useHealthProbe();
  const dot = healthDot[health] ?? healthDot.checking;
  return (
    <>
      <nav className="flex flex-1 flex-col gap-5 overflow-y-auto px-3 py-5">
        {nav.map((group) => (
          <div key={group.section} className="flex flex-col gap-0.5">
            <div className="eyebrow px-3 pb-1.5">{group.section}</div>
            {group.items.map((n) => (
              <NavItem key={n.to} {...n} onNavigate={onNavigate} />
            ))}
          </div>
        ))}
      </nav>
      <div className="flex flex-col gap-3 border-t border-border px-3 py-4">
        <ThemeToggle />
        <button
          onClick={onSettings}
          className="flex items-center gap-3 rounded-md px-3 py-2 text-body font-medium text-muted transition hover:bg-surface-2 hover:text-text"
        >
          <IconSettings className="shrink-0" />
          Settings
        </button>
        <div className="flex items-center gap-2 px-3 text-caption text-muted" title={dot.label}>
          <span className={`h-1.5 w-1.5 rounded-full ${dot.cls}`} />
          <span>{dot.label}</span>
        </div>
      </div>
    </>
  );
}

export default function Layout() {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  // Lock scroll behind the mobile drawer.
  useEffect(() => {
    document.body.style.overflow = drawerOpen ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [drawerOpen]);

  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <aside className="sticky top-0 hidden h-screen w-64 shrink-0 flex-col border-r border-border bg-surface lg:flex">
        <div className="px-5 py-5">
          <Brand />
        </div>
        <SidebarBody onSettings={() => setSettingsOpen(true)} />
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Mobile top bar */}
        <header className="sticky top-0 z-30 flex items-center justify-between border-b border-border bg-surface/85 px-4 py-3 backdrop-blur-md lg:hidden">
          <Brand />
          <button
            onClick={() => setDrawerOpen(true)}
            className="grid h-9 w-9 place-items-center rounded-md border border-border text-muted hover:bg-surface-2 hover:text-text"
            aria-label="Open menu"
          >
            <IconMenu />
          </button>
        </header>

        <main className="min-w-0 flex-1">
          <Outlet />
        </main>
      </div>

      {/* Mobile drawer */}
      {drawerOpen && (
        <div className="fixed inset-0 z-50 lg:hidden" role="dialog" aria-modal="true">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => setDrawerOpen(false)} />
          <aside className="animate-sheet-in absolute inset-y-0 left-0 flex w-72 max-w-[82%] flex-col border-r border-border bg-surface">
            <div className="flex items-center justify-between px-5 py-4">
              <Brand />
              <button
                onClick={() => setDrawerOpen(false)}
                className="grid h-8 w-8 place-items-center rounded-md text-muted hover:bg-surface-2 hover:text-text"
                aria-label="Close menu"
              >
                <IconClose />
              </button>
            </div>
            <SidebarBody
              onNavigate={() => setDrawerOpen(false)}
              onSettings={() => {
                setDrawerOpen(false);
                setSettingsOpen(true);
              }}
            />
          </aside>
        </div>
      )}

      {settingsOpen && <SettingsModal onClose={() => setSettingsOpen(false)} />}
    </div>
  );
}

/* ── Page shell ───────────────────────────────────────────────────────── */

export function Page({
  title,
  subtitle,
  actions,
  children,
}: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="mx-auto max-w-6xl px-5 py-6 sm:px-8 sm:py-9">
      <header className="mb-6 flex flex-col gap-3 sm:mb-8 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
        <div className="min-w-0">
          <h1 className="text-h1 font-semibold tracking-tight">{title}</h1>
          {subtitle && <p className="mt-1.5 max-w-2xl text-body text-muted">{subtitle}</p>}
        </div>
        {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
      </header>
      {children}
    </div>
  );
}
