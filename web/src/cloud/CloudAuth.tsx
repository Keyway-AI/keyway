import { createContext, useCallback, useContext, useEffect, useState } from "react";
import type { ReactNode } from "react";
import { cloud, isUnauthorized } from "./api";
import type { CloudConfig, CloudUser } from "./api";

/**
 * CloudAuth holds the hosted-layer session: server capabilities (`/v1/config`)
 * and the signed-in user (`/v1/me`). `user === null` after a settled load means
 * "not signed in" — the guard routes those to the sign-in page. Errors other than
 * 401 (e.g. the API is down) are surfaced so we don't silently pretend to be
 * logged out.
 */
interface CloudAuthState {
  loading: boolean;
  config: CloudConfig | null;
  user: CloudUser | null;
  error: string | null;
  refresh: () => Promise<void>;
  signOut: () => Promise<void>;
}

const Ctx = createContext<CloudAuthState | null>(null);

export function CloudAuthProvider({ children }: { children: ReactNode }) {
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState<CloudConfig | null>(null);
  const [user, setUser] = useState<CloudUser | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const cfg = await cloud.config();
      setConfig(cfg);
    } catch (e) {
      setConfig(null);
      setError(e instanceof Error ? e.message : "Cloud API unavailable");
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      setUser(await cloud.me());
    } catch (e) {
      if (isUnauthorized(e)) {
        setUser(null); // simply not signed in
      } else {
        setError(e instanceof Error ? e.message : "Failed to load session");
      }
    } finally {
      setLoading(false);
    }
  }, []);

  const signOut = useCallback(async () => {
    try {
      await cloud.logout();
    } catch {
      /* best-effort */
    }
    setUser(null);
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return (
    <Ctx.Provider value={{ loading, config, user, error, refresh, signOut }}>
      {children}
    </Ctx.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useCloudAuth(): CloudAuthState {
  const v = useContext(Ctx);
  if (!v) throw new Error("useCloudAuth must be used within CloudAuthProvider");
  return v;
}
