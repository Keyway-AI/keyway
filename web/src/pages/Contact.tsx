import type { ReactNode } from "react";
import { MarketingShell } from "../components/MarketingChrome";
import { LogoBadge } from "../components/Logo";

const GITHUB = "https://github.com/Keyway-AI/keyway";

type Channel = {
  title: string;
  body: string;
  action: string;
  href: string;
  icon: () => ReactNode;
};

function IssueIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8v4M12 16h.01" />
    </svg>
  );
}
function ChatIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H8l-4 4V5a2 2 0 0 1 2-2h13a2 2 0 0 1 2 2Z" />
    </svg>
  );
}
function ShieldIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3l7 3v5c0 4.5-3 8-7 10-4-2-7-5.5-7-10V6Z" />
      <path d="m9 12 2 2 4-4" />
    </svg>
  );
}
function MailIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="m3 7 9 6 9-6" />
    </svg>
  );
}

const channels: Channel[] = [
  {
    title: "Open an issue",
    body: "Found a bug, a false positive, or a missing detection? File it on GitHub — that's where the work happens in the open.",
    action: "Open an issue",
    href: `${GITHUB}/issues/new`,
    icon: IssueIcon,
  },
  {
    title: "Start a discussion",
    body: "Questions, feature ideas, or want to talk through your auth setup? Discussions are the place for the long-form stuff.",
    action: "Start a discussion",
    href: `${GITHUB}/discussions`,
    icon: ChatIcon,
  },
  {
    title: "Report a vulnerability",
    body: "Security issues get a private disclosure path. Please don't file a public issue for anything exploitable — follow the security policy.",
    action: "Security policy",
    href: `${GITHUB}/blob/main/SECURITY.md`,
    icon: ShieldIcon,
  },
  {
    title: "Email us",
    body: "Prefer email — for partnerships, the managed cloud waitlist, or anything not suited to a public thread? Reach the maintainers directly.",
    action: "hello@keyway.dev",
    href: "mailto:hello@keyway.dev?subject=Keyway",
    icon: MailIcon,
  },
];

export default function Contact() {
  return (
    <MarketingShell>
      <section className="mx-auto max-w-5xl px-5 py-16 sm:px-8 sm:py-24">
        <div className="mx-auto max-w-2xl text-center">
          <span className="mx-auto grid h-12 w-12 place-items-center">
            <LogoBadge size={44} />
          </span>
          <span className="eyebrow mt-4 block">Contact</span>
          <h1 className="mt-2 text-h1 font-semibold tracking-tight sm:text-[2.5rem]">Get in touch.</h1>
          <p className="mx-auto mt-3 max-w-lg text-body-lg text-muted">
            Keyway is built in the open. The fastest way to reach us — and to shape where it goes — is on GitHub. For anything private, email works too.
          </p>
        </div>

        <div className="mx-auto mt-12 grid max-w-3xl grid-cols-1 gap-4 sm:grid-cols-2">
          {channels.map((c) => (
            <a
              key={c.title}
              href={c.href}
              className="group flex flex-col rounded-2xl border border-border bg-surface p-6 shadow-xs transition hover:border-border-strong hover:shadow-sm"
            >
              <span className="grid h-11 w-11 place-items-center rounded-xl bg-accent-soft text-accent">
                <c.icon />
              </span>
              <h2 className="mt-4 text-[1.05rem] font-semibold tracking-tight">{c.title}</h2>
              <p className="mt-2 flex-1 text-body text-muted">{c.body}</p>
              <span className="mt-4 inline-flex items-center gap-1.5 text-body font-medium text-accent">
                {c.action}
                <span className="transition group-hover:translate-x-0.5">→</span>
              </span>
            </a>
          ))}
        </div>

        <p className="mx-auto mt-10 max-w-lg text-center text-caption text-faint">
          Keyway is open source under the MIT license. There is no sales team and no contact form to nowhere — every channel above reaches a real maintainer.
        </p>
      </section>
    </MarketingShell>
  );
}
