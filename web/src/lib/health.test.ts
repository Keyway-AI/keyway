import { describe, expect, it } from "vitest";
import { classifyHealth } from "./health";

describe("classifyHealth", () => {
  it("reports ok whenever the backend is reachable, regardless of mode", () => {
    expect(classifyHealth(true, true)).toBe("ok");
    expect(classifyHealth(true, false)).toBe("ok");
  });

  it("reports down when unreachable in live mode", () => {
    expect(classifyHealth(false, true)).toBe("down");
  });

  it("falls back to mock when unreachable and not in live mode", () => {
    expect(classifyHealth(false, false)).toBe("mock");
  });
});
