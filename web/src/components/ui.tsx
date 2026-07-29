import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
} from "react";
import { useEffect } from "react";
import type { Severity } from "../api/types";
import { severityDot } from "../lib/format";

/* ── Card ─────────────────────────────────────────────────────────────── */

export function Card({
  title,
  action,
  children,
  className = "",
  bodyClassName = "p-5",
}: {
  title?: ReactNode;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
}) {
  return (
    <section
      className={`rounded-xl border border-border bg-surface shadow-xs ${className}`}
    >
      {(title || action) && (
        <header className="flex items-center justify-between gap-3 border-b border-border px-5 py-3.5">
          <h2 className="text-[0.95rem] font-semibold tracking-tight text-text">{title}</h2>
          {action}
        </header>
      )}
      <div className={bodyClassName}>{children}</div>
    </section>
  );
}

/* ── StatTile ─────────────────────────────────────────────────────────── */

export function StatTile({
  label,
  value,
  hint,
  accent = "text-text",
}: {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  accent?: string;
}) {
  return (
    <div className="rounded-xl border border-border bg-surface p-5 shadow-xs">
      <div className="eyebrow">{label}</div>
      <div className={`mt-2 text-[2rem] font-semibold leading-none tracking-tight tabular-nums ${accent}`}>
        {value}
      </div>
      {hint && <div className="mt-1.5 text-caption text-muted">{hint}</div>}
    </div>
  );
}

/* ── Badges & pills ───────────────────────────────────────────────────── */

export function SeverityBadge({ severity }: { severity: Severity }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-pill border border-border bg-surface px-2.5 py-1 text-caption font-medium">
      <span className={`h-1.5 w-1.5 rounded-full ${severityDot[severity]}`} />
      <span className="capitalize">{severity}</span>
    </span>
  );
}

export function Pill({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <span
      className={`inline-flex items-center rounded-pill border border-border bg-surface-2 px-2.5 py-0.5 text-caption ${className}`}
    >
      {children}
    </span>
  );
}

export function KeyStatusPill({ status }: { status: string }) {
  const styles: Record<string, string> = {
    announced: "border-high/40 text-high",
    active: "border-low/40 text-low",
    retiring: "border-high/40 text-high",
    retired: "border-border text-muted",
  };
  return (
    <span
      className={`inline-flex items-center rounded-pill border bg-surface px-2.5 py-1 text-caption font-medium capitalize ${
        styles[status] ?? "border-border text-muted"
      }`}
    >
      {status}
    </span>
  );
}

/* ── Empty state ──────────────────────────────────────────────────────── */

export function Empty({ children }: { children: ReactNode }) {
  return <div className="py-14 text-center text-body text-muted">{children}</div>;
}

/* ── Table ────────────────────────────────────────────────────────────── */

export function Table({ children }: { children: ReactNode }) {
  return (
    <div className="-mx-5 overflow-x-auto px-5">
      <table className="w-full border-collapse">{children}</table>
    </div>
  );
}

export function Th({ children, className = "" }: { children?: ReactNode; className?: string }) {
  return (
    <th
      className={`border-b border-border px-4 py-2.5 text-left text-micro font-semibold uppercase tracking-wider text-faint ${className}`}
    >
      {children}
    </th>
  );
}

export function Td({ children, className = "" }: { children?: ReactNode; className?: string }) {
  return <td className={`border-b border-border/70 px-4 py-3 text-body ${className}`}>{children}</td>;
}

/* ── Button ───────────────────────────────────────────────────────────── */

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md";

const buttonBase =
  "inline-flex select-none items-center justify-center gap-2 rounded-md border font-medium tracking-tight transition active:scale-[0.98] disabled:pointer-events-none disabled:opacity-45";

const buttonVariants: Record<ButtonVariant, string> = {
  primary: "border-transparent bg-accent text-accent-fg hover:bg-accent-strong shadow-xs",
  secondary: "border-border bg-surface text-text hover:bg-surface-2 shadow-xs",
  ghost: "border-transparent bg-transparent text-muted hover:bg-surface-2 hover:text-text",
  danger: "border-transparent bg-critical text-white hover:brightness-95 shadow-xs",
};

const buttonSizes: Record<ButtonSize, string> = {
  sm: "h-8 px-3 text-caption",
  md: "h-10 px-4 text-body",
};

export function Button({
  variant = "secondary",
  size = "md",
  loading = false,
  children,
  className = "",
  disabled,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
}) {
  return (
    <button
      {...props}
      disabled={disabled || loading}
      className={`${buttonBase} ${buttonSizes[size]} ${buttonVariants[variant]} ${className}`}
    >
      {loading && (
        <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
      )}
      {children}
    </button>
  );
}

/* ── Form controls ────────────────────────────────────────────────────── */

export function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: ReactNode;
  children: ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-caption font-medium text-muted">{label}</span>
      {children}
      {hint && <span className="mt-1.5 block text-caption text-faint">{hint}</span>}
    </label>
  );
}

const controlBase =
  "w-full rounded-md border border-border bg-surface-2 px-3.5 text-body text-text outline-none transition placeholder:text-faint hover:border-border-strong focus:border-accent";

export function Input({ className = "", ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={`h-10 ${controlBase} ${className}`} />;
}

export function Select({ className = "", children, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select {...props} className={`h-10 ${controlBase} ${className}`}>
      {children}
    </select>
  );
}

/* ── Modal ────────────────────────────────────────────────────────────── */

export function Modal({
  title,
  onClose,
  children,
  footer,
}: {
  title: ReactNode;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/30 p-0 backdrop-blur-sm sm:items-center sm:p-4"
      onClick={onClose}
    >
      <div
        className="animate-sheet-in w-full max-w-lg rounded-t-2xl border border-border bg-elevated shadow-lg sm:rounded-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <header className="flex items-center justify-between gap-3 border-b border-border px-5 py-4">
          <h2 className="text-[0.95rem] font-semibold tracking-tight text-text">{title}</h2>
          <button
            onClick={onClose}
            className="grid h-7 w-7 place-items-center rounded-md text-muted hover:bg-surface-2 hover:text-text"
            aria-label="Close"
          >
            ✕
          </button>
        </header>
        <div className="px-5 py-4">{children}</div>
        {footer && (
          <footer className="flex justify-end gap-2 border-t border-border px-5 py-3.5">{footer}</footer>
        )}
      </div>
    </div>
  );
}
