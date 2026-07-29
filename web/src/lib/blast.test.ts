import { describe, expect, it } from "vitest";
import type { Consumer } from "../api/types";
import { computeBlast, computeRotateKey } from "./blast";

function mkConsumer(over: Partial<Consumer> & { stable_id: string }): Consumer {
  const sid = over.stable_id;
  return {
    id: sid,
    kind: "service",
    name: sid,
    endpoints: [],
    expects: {
      issuers: ["https://idp.example.com"],
      audiences: ["api"],
      algorithms: ["RS256"],
      required_claims: [],
      clock_skew_sec: 60,
    },
    jwks_behavior: { source: "config", refreshes_on_unknown_kid: true, cache_ttl_sec: 600 },
    confidence: {},
    probeable: true,
    ...over,
  };
}

describe("computeRotateKey", () => {
  it("marks a consumer that won't refresh on unknown kid as will_break", () => {
    const c = mkConsumer({
      stable_id: "mobile-gw",
      jwks_behavior: { source: "config", refreshes_on_unknown_kid: false, cache_ttl_sec: 6 * 3600 },
    });
    const r = computeRotateKey([c], "2024-06");
    expect(r.affected[0].verdict).toBe("will_break");
    expect(r.affected[0].reason).toMatch(/RefreshUnknownKID=false/);
  });

  it("marks a refreshing consumer as ready", () => {
    const r = computeRotateKey([mkConsumer({ stable_id: "orders" })], "2024-06");
    expect(r.affected[0].verdict).toBe("ready");
  });

  it("treats non-probeable consumers as unknown", () => {
    const c = mkConsumer({ stable_id: "legacy", probeable: false });
    const r = computeRotateKey([c], "2024-06");
    expect(r.affected[0].verdict).toBe("unknown");
    expect(r.unknown).toHaveLength(1);
  });

  it("derives the grace period as 1.5x the widest ready cache, floored at 1h", () => {
    // widest ready window is 600s -> 900s, but floor is 3600s
    const r = computeRotateKey([mkConsumer({ stable_id: "orders" })], "k");
    expect(r.recommended_grace_period_seconds).toBe(3600);

    // a 10h cache -> 15h grace, above the floor
    const big = mkConsumer({
      stable_id: "slow",
      jwks_behavior: { source: "config", refreshes_on_unknown_kid: true, cache_ttl_sec: 10 * 3600 },
    });
    const r2 = computeRotateKey([big], "k");
    expect(r2.recommended_grace_period_seconds).toBe(Math.round(10 * 3600 * 1.5));
    expect(r2.grace_basis).toBe("slow");
  });
});

describe("computeBlast — non-rotation proposals", () => {
  it("breaks consumers that require a claim being removed", () => {
    const c = mkConsumer({
      stable_id: "reports",
      expects: {
        issuers: ["i"],
        audiences: ["api"],
        algorithms: ["RS256"],
        required_claims: ["dept"],
        clock_skew_sec: 60,
      },
    });
    const r = computeBlast([c], { kind: "remove_claim", issuer_id: "", claim_name: "dept" });
    expect(r.affected).toHaveLength(1);
    expect(r.affected[0].verdict).toBe("will_break");
    expect(r.affected[0].reason).toContain("dept");
  });

  it("only breaks on drop_algorithm when it's the consumer's sole algorithm", () => {
    const soleRS = mkConsumer({ stable_id: "a", expects: { issuers: ["i"], audiences: ["x"], algorithms: ["RS256"], required_claims: [], clock_skew_sec: 0 } });
    const multi = mkConsumer({ stable_id: "b", expects: { issuers: ["i"], audiences: ["x"], algorithms: ["RS256", "ES256"], required_claims: [], clock_skew_sec: 0 } });
    const r = computeBlast([soleRS, multi], { kind: "drop_algorithm", issuer_id: "", algorithm: "RS256" });
    expect(r.affected.map((a) => a.consumer.stable_id)).toEqual(["a"]);
  });

  it("breaks consumers that trust an issuer being changed", () => {
    const c = mkConsumer({ stable_id: "c", expects: { issuers: ["old-idp"], audiences: ["x"], algorithms: ["RS256"], required_claims: [], clock_skew_sec: 0 } });
    const r = computeBlast([c], { kind: "change_issuer", issuer_id: "old-idp" });
    expect(r.affected).toHaveLength(1);
    expect(r.affected[0].reason).toContain("old-idp");
  });

  it("reports no impact when nothing matches", () => {
    const c = mkConsumer({ stable_id: "safe" });
    const r = computeBlast([c], { kind: "remove_claim", issuer_id: "", claim_name: "nonexistent" });
    expect(r.affected).toHaveLength(0);
  });
});
