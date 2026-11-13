import type { ReactNode } from "react";
import { MarketingShell } from "../components/MarketingChrome";

/**
 * Legal pages — Privacy and Terms. These are DRAFT templates: the structure and
 * the data-practice sections reflect what the site actually does today (GitHub
 * login, user-uploaded config, an essential session cookie, and no third-party
 * tracking analytics), but every bracketed [PLACEHOLDER] must be filled with your
 * real legal entity details and reviewed by counsel before you rely on them.
 */

function LegalShell({ title, updated, children }: { title: string; updated: string; children: ReactNode }) {
  return (
    <MarketingShell>
      <section className="mx-auto max-w-3xl px-5 py-16 sm:px-8 sm:py-24">
        <div className="rounded-lg border border-high/30 bg-high/5 px-4 py-3 text-caption text-muted">
          <strong className="text-text">Draft — not yet in effect.</strong> This template reflects the
          site&apos;s current data practices, but the bracketed fields and legal review are still pending.
          It is not legal advice.
        </div>
        <h1 className="display mt-8 text-[2rem] font-semibold tracking-tight sm:text-[2.4rem]">{title}</h1>
        <p className="mt-2 text-caption text-faint">Last updated: {updated}</p>
        <div className="legal-prose mt-8 space-y-6 text-body text-muted">{children}</div>
      </section>
    </MarketingShell>
  );
}

function H({ children }: { children: ReactNode }) {
  return <h2 className="text-h3 font-semibold tracking-tight text-text">{children}</h2>;
}

export function Privacy() {
  return (
    <LegalShell title="Privacy Policy" updated="[EFFECTIVE DATE]">
      <p>
        This policy explains what data <strong className="text-text">[COMPANY LEGAL NAME]</strong>{" "}
        (&ldquo;Keyway&rdquo;, &ldquo;we&rdquo;) collects when you use the Keyway website and hosted
        service at keywayhq.com, and your choices. The open-source Keyway CLI runs entirely in your own
        environment and sends us no data.
      </p>

      <div>
        <H>What we collect</H>
        <ul className="mt-2 list-disc space-y-1 pl-5">
          <li><strong className="text-text">Account:</strong> if you sign in, your GitHub profile basics (id, username, avatar, email) to create your account.</li>
          <li><strong className="text-text">Content you provide:</strong> the configuration files or projects you upload or connect, used only to produce your analysis.</li>
          <li><strong className="text-text">Essential cookies:</strong> a session cookie to keep you signed in, and a local theme preference stored in your browser. We do not use advertising or cross-site tracking cookies.</li>
        </ul>
      </div>

      <div>
        <H>What we don&apos;t do</H>
        <p className="mt-2">
          We do not currently run third-party analytics or tracking. We do not sell your data. If we add
          optional analytics in future, we will ask for your consent first and update this policy.
        </p>
      </div>

      <div>
        <H>How we use it</H>
        <p className="mt-2">Only to provide and secure the service — authenticate you, run the analysis you request, and keep your history. [Add any other specific uses.]</p>
      </div>

      <div>
        <H>Sharing &amp; sub-processors</H>
        <p className="mt-2">
          We share data only with the infrastructure providers needed to run the service, for example
          [HOSTING PROVIDER] and [DATABASE/EMAIL PROVIDERS], under their data-processing terms. [List your
          actual sub-processors.]
        </p>
      </div>

      <div>
        <H>Retention &amp; deletion</H>
        <p className="mt-2">
          You can delete a project (and its analyses) at any time from the app, and you can request deletion
          of your entire account and associated data by emailing{" "}
          <a href="mailto:[PRIVACY CONTACT EMAIL]" className="text-accent hover:underline">[PRIVACY CONTACT EMAIL]</a>.
          We delete or anonymize your data within [N] days of such a request, except where we must retain it by law.
        </p>
      </div>

      <div>
        <H>Your rights</H>
        <p className="mt-2">
          Depending on where you live (e.g. the EU/UK GDPR or California CCPA), you may have rights to access,
          correct, export, or delete your data, and to object to certain processing. To exercise them, contact{" "}
          <a href="mailto:[PRIVACY CONTACT EMAIL]" className="text-accent hover:underline">[PRIVACY CONTACT EMAIL]</a>.
        </p>
      </div>

      <div>
        <H>Contact</H>
        <p className="mt-2">
          [COMPANY LEGAL NAME], [REGISTERED ADDRESS]. Questions:{" "}
          <a href="mailto:[PRIVACY CONTACT EMAIL]" className="text-accent hover:underline">[PRIVACY CONTACT EMAIL]</a>.
        </p>
      </div>
    </LegalShell>
  );
}

export function Terms() {
  return (
    <LegalShell title="Terms of Service" updated="[EFFECTIVE DATE]">
      <p>
        These terms govern your use of the Keyway website and hosted service operated by{" "}
        <strong className="text-text">[COMPANY LEGAL NAME]</strong>. The Keyway software itself is open source
        under the Apache 2.0 license; your use of the source code is governed by that license, not these terms.
      </p>

      <div>
        <H>The service</H>
        <p className="mt-2">
          Keyway analyzes authorization configuration and tokens you provide. You are responsible for having the
          right to submit any configuration or token you upload, and for using the hosted service only against
          systems you are authorized to test.
        </p>
      </div>

      <div>
        <H>Accounts</H>
        <p className="mt-2">You must provide accurate account information and keep your credentials secure. You are responsible for activity under your account.</p>
      </div>

      <div>
        <H>Acceptable use</H>
        <p className="mt-2">Don&apos;t use the service to break the law, infringe others&apos; rights, probe systems you don&apos;t own or have permission to test, or disrupt the service.</p>
      </div>

      <div>
        <H>Plans &amp; payment</H>
        <p className="mt-2">[If/when paid plans exist: describe billing, renewal, and your refund policy here, or link a separate Refund Policy. Remove this section while the service is free.]</p>
      </div>

      <div>
        <H>Warranty &amp; liability</H>
        <p className="mt-2">
          The service is provided &ldquo;as is,&rdquo; without warranties. To the extent permitted by law,
          [COMPANY LEGAL NAME] is not liable for indirect or consequential damages. [Insert the liability cap and
          carve-outs your counsel advises.]
        </p>
      </div>

      <div>
        <H>Changes &amp; governing law</H>
        <p className="mt-2">
          We may update these terms and will post the new version here. These terms are governed by the laws of
          [JURISDICTION]. Contact:{" "}
          <a href="mailto:[LEGAL CONTACT EMAIL]" className="text-accent hover:underline">[LEGAL CONTACT EMAIL]</a>.
        </p>
      </div>
    </LegalShell>
  );
}
