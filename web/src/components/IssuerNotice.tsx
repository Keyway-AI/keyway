/**
 * Guidance shown when an issuer-dependent feature (probes, key rotation, canary)
 * has no issuer configured. Keyway mints test tokens with an operated issuer key,
 * which is registered via config / the CLI — not the web UI — so this points the
 * user at the right setup step instead of letting the action fail silently.
 */
export function IssuerNotice({ action }: { action: string }) {
  return (
    <div className="mb-6 flex items-start gap-3 rounded-xl border border-medium/40 bg-medium/5 p-4">
      <svg
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="mt-0.5 shrink-0 text-medium"
        aria-hidden
      >
        <path d="M12 9v4M12 17h.01" />
        <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
      </svg>
      <div className="min-w-0">
        <div className="text-sm font-semibold text-text">No issuer configured</div>
        <p className="mt-1 text-sm text-muted">
          {action} needs an issuer so Keyway can mint the test tokens. Register one in your Keyway
          config or with <code className="font-mono text-xs text-text">keyway issuer add</code>, then
          restart. See the{" "}
          <a
            href="https://github.com/nometria/keyway#quickstart"
            className="font-medium text-accent hover:underline"
          >
            setup guide
          </a>
          .
        </p>
      </div>
    </div>
  );
}
