import { api } from "../api/client";
import { Page } from "../components/Layout";
import { Card, Empty, Pill } from "../components/ui";
import { useAsync } from "../lib/useAsync";

// Canary status board. The announce/promote actions are wired to the API in M6;
// this view shows which consumers have picked up an announced (canary) key.
export default function Canary() {
  const consumers = useAsync(() => api.consumers());

  return (
    <Page
      title="Canary"
      subtitle="Announce a key in JWKS without signing, then measure who picks it up. Measured windows tighten the grace period on every run."
    >
      <Card
        title="Announced keys"
        action={
          <button className="rounded-lg border border-border bg-surface-2 px-3 py-1.5 text-xs text-muted hover:text-text">
            + Announce key (M6)
          </button>
        }
      >
        <Empty>No canary key announced. Start one with `keyway canary start --issuer … --alg RS256`.</Empty>
      </Card>

      <Card title="Pickup by consumer" className="mt-6">
        {consumers.data && consumers.data.length > 0 ? (
          <ul className="space-y-2">
            {consumers.data
              .filter((c) => c.probeable)
              .map((c) => (
                <li
                  key={c.stable_id}
                  className="flex items-center justify-between rounded-lg border border-border bg-surface px-4 py-3"
                >
                  <div>
                    <div className="font-medium">{c.name}</div>
                    <div className="text-xs text-muted">{c.stable_id}</div>
                  </div>
                  <Pill className="text-muted">awaiting canary</Pill>
                </li>
              ))}
          </ul>
        ) : (
          <Empty>{consumers.loading ? "Loading…" : "No probeable consumers."}</Empty>
        )}
      </Card>
    </Page>
  );
}
