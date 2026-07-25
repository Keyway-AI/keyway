// findings.ts translates raw contract-change events into plain-English
// "findings" a non-specialist can act on. Keyway's whole value is telling a team
// *what changed about who can log into their service and why it matters* — this
// is where the jargon ("expects.audiences widened") becomes a sentence.

import type { ChangeEvent, Severity } from "../api/types";

export interface Finding {
  id: string;
  serviceId: string;
  service: string; // short, human name
  severity: Severity;
  /** one-line headline, e.g. "Stopped requiring a permission claim" */
  headline: string;
  /** what actually changed, in plain words */
  meaning: string;
  /** why it matters — the real-world consequence */
  impact: string;
  /** what a human should do about it */
  action: string;
  /** short category tag for grouping/scanning */
  kind: string;
  /** loosening = accepts more (usually the risky direction); tightening = accepts
   * less (can break existing clients); rotation = affects key-rotation
   * propagation, not which tokens are accepted; neutral = inventory change */
  direction: "loosening" | "tightening" | "rotation" | "neutral" | "unknown";
  event: ChangeEvent;
}

function shortName(consumerID: string): string {
  // Drop the scheme (k8s:, route:, oidc:, url:) and empty segments.
  const parts = consumerID.split("/").filter((p) => p && !p.endsWith(":"));
  if (parts.length === 0) return consumerID;
  const last = parts[parts.length - 1];
  // A trailing generic segment (namespace-style "default") is a poor name; prefer
  // the segment before it when one exists.
  if ((last === "default" || last === "prod" || last === "main") && parts.length >= 2) {
    return parts[parts.length - 2];
  }
  return last;
}

function asList(v: unknown): string[] {
  if (Array.isArray(v)) return v.map(String);
  if (v == null) return [];
  return [String(v)];
}

// added/removed elements between old and new set-valued fields.
function delta(oldV: unknown, newV: unknown): { added: string[]; removed: string[] } {
  const o = new Set(asList(oldV));
  const n = new Set(asList(newV));
  return {
    added: [...n].filter((x) => !o.has(x)),
    removed: [...o].filter((x) => !n.has(x)),
  };
}

interface Desc {
  headline: string;
  meaning: string;
  impact: string;
  action: string;
  kind: string;
  direction: Finding["direction"];
}

// describe maps one event to its human explanation.
function describe(e: ChangeEvent): Desc {
  const svc = shortName(e.consumer_id);
  const { added, removed } = delta(e.old_value, e.new_value);
  const addedStr = added.join(", ");
  const removedStr = removed.join(", ");

  switch (e.field) {
    case "expects.audiences":
      if (e.class === "widened")
        return {
          kind: "Token audience",
          direction: "loosening",
          headline: `Now accepts tokens for “${addedStr}”`,
          meaning: `${svc} began accepting tokens whose audience is “${addedStr}”, in addition to what it already took.`,
          impact:
            "A token minted for another service could now be replayed against this one. If that audience belongs elsewhere, this is a cross-service token-reuse risk.",
          action: "Confirm this audience is meant to reach this service. If not, remove it from the RequestAuthentication.",
        };
      return {
        kind: "Token audience",
        direction: "tightening",
        headline: `Stopped accepting tokens for “${removedStr}”`,
        meaning: `${svc} no longer accepts tokens whose audience is “${removedStr}”.`,
        impact: "Any client still sending tokens with that audience will start getting 401/403 rejections.",
        action: "Make sure no live client depends on the removed audience before this ships.",
      };

    case "expects.issuers":
      if (e.class === "widened")
        return {
          kind: "Trusted login provider",
          direction: "loosening",
          headline: `Now trusts a new login provider`,
          meaning: `${svc} began trusting tokens issued by “${addedStr}”.`,
          impact:
            "Anyone who can get a token from that issuer can now reach this service. Trusting an extra identity provider is one of the highest-impact widenings.",
          action: "Verify the new issuer is intended and controlled by you. An unexpected issuer here is a serious finding.",
        };
      return {
        kind: "Trusted login provider",
        direction: "tightening",
        headline: `Stopped trusting a login provider`,
        meaning: `${svc} no longer trusts tokens from “${removedStr}”.`,
        impact: "Users who log in via that provider will be locked out of this service.",
        action: "Confirm the removed issuer is truly being retired and no one still logs in through it.",
      };

    case "expects.algorithms":
      if (added.includes("none"))
        return {
          kind: "Signature bypass",
          direction: "loosening",
          headline: `Now accepts UNSIGNED tokens (alg=none)`,
          meaning: `${svc} will accept tokens that carry no signature at all.`,
          impact:
            "This is a critical authentication bypass: anyone can hand-craft a token and be trusted. It is the classic “alg=none” vulnerability.",
          action: "Remove “none” from the accepted algorithms immediately. There is almost never a legitimate reason for it.",
        };
      if (e.class === "widened")
        return {
          kind: "Signing algorithm",
          direction: "loosening",
          headline: `Now accepts an additional signing algorithm (${addedStr})`,
          meaning: `${svc} began accepting tokens signed with “${addedStr}”.`,
          impact: "Extra algorithms widen the attack surface — e.g. an HS/RS mix-up can enable key-confusion forgery.",
          action: "Confirm the added algorithm is required and matches how your issuer actually signs.",
        };
      return {
        kind: "Signing algorithm",
        direction: "tightening",
        headline: `Stopped accepting a signing algorithm (${removedStr})`,
        meaning: `${svc} no longer accepts tokens signed with “${removedStr}”.`,
        impact: "Tokens signed with the removed algorithm will be rejected.",
        action: "Ensure your issuer no longer signs with the removed algorithm.",
      };

    case "expects.required_claims":
      if (e.class === "widened")
        return {
          kind: "Permission check",
          direction: "loosening",
          headline: `Stopped requiring a permission claim (${removedStr})`,
          meaning: `${svc} used to reject tokens that lacked the “${removedStr}” claim; now it accepts them.`,
          impact:
            "A dropped required claim usually means an authorization check was removed — tokens that should not have access now get in. This is critical.",
          action: "Restore the required claim unless this relaxation was a deliberate, reviewed decision.",
        };
      return {
        kind: "Permission check",
        direction: "tightening",
        headline: `Now requires an additional claim (${addedStr})`,
        meaning: `${svc} began requiring the “${addedStr}” claim on every token.`,
        impact: "Existing tokens that don't carry this claim will be rejected until issuers add it.",
        action: "Coordinate with the token issuer so valid tokens include the new claim before this ships.",
      };

    case "jwks_behavior.refreshes_on_unknown_kid":
      if (e.new_value === false || String(e.new_value) === "false")
        return {
          kind: "Key-rotation readiness",
          direction: "rotation",
          headline: `Won't pick up rotated signing keys on demand`,
          meaning: `${svc} stopped re-fetching the key set when it sees a token signed by an unknown key.`,
          impact:
            "When you rotate signing keys, this service will reject every new token until its cache expires — a self-inflicted outage. This is the OpenFGA-class failure.",
          action: "Plan a grace period around any key rotation, or enable refresh-on-unknown-kid in the JWKS client.",
        };
      return {
        kind: "Key-rotation readiness",
        direction: "rotation",
        headline: `Now re-fetches keys on an unknown key id`,
        meaning: `${svc} will now refresh its key set when it encounters an unknown signing key.`,
        impact: "This is safer — key rotations propagate faster.",
        action: "No action needed; this improves rotation resilience.",
      };

    case "jwks_behavior.cache_ttl_sec": {
      const inc = Number(e.new_value) > Number(e.old_value);
      return {
        kind: "Key-cache duration",
        direction: "rotation",
        headline: inc ? `Caches signing keys longer` : `Caches signing keys for less time`,
        meaning: `${svc} changed how long it caches the signing key set (${e.old_value}s → ${e.new_value}s).`,
        impact: inc
          ? "A rotated key now takes longer to be picked up, so any rotation needs a longer safety window."
          : "Rotations propagate faster, but the service hits the issuer's key endpoint more often.",
        action: inc ? "Extend the grace period on your next key rotation to match the longer cache." : "No action needed.",
      };
    }

    case "expects.clock_skew_sec":
      return {
        kind: "Clock tolerance",
        direction: e.class === "widened" ? "loosening" : "tightening",
        headline: e.class === "widened" ? `Allows more clock skew` : `Allows less clock skew`,
        meaning: `${svc} changed how far outside its validity window a token may be (${e.old_value}s → ${e.new_value}s).`,
        impact:
          e.class === "widened"
            ? "Expired or not-yet-valid tokens are accepted for a wider window — a small widening of trust."
            : "Tokens near the edge of their lifetime may now be rejected.",
        action: "Confirm the skew matches your infrastructure's real clock drift.",
      };

    case "consumer":
      if (e.new_value != null && e.old_value == null)
        return {
          kind: "Inventory",
          direction: "neutral",
          headline: `New service discovered`,
          meaning: `${svc} appeared in discovery and now validates tokens.`,
          impact: "Not a risk by itself, but a newly onboarded service worth confirming coverage for.",
          action: "Confirm the new service's token contract looks right.",
        };
      return {
        kind: "Inventory",
        direction: "neutral",
        headline: `Service no longer present`,
        meaning: `${svc} disappeared from discovery.`,
        impact: "Not a risk by itself; useful for closing out its rotation tracking.",
        action: "Confirm the service was intentionally decommissioned.",
      };
  }

  // Unknown / low-confidence fallback.
  return {
    kind: "Low-confidence change",
    direction: "unknown",
    headline: `A low-confidence change was noticed`,
    meaning: `${svc} changed in a way Keyway can't classify confidently (${e.field}).`,
    impact: "Kept for the record; it does not page because the confidence is below the alerting floor.",
    action: "Increase discovery confidence for this field (e.g. confirm the source) if you want it to alert.",
  };
}

/** toFinding turns a raw change event into a human finding. */
export function toFinding(e: ChangeEvent): Finding {
  const d = describe(e);
  return {
    id: e.id,
    serviceId: e.consumer_id,
    service: shortName(e.consumer_id),
    severity: e.severity,
    ...d,
    event: e,
  };
}

export const severityRank: Record<Severity, number> = {
  critical: 0,
  high: 1,
  medium: 2,
  low: 3,
  info: 4,
};

export interface ServiceFindings {
  serviceId: string;
  service: string;
  findings: Finding[];
  worst: Severity;
}

/** groupByService buckets findings per service, worst-severity first. */
export function groupByService(findings: Finding[]): ServiceFindings[] {
  const byService = new Map<string, Finding[]>();
  for (const f of findings) {
    const arr = byService.get(f.serviceId) ?? [];
    arr.push(f);
    byService.set(f.serviceId, arr);
  }
  const groups: ServiceFindings[] = [];
  for (const [serviceId, fs] of byService) {
    fs.sort((a, b) => severityRank[a.severity] - severityRank[b.severity]);
    groups.push({ serviceId, service: fs[0].service, findings: fs, worst: fs[0].severity });
  }
  groups.sort((a, b) => severityRank[a.worst] - severityRank[b.worst]);
  return groups;
}

export function countBySeverity(findings: Finding[]): Record<Severity, number> {
  const counts: Record<Severity, number> = { critical: 0, high: 0, medium: 0, low: 0, info: 0 };
  for (const f of findings) counts[f.severity]++;
  return counts;
}
