import { describe, expect, it } from "vitest";
import type { ChangeEvent } from "../api/types";
import { countBySeverity, groupByService, severityRank, toFinding } from "./findings";

function ev(overrides: Partial<ChangeEvent>): ChangeEvent {
  return {
    id: "e1",
    from_version: "v1",
    to_version: "v2",
    consumer_id: "k8s:prod/payments-api",
    field: "expects.audiences",
    old_value: [],
    new_value: [],
    class: "widened",
    severity: "high",
    confidence: 0.9,
    evidence: [],
    detected_at: new Date().toISOString(),
    ...overrides,
  };
}

describe("toFinding — human translation", () => {
  it("flags alg=none as a critical signature bypass regardless of class", () => {
    const f = toFinding(
      ev({ field: "expects.algorithms", old_value: ["RS256"], new_value: ["RS256", "none"] }),
    );
    expect(f.headline).toMatch(/UNSIGNED/i);
    expect(f.kind).toBe("Signature bypass");
    expect(f.direction).toBe("loosening");
  });

  it("reads a dropped required claim as a loosening (auth check removed)", () => {
    const f = toFinding(
      ev({
        field: "expects.required_claims",
        class: "widened",
        old_value: ["dept"],
        new_value: [],
      }),
    );
    expect(f.direction).toBe("loosening");
    expect(f.headline).toMatch(/Stopped requiring/i);
    expect(f.meaning).toContain("dept");
  });

  it("classifies refreshes_on_unknown_kid=false as a rotation risk", () => {
    const f = toFinding(
      ev({ field: "jwks_behavior.refreshes_on_unknown_kid", old_value: true, new_value: false }),
    );
    expect(f.direction).toBe("rotation");
    expect(f.headline).toMatch(/rotated signing keys/i);
  });

  it("derives a short human service name from the consumer id", () => {
    // trailing generic segment 'prod' should defer to the segment before it
    expect(toFinding(ev({ consumer_id: "k8s:prod/payments-api" })).service).toBe("payments-api");
    expect(toFinding(ev({ consumer_id: "route:/gateway/orders/default" })).service).toBe("orders");
  });

  it("falls back to a low-confidence finding for unknown fields", () => {
    const f = toFinding(ev({ field: "expects.something_new", class: "unknown" }));
    expect(f.direction).toBe("unknown");
    expect(f.kind).toBe("Low-confidence change");
  });
});

describe("groupByService", () => {
  it("buckets by service and orders worst-severity first", () => {
    const findings = [
      toFinding(ev({ id: "a", consumer_id: "k8s:prod/orders", severity: "low" })),
      toFinding(ev({ id: "b", consumer_id: "k8s:prod/payments", severity: "critical" })),
      toFinding(ev({ id: "c", consumer_id: "k8s:prod/orders", severity: "high" })),
    ];
    const groups = groupByService(findings);
    expect(groups).toHaveLength(2);
    // payments (critical) sorts ahead of orders (worst = high)
    expect(groups[0].service).toBe("payments");
    expect(groups[0].worst).toBe("critical");
    // within a service, worst-first
    expect(groups[1].findings[0].severity).toBe("high");
  });
});

describe("countBySeverity", () => {
  it("tallies each severity bucket", () => {
    const findings = [
      toFinding(ev({ id: "a", severity: "critical" })),
      toFinding(ev({ id: "b", severity: "critical" })),
      toFinding(ev({ id: "c", severity: "low" })),
    ];
    const counts = countBySeverity(findings);
    expect(counts.critical).toBe(2);
    expect(counts.low).toBe(1);
    expect(counts.high).toBe(0);
  });
});

describe("severityRank", () => {
  it("orders critical highest (lowest number) through info", () => {
    expect(severityRank.critical).toBeLessThan(severityRank.high);
    expect(severityRank.high).toBeLessThan(severityRank.medium);
    expect(severityRank.medium).toBeLessThan(severityRank.low);
    expect(severityRank.low).toBeLessThan(severityRank.info);
  });
});
