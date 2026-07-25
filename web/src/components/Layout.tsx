import { NavLink, Outlet } from "react-router-dom";
import type { ReactNode } from "react";

const nav = [
  { to: "/", label: "Dashboard", end: true, icon: "◉" },
  { to: "/findings", label: "Findings", icon: "⚠" },
  { to: "/consumers", label: "Consumers", icon: "▤" },
  { to: "/changes", label: "Changes", icon: "◔" },
  { to: "/blast-radius", label: "Blast radius", icon: "✷" },
  { to: "/canary", label: "Canary", icon: "◇" },
];

function NavItem({ to, label, end, icon }: { to: string; label: string; end?: boolean; icon: ReactNode }) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        `flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
          isActive
            ? "bg-surface-2 text-text"
            : "text-muted hover:bg-surface-2 hover:text-text"
        }`
      }
    >
      <span className="w-4 text-center text-muted">{icon}</span>
      {label}
    </NavLink>
  );
}

export default function Layout() {
  return (
    <div className="flex min-h-screen">
      <aside className="flex w-60 shrink-0 flex-col border-r border-border bg-surface px-3 py-5">
        <div className="flex items-center gap-2 px-3 pb-6">
          <span className="text-xl">🔑</span>
          <span className="text-lg font-semibold tracking-tight">Keyway</span>
        </div>
        <nav className="flex flex-col gap-1">
          {nav.map((n) => (
            <NavItem key={n.to} {...n} />
          ))}
        </nav>
        <div className="mt-auto px-3 pt-6 text-xs text-muted">
          <p>Contract verification for JWT consumers.</p>
        </div>
      </aside>
      <main className="flex-1 overflow-x-hidden">
        <Outlet />
      </main>
    </div>
  );
}

export function Page({ title, subtitle, actions, children }: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="mx-auto max-w-6xl px-8 py-8">
      <header className="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
          {subtitle && <p className="mt-1 text-sm text-muted">{subtitle}</p>}
        </div>
        {actions}
      </header>
      {children}
    </div>
  );
}
