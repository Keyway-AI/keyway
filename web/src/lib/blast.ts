import type { AffectedConsumer, BlastRadiusResult, Consumer } from "../api/types";

// Client-side approximation of the rotate_key resolution (PRD §10.2) and grace
// period (§10.3), used until POST /v1/blast-radius lands (M7). The real answer
// comes from the backend, which also folds in canary probe evidence.

const HOUR = 3600;
const DAY = 86400;

// computeBlast is a client-side approximation of POST /v1/blast-radius across all
// proposal kinds — used as the offline/mock fallback. The authoritative answer
// (which also folds in live canary-probe evidence) comes from the backend.
export function computeBlast(
  consumers: Consumer[],
  proposal: BlastRadiusResult["proposal"],
): BlastRadiusResult {
  if (proposal.kind === "rotate_key" || proposal.kind === "retire_key") {
    const r = computeRotateKey(consumers, proposal.kid ?? "");
    return { ...r, proposal };
  }
  const affected: AffectedConsumer[] = [];
  for (const c of consumers) {
    let hit = false;
    let reason = "";
    if (proposal.kind === "remove_claim" && c.expects.required_claims.includes(proposal.claim_name ?? "")) {
      hit = true;
      reason = `requires claim "${proposal.claim_name}"`;
    } else if (proposal.kind === "change_issuer" && c.expects.issuers.includes(proposal.issuer_id ?? "")) {
      hit = true;
      reason = `trusts the old issuer "${proposal.issuer_id}"`;
    } else if (
      proposal.kind === "drop_algorithm" &&
      c.expects.algorithms.length === 1 &&
      c.expects.algorithms[0] === proposal.algorithm
    ) {
      hit = true;
      reason = `only accepts "${proposal.algorithm}"`;
    }
    if (hit) {
      affected.push({ consumer: c, verdict: "will_break", reason, evidence: ["expects.*"], confidence: 1 });
    }
  }
  return {
    proposal,
    affected,
    unknown: [],
    recommended_grace_period_seconds: 0,
    grace_basis: "",
    generated_at: new Date().toISOString(),
  };
}

export function computeRotateKey(consumers: Consumer[], kid: string): BlastRadiusResult {
  const affected: AffectedConsumer[] = [];
  const unknown: Consumer[] = [];
  const readyWindows: { c: Consumer; window: number }[] = [];

  for (const c of consumers) {
    if (!c.probeable) {
      unknown.push(c);
      affected.push({ consumer: c, verdict: "unknown", reason: "not probeable — insufficient evidence", evidence: [], confidence: 0 });
      continue;
    }
    const jwks = c.jwks_behavior;
    if (jwks.refreshes_on_unknown_kid === false) {
      const lib = c.library ? `lib:${c.library.name} ${c.library.version}` : "config";
      const cacheNote = jwks.cache_ttl_sec ? `${Math.round(jwks.cache_ttl_sec / HOUR)}h JWKS cache, ` : "";
      affected.push({
        consumer: c,
        verdict: "will_break",
        reason: `${cacheNote}RefreshUnknownKID=false`,
        evidence: [lib],
        confidence: 0.8,
      });
      continue;
    }
    if (jwks.cache_ttl_sec != null) {
      readyWindows.push({ c, window: jwks.cache_ttl_sec });
      affected.push({
        consumer: c,
        verdict: "ready",
        reason: `refreshes on unknown kid; ${Math.round(jwks.cache_ttl_sec / 60)}m cache`,
        evidence: [jwks.source],
        confidence: 0.7,
      });
      continue;
    }
    unknown.push(c);
    affected.push({ consumer: c, verdict: "unknown", reason: "no JWKS behaviour known", evidence: [], confidence: 0 });
  }

  // Grace: max ready window * 1.5, floor 1h, ceil 30d.
  let maxWindow = 0;
  let basis = "";
  for (const r of readyWindows) {
    if (r.window > maxWindow) {
      maxWindow = r.window;
      basis = r.c.stable_id;
    }
  }
  let grace = Math.round(maxWindow * 1.5);
  grace = Math.max(HOUR, Math.min(30 * DAY, grace));

  return {
    proposal: { kind: "rotate_key", issuer_id: "", kid },
    affected,
    unknown,
    recommended_grace_period_seconds: grace,
    grace_basis: basis,
    generated_at: new Date().toISOString(),
  };
}
