import type { ReactNode } from "react";
import { MarketingShell, PageHero } from "../components/MarketingChrome";
import { Reveal } from "../components/Reveal";

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
    body: "Bugs, false positives, or a detection we're missing.",
    action: "Open an issue",
    href: `${GITHUB}/issues/new`,
    icon: IssueIcon,
  },
  {
    title: "Start a discussion",
    body: "Questions, ideas, or talk through your auth setup.",
    action: "Start a discussion",
    href: `${GITHUB}/discussions`,
    icon: ChatIcon,
  },
  {
    title: "Report a vulnerability",
    body: "Private disclosure for anything exploitable.",
    action: "Security policy",
    href: `${GITHUB}/blob/main/SECURITY.md`,
    icon: ShieldIcon,
  },
  {
    title: "Email us",
    body: "Partnerships, the cloud waitlist, or anything private.",
    action: "hello@keyway.dev",
    href: "mailto:hello@keyway.dev?subject=Keyway",
    icon: MailIcon,
  },
];

export default function Contact() {
  return (
    <MarketingShell>
      <PageHero
        eyebrow="Contact"
        title="Get in touch."
        line="Built in the open. Every channel below reaches a real maintainer — no sales team, no form to nowhere."
      />
      <section className="px-5 py-16 sm:px-8 sm:py-20">
        <div className="mx-auto grid max-w-3xl grid-cols-1 gap-4 sm:grid-cols-2">
          {channels.map((c, i) => (
            <Reveal key={c.title} delay={(i % 2) * 80}>
              <a
                href={c.href}
                className="lift group flex h-full flex-col rounded-2xl border border-border bg-surface p-6 shadow-xs"
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
            </Reveal>
          ))}
        </div>
      </section>
    </MarketingShell>
  );
}
