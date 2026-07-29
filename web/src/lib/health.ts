import { useState } from "react";
import { api } from "../api/client";
import { usePolling } from "./usePolling";

export type Health = "checking" | "ok" | "down" | "mock";

/** How long the sidebar/settings connection indicator waits between probes. */
export const HEALTH_POLL_MS = 20_000;

/**
 * classifyHealth is the single source of truth for turning a probe result into a
 * displayed state. In live mode an unreachable backend is a real error ("down");
 * otherwise the app is intentionally running on sample data ("mock").
 */
export function classifyHealth(reachable: boolean, live: boolean): Health {
  if (reachable) return "ok";
  return live ? "down" : "mock";
}

function isLive(): boolean {
  return typeof localStorage !== "undefined" && localStorage.getItem("keyway.live") === "1";
}

/**
 * useHealthProbe polls `/v1/health` on an interval so the connection indicator
 * reflects reality as the backend comes and goes — not just its state at mount.
 * The probe pauses while the tab is hidden.
 */
export function useHealthProbe(intervalMs: number = HEALTH_POLL_MS): Health {
  const [health, setHealth] = useState<Health>("checking");
  usePolling(
    async () => {
      try {
        await api.health();
        setHealth(classifyHealth(true, isLive()));
      } catch {
        setHealth(classifyHealth(false, isLive()));
      }
    },
    { intervalMs },
  );
  return health;
}
