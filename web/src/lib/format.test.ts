import { describe, expect, it } from "vitest";
import { humanizeDuration, relativeTime, ttlLabel } from "./format";

describe("humanizeDuration", () => {
  it("returns 0s for non-positive input", () => {
    expect(humanizeDuration(0)).toBe("0s");
    expect(humanizeDuration(-10)).toBe("0s");
  });

  it("formats days and hours, dropping minutes once days are present", () => {
    // 9d 5h 0m 0s = 795600
    expect(humanizeDuration(795600)).toBe("9d 5h");
    // 9d 5h 3m — minutes suppressed when days present
    expect(humanizeDuration(795600 + 180)).toBe("9d 5h");
  });

  it("shows hours and minutes below a day", () => {
    expect(humanizeDuration(2 * 3600 + 30 * 60)).toBe("2h 30m");
  });

  it("shows just minutes under an hour", () => {
    expect(humanizeDuration(45 * 60)).toBe("45m");
  });

  it("falls back to seconds when under a minute", () => {
    expect(humanizeDuration(30)).toBe("30s");
  });
});

describe("ttlLabel", () => {
  it("renders an em dash for null/undefined", () => {
    expect(ttlLabel(null)).toBe("—");
    expect(ttlLabel(undefined)).toBe("—");
  });

  it("humanizes a real duration", () => {
    expect(ttlLabel(3600)).toBe("1h");
  });
});

describe("relativeTime", () => {
  it("says 'just now' within the minute", () => {
    expect(relativeTime(new Date(Date.now() - 5_000).toISOString())).toBe("just now");
  });

  it("counts minutes, hours, then days", () => {
    expect(relativeTime(new Date(Date.now() - 5 * 60_000).toISOString())).toBe("5m ago");
    expect(relativeTime(new Date(Date.now() - 3 * 3600_000).toISOString())).toBe("3h ago");
    expect(relativeTime(new Date(Date.now() - 2 * 86_400_000).toISOString())).toBe("2d ago");
  });
});
