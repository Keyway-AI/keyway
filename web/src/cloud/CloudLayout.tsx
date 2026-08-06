import { useState } from "react";
import { Link, Outlet, useNavigate } from "react-router-dom";
import { useCloudAuth } from "./CloudAuth";
import CloudSignIn from "./CloudSignIn";
import { LogoBadge } from "../components/Logo";
import { ThemeToggle } from "../components/ThemeToggle";

/**
 * Guarded shell for the hosted app. While the session loads we show a quiet
 * splash; a settled `user === null` renders the sign-in front door; otherwise the
 * chrome (brand + user menu) wraps the routed project pages.
 */
export default function CloudLayout() {
  const { loading, user } = useCloudAuth();

  if (loading) {
    return (
      <div className="aurora grid min-h-screen place-items-center">
        <span className="h-6 w-6 animate-spin rounded-full border-2 border-accent border-t-transparent" />
      </div>
    );
  }
  if (!user) return <CloudSignIn />;

  return (
    <div className="aurora min-h-screen">
      <CloudNav />
      <main className="mx-auto max-w-6xl px-5 py-8 sm:px-8">
        <Outlet />
      </main>
    </div>
  );
}

function CloudNav() {
  const { user, signOut } = useCloudAuth();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-30 border-b border-border/70 bg-bg/70 backdrop-blur-md">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-5 py-3 sm:px-8">
        <Link to="/cloud" className="flex items-center gap-2.5">
          <LogoBadge size={26} />
          <span className="text-[1.02rem] font-semibold tracking-[-0.02em]">Keyway</span>
          <span className="rounded-pill border border-border bg-surface/70 px-2 py-0.5 text-micro font-semibold uppercase tracking-wider text-muted">
            Cloud
          </span>
        </Link>

        <div className="flex items-center gap-2.5">
          <ThemeToggle />
          <div className="relative">
            <button
              onClick={() => setOpen((v) => !v)}
              className="flex items-center gap-2 rounded-pill border border-border bg-surface/70 py-1 pl-1 pr-2.5 text-caption font-medium transition hover:bg-surface-2"
            >
              {user?.avatar_url ? (
                <img src={user.avatar_url} alt="" className="h-6 w-6 rounded-full" />
              ) : (
                <span className="grid h-6 w-6 place-items-center rounded-full bg-accent text-[0.7rem] font-semibold text-accent-fg">
                  {(user?.login ?? "?").slice(0, 1).toUpperCase()}
                </span>
              )}
              <span className="max-w-[12ch] truncate">{user?.login}</span>
            </button>
            {open && (
              <>
                <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
                <div className="absolute right-0 z-20 mt-2 w-52 overflow-hidden rounded-xl border border-border bg-elevated shadow-lg">
                  <div className="border-b border-border px-3.5 py-2.5">
                    <div className="truncate text-caption font-medium text-text">{user?.name || user?.login}</div>
                    {user?.email && <div className="truncate text-micro text-muted">{user.email}</div>}
                  </div>
                  <button
                    onClick={() => {
                      setOpen(false);
                      void signOut().then(() => navigate("/cloud"));
                    }}
                    className="block w-full px-3.5 py-2.5 text-left text-caption text-muted transition hover:bg-surface-2 hover:text-text"
                  >
                    Sign out
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
