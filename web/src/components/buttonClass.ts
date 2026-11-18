export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
export type ButtonSize = "sm" | "md" | "lg";

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
  lg: "h-11 px-5 text-body", // marketing CTAs
};

/** The single source of truth for button formatting — shared by the <Button>
 * component (app) and the marketing <Link>/<a> CTAs, so both read identically.
 * Lives outside ui.tsx so that component file keeps only component exports
 * (react-refresh/only-export-components). */
export function buttonClass(variant: ButtonVariant = "primary", size: ButtonSize = "md", className = "") {
  return `${buttonBase} ${buttonSizes[size]} ${buttonVariants[variant]} ${className}`.trim();
}
