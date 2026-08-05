import { useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import { Page } from "../components/Layout";
import {
  Button,
  Card,
  Empty,
  Field,
  KeyStatusPill,
  Modal,
  Select,
  Table,
  Td,
  Th,
} from "../components/ui";
import { useToast } from "../components/toast";
import { useAsync } from "../lib/useAsync";
import { relativeTime } from "../lib/format";
import type { IssuerInfo } from "../api/types";

const ALGS = ["RS256", "RS384", "RS512", "ES256", "ES384", "PS256"];

// Canary lifecycle board (PRD §M6). Announce a new key into JWKS without signing
// (announced), watch which consumers pick it up, then promote it to signing
// (active). Every promotion tightens the measured grace window.
export default function Canary() {
  const toast = useToast();
  const issuers = useAsync(() => api.issuers());
  const [selected, setSelected] = useState<string>("");

  const issuerId = selected || issuers.data?.[0]?.id || issuers.data?.[0]?.name || "";
  const issuer = issuers.data?.find((i) => (i.id ?? i.name) === issuerId);

  const status = useAsync(() => (issuerId ? api.canaryStatus(issuerId) : Promise.resolve(null)), [issuerId]);

  const [announceOpen, setAnnounceOpen] = useState(false);
  const [alg, setAlg] = useState("RS256");
  const [busyKid, setBusyKid] = useState<string>("");
  const [announcing, setAnnouncing] = useState(false);

  const keys = status.data?.keys ?? [];
  const announcedKid = status.data?.announced_kid ?? "";

  async function announce() {
    setAnnouncing(true);
    try {
      const key = await api.canaryAnnounce(issuerId, alg);
      toast.success(`Announced ${key.kid} (${key.alg}) — not yet signing`);
      setAnnounceOpen(false);
      status.reload();
      issuers.reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Announce failed");
    } finally {
      setAnnouncing(false);
    }
  }

  async function promote(kid: string) {
    setBusyKid(kid);
    try {
      await api.canaryPromote(issuerId, kid);
      toast.success(`Promoted ${kid} to active signing key`);
      status.reload();
      issuers.reload();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Promote failed");
    } finally {
      setBusyKid("");
    }
  }

  return (
    <Page
      title="Canary"
      subtitle="Announce a key in JWKS without signing, then measure who picks it up. Promoting tightens the grace period on the next rotation."
      actions={
        <div className="flex items-center gap-3">
          <IssuerPicker issuers={issuers.data ?? []} value={issuerId} onChange={setSelected} />
          <Button variant="primary" onClick={() => setAnnounceOpen(true)} disabled={!issuerId}>
            + Announce key
          </Button>
        </div>
      }
    >
      <Card title="Keys in JWKS">
        {keys.length > 0 ? (
          <Table minWidth="min-w-[36rem]">
            <thead>
              <tr>
                <Th>Key ID</Th>
                <Th>Algorithm</Th>
                <Th>Status</Th>
                <Th>First seen</Th>
                <Th />
              </tr>
            </thead>
            <tbody>
              {keys.map((k) => (
                <tr key={k.kid}>
                  <Td className="font-mono text-xs">{k.kid}</Td>
                  <Td>{k.alg}</Td>
                  <Td>
                    <KeyStatusPill status={k.status} />
                  </Td>
                  <Td className="text-muted">
                    {k.first_seen_in_jwks ? relativeTime(k.first_seen_in_jwks) : "—"}
                  </Td>
                  <Td className="text-right">
                    {k.status === "announced" && (
                      <Button
                        variant="secondary"
                        loading={busyKid === k.kid}
                        onClick={() => promote(k.kid)}
                      >
                        Promote to signing
                      </Button>
                    )}
                  </Td>
                </tr>
              ))}
            </tbody>
          </Table>
        ) : (
          <Empty>
            {status.loading
              ? "Loading key state…"
              : issuerId
                ? "No keys yet. Announce a canary key to begin a rotation."
                : "No issuer configured. Add one in your Keyway config to drive the canary flow."}
          </Empty>
        )}
      </Card>

      <Card title="Pickup by consumer" className="mt-6">
        <PickupPanel issuer={issuer} announcedKid={announcedKid} />
      </Card>

      {announceOpen && (
        <Modal
          title={`Announce a canary key · ${issuerId}`}
          onClose={() => setAnnounceOpen(false)}
          footer={
            <>
              <Button variant="ghost" onClick={() => setAnnounceOpen(false)}>
                Cancel
              </Button>
              <Button variant="primary" loading={announcing} onClick={announce}>
                Announce
              </Button>
            </>
          }
        >
          <p className="mb-4 text-sm text-muted">
            A new key is published in the issuer&apos;s JWKS but is <strong>not used to sign</strong>{" "}
            tokens. Consumers that refresh their key set will pick it up; those that don&apos;t are the
            ones a real rotation would break.
          </p>
          <Field label="Signing algorithm">
            <Select value={alg} onChange={(e) => setAlg(e.target.value)}>
              {ALGS.map((a) => (
                <option key={a} value={a}>
                  {a}
                </option>
              ))}
            </Select>
          </Field>
        </Modal>
      )}
    </Page>
  );
}

function IssuerPicker({
  issuers,
  value,
  onChange,
}: {
  issuers: IssuerInfo[];
  value: string;
  onChange: (v: string) => void;
}) {
  if (issuers.length <= 1) {
    return value ? <span className="text-sm text-muted">issuer: {value}</span> : null;
  }
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="rounded-lg border border-border bg-surface-2 px-3 py-2 text-sm outline-none focus:border-brand"
    >
      {issuers.map((i) => {
        const id = i.id ?? i.name;
        return (
          <option key={id} value={id}>
            {i.name}
          </option>
        );
      })}
    </select>
  );
}

// PickupPanel lists the consumers a rotation would test for adoption of the
// announced key. Per-consumer pickup is *measured by running canary probes over
// time* — Keyway derives the grace window from the announce→pickup latency — so
// we show these as "awaiting probe" and link to the probe cycle rather than
// fabricating an adoption status.
function PickupPanel({ issuer, announcedKid }: { issuer?: IssuerInfo; announcedKid: string }) {
  const consumers = useAsync(() => api.consumers());
  const probeable = (consumers.data ?? []).filter((c) => c.probeable);

  if (!announcedKid) {
    return <Empty>No key announced. Announce one to start measuring pickup.</Empty>;
  }
  if (probeable.length === 0) {
    return <Empty>{consumers.loading ? "Loading…" : "No probeable consumers to measure."}</Empty>;
  }
  return (
    <>
      <p className="mb-3 text-xs text-muted">
        {probeable.length} consumer(s) will be tested for adoption of{" "}
        <span className="font-mono text-text">{announcedKid}</span>
        {issuer ? ` on ${issuer.name}` : ""}. Pickup is measured by running a{" "}
        <Link to="/app/probes" className="font-medium text-accent hover:underline">
          probe cycle
        </Link>{" "}
        over time — the grace window comes from the announce→pickup latency.
      </p>
      <ul className="space-y-2">
        {probeable.map((c) => (
          <li
            key={c.stable_id}
            className="flex items-center justify-between rounded-lg border border-border bg-surface px-4 py-3"
          >
            <div>
              <div className="font-medium">{c.name}</div>
              <div className="text-xs text-muted">{c.stable_id}</div>
            </div>
            <span className="text-xs text-muted">awaiting probe</span>
          </li>
        ))}
      </ul>
    </>
  );
}
