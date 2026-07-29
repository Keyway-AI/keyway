import { useState } from "react";
import { getToken, setToken } from "../api/client";
import { Button, Field, Input, Modal } from "./ui";
import { useToast } from "./toast";
import { classifyHealth, HEALTH_POLL_MS, type Health } from "../lib/health";
import { usePolling } from "../lib/usePolling";

// Settings holds connection config the browser client needs: the API bearer
// token and the live/mock toggle (localStorage `keyway.live`). It also probes
// /v1/health so the user can see whether they are talking to a real backend.
export function SettingsModal({ onClose }: { onClose: () => void }) {
  const toast = useToast();
  const [token, setTokenState] = useState(getToken());
  const [live, setLive] = useState(localStorage.getItem("keyway.live") === "1");
  const [health, setHealth] = useState<Health>("checking");
  const [version, setVersion] = useState("");

  // Re-probe while the modal is open so the indicator tracks the backend live.
  usePolling(
    async () => {
      try {
        const r = await fetch("/v1/health", { headers: { Authorization: `Bearer ${getToken()}` } });
        if (!r.ok) throw new Error(String(r.status));
        const h: { version?: string } = await r.json();
        setHealth(classifyHealth(true, localStorage.getItem("keyway.live") === "1"));
        setVersion(h.version ?? "");
      } catch {
        setHealth(classifyHealth(false, localStorage.getItem("keyway.live") === "1"));
      }
    },
    { intervalMs: HEALTH_POLL_MS },
  );

  function save() {
    setToken(token.trim());
    if (live) localStorage.setItem("keyway.live", "1");
    else localStorage.removeItem("keyway.live");
    toast.success("Settings saved");
    onClose();
    // A full reload re-fires every page's data fetch against the new config.
    window.location.reload();
  }

  const badge = {
    checking: { label: "Checking…", cls: "text-muted" },
    ok: { label: `Connected${version ? ` · v${version}` : ""}`, cls: "text-low" },
    down: { label: "Unreachable", cls: "text-critical" },
    mock: { label: "Sample data (no backend)", cls: "text-high" },
  }[health];

  return (
    <Modal
      title="Settings"
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" onClick={save}>
            Save
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <div className="flex items-center justify-between rounded-lg border border-border bg-surface-2 px-4 py-3">
          <span className="text-sm text-muted">Backend</span>
          <span className={`flex items-center gap-2 text-sm font-medium ${badge.cls}`}>
            <span className="h-2 w-2 rounded-full bg-current" />
            {badge.label}
          </span>
        </div>

        <Field label="API token" hint="Sent as a Bearer token on every request.">
          <Input
            type="password"
            value={token}
            onChange={(e) => setTokenState(e.target.value)}
            placeholder="paste your API token"
            autoComplete="off"
          />
        </Field>

        <label className="flex items-start gap-3 rounded-lg border border-border bg-surface-2 px-4 py-3">
          <input
            type="checkbox"
            checked={live}
            onChange={(e) => setLive(e.target.checked)}
            className="mt-0.5 h-4 w-4 accent-[var(--color-brand)]"
          />
          <span className="text-sm">
            <span className="font-medium text-text">Live mode</span>
            <span className="mt-0.5 block text-xs text-muted">
              Talk only to the real backend. When off, pages fall back to sample data if the API is
              unavailable, so the app stays navigable during a demo.
            </span>
          </span>
        </label>
      </div>
    </Modal>
  );
}
