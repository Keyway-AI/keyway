import { useEffect, useRef } from "react";

export interface PollingOptions {
  /** Milliseconds between successive runs. */
  intervalMs: number;
  /** When false, polling is suspended entirely. */
  enabled?: boolean;
  /**
   * Pause the timer while the tab is hidden and fire once immediately when it
   * becomes visible again — a backgrounded tab shouldn't hammer the backend, but
   * it should reflect reality the moment the user returns.
   */
  pauseWhenHidden?: boolean;
}

/**
 * Runs `fn` immediately, then every `intervalMs`, cleaning up the timer (and the
 * visibility listener) on unmount or when options change. The latest `fn` is
 * always used, so callers don't need to memoize it.
 */
export function usePolling(fn: () => void | Promise<void>, options: PollingOptions): void {
  const { intervalMs, enabled = true, pauseWhenHidden = true } = options;
  const saved = useRef(fn);
  saved.current = fn;

  useEffect(() => {
    if (!enabled) return;

    let timer: ReturnType<typeof setInterval> | undefined;
    const tick = () => void saved.current();

    const start = () => {
      if (timer !== undefined) return; // already running
      tick(); // run immediately so state is fresh, then on the interval
      timer = setInterval(tick, intervalMs);
    };
    const stop = () => {
      if (timer !== undefined) {
        clearInterval(timer);
        timer = undefined;
      }
    };
    const onVisibility = () => {
      if (document.hidden) stop();
      else start();
    };

    const hidden = pauseWhenHidden && typeof document !== "undefined" && document.hidden;
    if (hidden) {
      // Fire once immediately even when hidden, so a tab that loads in the
      // background doesn't get stuck on its initial state; defer the recurring
      // interval until the tab becomes visible.
      tick();
    } else {
      start();
    }

    if (pauseWhenHidden && typeof document !== "undefined") {
      document.addEventListener("visibilitychange", onVisibility);
    }
    return () => {
      stop();
      if (pauseWhenHidden && typeof document !== "undefined") {
        document.removeEventListener("visibilitychange", onVisibility);
      }
    };
  }, [intervalMs, enabled, pauseWhenHidden]);
}
