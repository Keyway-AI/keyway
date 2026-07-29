// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { usePolling } from "./usePolling";

describe("usePolling", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("runs immediately, then once per interval", () => {
    const fn = vi.fn();
    renderHook(() => usePolling(fn, { intervalMs: 1000 }));
    expect(fn).toHaveBeenCalledTimes(1); // immediate

    vi.advanceTimersByTime(1000);
    expect(fn).toHaveBeenCalledTimes(2);

    vi.advanceTimersByTime(2000);
    expect(fn).toHaveBeenCalledTimes(4);
  });

  it("stops the timer on unmount", () => {
    const fn = vi.fn();
    const { unmount } = renderHook(() => usePolling(fn, { intervalMs: 1000 }));
    expect(fn).toHaveBeenCalledTimes(1);
    unmount();
    vi.advanceTimersByTime(5000);
    expect(fn).toHaveBeenCalledTimes(1); // no further ticks after unmount
  });

  it("does not run at all when disabled", () => {
    const fn = vi.fn();
    renderHook(() => usePolling(fn, { intervalMs: 1000, enabled: false }));
    vi.advanceTimersByTime(5000);
    expect(fn).not.toHaveBeenCalled();
  });

  it("always calls the latest fn without needing a re-subscribe", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = renderHook(({ f }) => usePolling(f, { intervalMs: 1000 }), {
      initialProps: { f: first },
    });
    expect(first).toHaveBeenCalledTimes(1);

    rerender({ f: second });
    vi.advanceTimersByTime(1000);
    expect(second).toHaveBeenCalledTimes(1);
    expect(first).toHaveBeenCalledTimes(1); // old closure not called again
  });

  it("pauses while the tab is hidden and resumes on visibility", () => {
    const fn = vi.fn();
    renderHook(() => usePolling(fn, { intervalMs: 1000, pauseWhenHidden: true }));
    expect(fn).toHaveBeenCalledTimes(1);

    // Hide the tab: the timer should stop.
    Object.defineProperty(document, "hidden", { configurable: true, get: () => true });
    document.dispatchEvent(new Event("visibilitychange"));
    vi.advanceTimersByTime(5000);
    expect(fn).toHaveBeenCalledTimes(1);

    // Show it again: fires once immediately, then resumes ticking.
    Object.defineProperty(document, "hidden", { configurable: true, get: () => false });
    document.dispatchEvent(new Event("visibilitychange"));
    expect(fn).toHaveBeenCalledTimes(2);
    vi.advanceTimersByTime(1000);
    expect(fn).toHaveBeenCalledTimes(3);
  });
});
