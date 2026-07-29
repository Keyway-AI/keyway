import { describe, expect, it } from "vitest";
import { agentInspect } from "./mock";

// Build a JWT-shaped string (header.payload.sig) whose payload is `claims`.
// The analyzer only reads the payload, so the header/sig are placeholders.
function jwt(claims: Record<string, unknown>): string {
  const payload = Buffer.from(JSON.stringify(claims)).toString("base64url");
  return `eyJhbGciOiJSUzI1NiJ9.${payload}.sig`;
}

const ids = (r: ReturnType<typeof agentInspect>) => r.findings.map((f) => f.threat_id);

describe("agentInspect — audience binding (MCP-01 / MCP-02)", () => {
  it("flags MCP-01 when the audience does not include the resource", () => {
    const r = agentInspect({ token: jwt({ aud: "other-service", exp: 9e9 }), audience: "payments-api" });
    expect(ids(r)).toContain("MCP-01");
    expect(r.findings.find((f) => f.threat_id === "MCP-01")?.severity).toBe("critical");
  });

  it("flags MCP-02 when the token carries no audience at all", () => {
    const r = agentInspect({ token: jwt({ exp: 9e9 }), audience: "payments-api" });
    expect(ids(r)).toContain("MCP-02");
  });

  it("passes audience when the resource is present in an aud array", () => {
    const r = agentInspect({ token: jwt({ aud: ["a", "payments-api"], exp: 9e9 }), audience: "payments-api" });
    expect(ids(r)).not.toContain("MCP-01");
    expect(ids(r)).not.toContain("MCP-02");
  });
});

describe("agentInspect — delegation (DEL-01)", () => {
  it("flags a missing act claim only when delegation is required", () => {
    const withReq = agentInspect({ token: jwt({ exp: 9e9 }), require_delegation: true });
    expect(ids(withReq)).toContain("DEL-01");

    const withoutReq = agentInspect({ token: jwt({ exp: 9e9 }), require_delegation: false });
    expect(ids(withoutReq)).not.toContain("DEL-01");
  });

  it("accepts a present act claim", () => {
    const r = agentInspect({ token: jwt({ exp: 9e9, act: { sub: "user-1" } }), require_delegation: true });
    expect(ids(r)).not.toContain("DEL-01");
  });
});

describe("agentInspect — scope hygiene (SCOPE-01)", () => {
  it("flags omnibus and wildcard scopes as over-broad", () => {
    expect(ids(agentInspect({ token: jwt({ exp: 9e9, scope: "files:*" }) }))).toContain("SCOPE-01");
    expect(ids(agentInspect({ token: jwt({ exp: 9e9, scope: "admin" }) }))).toContain("SCOPE-01");
    expect(ids(agentInspect({ token: jwt({ exp: 9e9, scp: ["admin:write"] }) }))).toContain("SCOPE-01");
  });

  it("flags a scope outside the declared allowlist", () => {
    const r = agentInspect({ token: jwt({ exp: 9e9, scope: "billing:read" }), allowed_scopes: ["files:read"] });
    const scope01 = r.findings.find((f) => f.threat_id === "SCOPE-01");
    expect(scope01?.severity).toBe("medium");
  });

  it("accepts a least-privilege scope inside the allowlist", () => {
    const r = agentInspect({ token: jwt({ exp: 9e9, scope: "files:read" }), allowed_scopes: ["files:read"] });
    expect(ids(r)).not.toContain("SCOPE-01");
  });
});

describe("agentInspect — expiry (SCOPE-02)", () => {
  it("flags a token with no exp as a non-expiring credential", () => {
    expect(ids(agentInspect({ token: jwt({ sub: "agent" }) }))).toContain("SCOPE-02");
  });

  it("flags a lifetime exceeding the maximum", () => {
    const iat = 1_000_000;
    const r = agentInspect({
      token: jwt({ iat, exp: iat + 7200 }),
      max_lifetime_seconds: 3600,
    });
    expect(ids(r)).toContain("SCOPE-02");
  });

  it("accepts a short-lived token within the maximum", () => {
    const iat = 1_000_000;
    const r = agentInspect({ token: jwt({ iat, exp: iat + 600 }), max_lifetime_seconds: 3600 });
    expect(ids(r)).not.toContain("SCOPE-02");
  });
});

describe("agentInspect — malformed input", () => {
  it("reports a single info finding when the token is not a JWT", () => {
    const r = agentInspect({ token: "not-a-jwt" });
    expect(r.count).toBe(1);
    expect(r.findings[0].severity).toBe("info");
  });

  it("returns a clean report for a well-formed, well-scoped token", () => {
    const iat = 1_000_000;
    const r = agentInspect({
      token: jwt({ aud: ["payments-api"], iat, exp: iat + 300, act: { sub: "u" }, scope: "files:read" }),
      audience: "payments-api",
      require_delegation: true,
      max_lifetime_seconds: 3600,
      allowed_scopes: ["files:read"],
    });
    expect(r.findings).toHaveLength(0);
  });
});
