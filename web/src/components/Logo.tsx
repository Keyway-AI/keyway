import type { SVGProps } from "react";

/**
 * The Keyway mark: a keyway — the profiled slot a key turns in. A rounded badge
 * with a keyhole knockout (circle + tapered ward). Geometric, ownable, and legible
 * from 20px up. LogoBadge is the filled app/nav mark; LogoGlyph is the bare
 * keyhole in currentColor.
 */
export function LogoBadge({ size = 28, className = "" }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      className={className}
      role="img"
      aria-label="Keyway"
    >
      <defs>
        <linearGradient id="kw-grad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="color-mix(in srgb, var(--color-accent) 88%, white)" />
          <stop offset="1" stopColor="var(--color-accent)" />
        </linearGradient>
      </defs>
      {/* badge + keyhole knockout via evenodd */}
      <path
        fill="url(#kw-grad)"
        fillRule="evenodd"
        d="M9 1h14a8 8 0 0 1 8 8v14a8 8 0 0 1-8 8H9a8 8 0 0 1-8-8V9a8 8 0 0 1 8-8Zm7 8.2a3.7 3.7 0 0 0-1.7 7l-1 6.1a.7.7 0 0 0 .7.85h4a.7.7 0 0 0 .7-.85l-1-6.1A3.7 3.7 0 0 0 16 9.2Z"
        clipRule="evenodd"
      />
    </svg>
  );
}

export function LogoGlyph(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 32 32" fill="currentColor" {...props}>
      <path
        fillRule="evenodd"
        d="M16 5a5 5 0 0 0-2.4 9.4l-1.4 8.5a1 1 0 0 0 1 1.1h5.6a1 1 0 0 0 1-1.1l-1.4-8.5A5 5 0 0 0 16 5Zm0 3a2 2 0 1 1 0 4 2 2 0 0 1 0-4Z"
        clipRule="evenodd"
      />
    </svg>
  );
}

export function Wordmark({ size = "md" }: { size?: "sm" | "md" }) {
  const badge = size === "sm" ? 22 : 26;
  const text = size === "sm" ? "text-[0.95rem]" : "text-[1.1rem]";
  return (
    <span className="flex items-center gap-2.5">
      <LogoBadge size={badge} />
      <span className={`font-semibold tracking-[-0.02em] ${text}`}>Keyway</span>
    </span>
  );
}
