import { useMemo, useState } from "react";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Button, Card, Empty, Field, Input, Pill, Select, StatTile } from "../components/ui";
import { IssuerNotice } from "../components/IssuerNotice";
import { useToast } from "../components/toast";
import { useAsync } from "../lib/useAsync";
import { humanizeDuration, verdictColor } from "../lib/format";
import type { AffectedConsumer, BlastRadiusResult } from "../api/types";

type Kind = "rotate_key" | "retire_key" | "remove_claim" | "change_issuer" | "drop_algorithm";

const KINDS: { value: Kind; label: string; blurb: string }[] = [
  { value: "rotate_key", label: "Rotate signing key", blurb: "Introduce a new kid and stop signing with the old one." },
  { value: "retire_key", label: "Retire a key", blurb: "Remove a kid from JWKS entirely." },
  { value: "remove_claim", label: "Remove a claim", blurb: "Stop issuing a claim some consumers require." },
  { value: "change_issuer", label: "Change issuer URL", blurb: "Move to a new issuer identifier / discovery URL." },
  { value: "drop_algorithm", label: "Drop an algorithm", blurb: "Stop signing with an algorithm consumers may pin." },
];

function VerdictGroup({
  title,
  verdict,
  rows,
}: {
  title: string;
  verdict: AffectedConsumer["verdict"];
  rows: AffectedConsumer[];
}) {
  if (rows.length === 0) return null;
  return (
    <div className="mb-5">
      <h3 className={`mb-2 text-sm font-semibold ${verdictColor[verdict]}`}>
        {title} ({rows.length})
      </h3>
      <div className="space-y-2">
        {rows.map((a) => (
          <div
            key={a.consumer.stable_id}
            className="flex items-start justify-between gap-4 rounded-lg border border-border bg-surface px-4 py-3"
          >
            <div>
              <div className="font-medium">{a.consumer.name}</div>
              <div className="text-xs text-muted">{a.reason}</div>
              {a.consumer.owner_team && (
                <div className="mt-1 text-xs text-muted">owner: {a.consumer.owner_team}</div>
              )}
            </div>
            <div className="flex flex-col items-end gap-1">
              {a.evidence.map((e) => (
                <Pill key={e} className="font-mono text-[10px]">
                  {e}
                </Pill>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default function BlastRadius() {
  const toast = useToast();
  const issuers = useAsync(() => api.issuers());
  const [kind, setKind] = useState<Kind>("rotate_key");
  const [issuerId, setIssuerId] = useState("");
  const [kid, setKid] = useState("rsa-2026-01");
  const [claim, setClaim] = useState("dept");
  const [newIssuer, setNewIssuer] = useState("https://auth.new.example/realms/main");
  const [algorithm, setAlgorithm] = useState("RS256");
  const [result, setResult] = useState<BlastRadiusResult>();
  const [computing, setComputing] = useState(false);

  const effectiveIssuer = issuerId || issuers.data?.[0]?.id || issuers.data?.[0]?.name || "";
  const kindMeta = KINDS.find((k) => k.value === kind)!;
  // Key- and issuer-scoped changes resolve against a specific issuer; claim/alg
  // changes resolve against the discovered contract regardless of issuer.
  const needsIssuer = kind === "rotate_key" || kind === "retire_key" || kind === "change_issuer";
  const hasIssuer = (issuers.data?.length ?? 0) > 0;

  async function run() {
    setComputing(true);
    try {
      const proposal: BlastRadiusResult["proposal"] & {
        new_issuer_url?: string;
        algorithm?: string;
      } = { kind, issuer_id: effectiveIssuer };
      if (kind === "rotate_key" || kind === "retire_key") proposal.kid = kid;
      if (kind === "remove_claim") proposal.claim_name = claim;
      if (kind === "change_issuer") proposal.new_issuer_url = newIssuer;
      if (kind === "drop_algorithm") proposal.algorithm = algorithm;
      const res = await api.blastRadius(proposal);
      setResult(res);
      const breaks = res.affected.filter((a) => a.verdict === "will_break").length;
      if (breaks > 0) toast.error(`${breaks} consumer(s) would break`);
      else toast.success("No consumers break under this change");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Blast-radius computation failed");
    } finally {
      setComputing(false);
    }
  }

  const willBreak = result?.affected.filter((a) => a.verdict === "will_break") ?? [];
  const ready = result?.affected.filter((a) => a.verdict === "ready") ?? [];
  const unknown = result?.affected.filter((a) => a.verdict === "unknown") ?? [];

  const summary = useMemo(() => {
    if (!result) return "";
    switch (result.proposal.kind) {
      case "rotate_key":
      case "retire_key":
        return `${kindMeta.label} · ${result.proposal.kid}`;
      case "remove_claim":
        return `${kindMeta.label} · ${result.proposal.claim_name}`;
      default:
        return kindMeta.label;
    }
  }, [result, kindMeta]);

  return (
    <Page
      title="Blast radius"
      subtitle="Model a proposed change and see who breaks, who's ready, and the safe grace period — before you ship it."
    >
      {needsIssuer && !hasIssuer && !issuers.loading && <IssuerNotice action="Modeling a key rotation" />}

      <Card className="mb-6">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Field label="Proposed change" hint={kindMeta.blurb}>
            <Select value={kind} onChange={(e) => setKind(e.target.value as Kind)}>
              {KINDS.map((k) => (
                <option key={k.value} value={k.value}>
                  {k.label}
                </option>
              ))}
            </Select>
          </Field>

          {(issuers.data?.length ?? 0) > 1 && (
            <Field label="Issuer">
              <Select value={effectiveIssuer} onChange={(e) => setIssuerId(e.target.value)}>
                {issuers.data?.map((i) => {
                  const id = i.id ?? i.name;
                  return (
                    <option key={id} value={id}>
                      {i.name}
                    </option>
                  );
                })}
              </Select>
            </Field>
          )}

          {(kind === "rotate_key" || kind === "retire_key") && (
            <Field label="Key ID (kid)">
              <Input value={kid} onChange={(e) => setKid(e.target.value)} className="font-mono" />
            </Field>
          )}
          {kind === "remove_claim" && (
            <Field label="Claim name">
              <Input value={claim} onChange={(e) => setClaim(e.target.value)} className="font-mono" />
            </Field>
          )}
          {kind === "change_issuer" && (
            <Field label="New issuer URL">
              <Input value={newIssuer} onChange={(e) => setNewIssuer(e.target.value)} className="font-mono" />
            </Field>
          )}
          {kind === "drop_algorithm" && (
            <Field label="Algorithm to drop">
              <Select value={algorithm} onChange={(e) => setAlgorithm(e.target.value)}>
                {["RS256", "RS384", "RS512", "ES256", "ES384", "PS256", "HS256"].map((a) => (
                  <option key={a} value={a}>
                    {a}
                  </option>
                ))}
              </Select>
            </Field>
          )}
        </div>
        <div className="mt-4">
          <Button variant="primary" onClick={run} loading={computing} disabled={needsIssuer && !effectiveIssuer}>
            Compute blast radius
          </Button>
        </div>
      </Card>

      {result ? (
        <>
          <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4 sm:gap-4">
            <StatTile label="Affected" value={result.affected.length} />
            <StatTile label="Will break" value={willBreak.length} accent="text-critical" />
            <StatTile label="Ready" value={ready.length} accent="text-low" />
            <StatTile
              label="Grace period"
              value={result.recommended_grace_period_seconds ? humanizeDuration(result.recommended_grace_period_seconds) : "—"}
              hint={result.grace_basis ? `bound by ${result.grace_basis.split("/").pop()}` : undefined}
              accent="text-brand"
            />
          </div>

          <Card title={summary}>
            <VerdictGroup title="Will break" verdict="will_break" rows={willBreak} />
            <VerdictGroup title="Ready" verdict="ready" rows={ready} />
            <VerdictGroup title="Unknown" verdict="unknown" rows={unknown} />
            {willBreak.length === 0 && ready.length === 0 && unknown.length === 0 && (
              <Empty>No consumers are affected by this change.</Empty>
            )}
            {unknown.length > 0 && (
              <p className="mt-2 text-xs text-medium">
                {unknown.length} consumer(s) unknown — treat the grace period as a lower bound.
              </p>
            )}
          </Card>
        </>
      ) : (
        <Card>
          <Empty>
            {issuers.loading
              ? "Loading issuers…"
              : "Pick a proposed change and press Compute. Will-break / ready verdicts and a recommended grace period appear here."}
          </Empty>
        </Card>
      )}
    </Page>
  );
}
