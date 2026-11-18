# Keyway launch kit — domain, deploy, and directories

Everything needed to point **keywayhq.com** at the site and popularize the open-source
launch. The brand assets live in `web/public/` (`logo.svg`, `logo-512.png`,
`apple-touch-icon.png`, `og.png`, `favicon.svg`).

---

## 1. Link keywayhq.com (Namecheap → Vercel)

The site deploys on Vercel (`web/vercel.json`, Vite). To attach the domain you bought on
Namecheap:

1. ✅ **Done:** `keywayhq.com` is added to the Vercel **keyway** project, connected to
   **Production**, as the apex (primary) domain — matching the site's
   `<link rel="canonical" href="https://keywayhq.com/">`. (It shows "Invalid Configuration"
   until the DNS below is live.)
2. **In Namecheap → Domain List → keywayhq.com → Manage → Advanced DNS**, keep **Namecheap
   BasicDNS** and add the single record Vercel asks for:
   | Type | Host | Value | TTL |
   |---|---|---|---|
   | A Record | `@` | `216.198.79.1` | Automatic |
   (Use the exact value Vercel's **View DNS configuration** shows — they occasionally change
   the apex IP.)
3. **Remove Namecheap's default records** that shadow it: the `CNAME @ → parkingpage.namecheap.com`
   and the `URL Redirect` on `@`/`www`. Leave only the A record above.
4. (Optional) To also serve `www.keywayhq.com`: in Vercel add `www.keywayhq.com` as a
   redirect to the apex, then add a Namecheap `CNAME www → cname.vercel-dns.com.`.
5. Wait for propagation (minutes–1h). Back in Vercel's Domains panel click **Refresh**;
   it should flip to **Valid Configuration** and auto-issue TLS, so `https://keywayhq.com`
   loads with a padlock.

> Alternative: set Namecheap to **Custom DNS → Vercel nameservers** if you'd rather Vercel
> manage all DNS. Not required for just the website.

**After it resolves, verify:**
- `https://keywayhq.com` loads the landing page over HTTPS.
- Social card: paste the URL into <https://www.opengraph.xyz> and the Twitter/X card
  validator — confirm `og.png` (the emerald "Verify your token contracts" card) renders.
- Favicon + apple-touch icon show in a tab and when saved to a phone home screen.

---

## 2. Pre-launch checklist (repo + release)

- [ ] **GitHub repo public** (`Keyway-AI/keyway`), description set to the one-liner below.
- [ ] **Repo social preview**: Settings → General → Social preview → upload `web/public/og.png`.
- [ ] **Topics/tags**: `jwt`, `oauth2`, `oidc`, `jwks`, `authorization`, `mcp`, `ai-agents`,
      `security`, `golang`, `kubernetes`, `istio`, `envoy`, `appsec`, `sbom`.
- [ ] **Website field** on the repo → `https://keywayhq.com`.
- [ ] Tag a **release** (`v0.1.0`) so there's something to link; confirm `install.sh` works
      from a clean machine (`curl -fsSL https://raw.githubusercontent.com/Keyway-AI/keyway/main/install.sh | sh`).
- [ ] Pin the repo on the `Keyway-AI` org and your profile.
- [ ] `README` website link + badge (done), license consistent as **Apache 2.0** (fixed).

---

## 3. Directory & community submissions (popularize)

Work top-down; the first group is highest-leverage for this audience.

**Developer front pages (time these together for a launch day):**
- [ ] **Hacker News — Show HN**: *"Show HN: Keyway – verify your JWT & AI-agent auth contracts before you ship"*. Post the keywayhq.com link; first comment = the backstory + the measurement finding.
- [ ] **Lobsters** (`show`, tags: `security`, `go`).
- [ ] **Reddit**: r/netsec, r/golang, r/kubernetes, r/devops, r/selfhosted (read each sub's self-promo rules first).
- [ ] **Product Hunt** (use `logo-512.png` as the thumbnail, `og.png` as the gallery image).

**Awesome lists (PR to add Keyway):**
- [ ] awesome-oauth, awesome-jwt
- [ ] awesome-mcp-servers / awesome-mcp (the agent-authorization angle is the hook)
- [ ] awesome-security, awesome-appsec, awesome-go
- [ ] awesome-kubernetes-security / awesome-istio (mesh discovery angle)

**Registries & landscapes:**
- [ ] **MCP registry** (modelcontextprotocol registry) — list `keyway agent inspect` as an MCP-auth tool.
- [ ] **OpenSSF landscape** (already have a Scorecard badge — get listed).
- [ ] **pkg.go.dev** (already indexed) — ensure the module doc/readme render.
- [ ] AlternativeTo, SaaSHub, LibHunt, StackShare.

**Written + newsletters:**
- [ ] A **dev.to / Hashnode** post: the 530-service measurement + the MCP-verifier finding (FastMCP/SDK accept passthrough + non-expiry) — links back to keywayhq.com.
- [ ] Pitch security newsletters: **tl;dr sec**, **Return on Security**, **Detection Engineering Weekly**, **Golang Weekly**.
- [ ] Post in relevant Slacks/Discords: CNCF, MCP community, OWASP.

---

## 4. Ready-to-paste launch copy

**One-liner (repo description, directories):**
> Keyway — discover, version, and adversarially test your JWT & AI-agent (OAuth/MCP) authorization contracts before you rotate a key or ship an agent. Open source.

**Short (Product Hunt tagline, ≤60 chars):**
> Verify your token contracts before you ship.

**Tweet / X:**
> Launching Keyway 🔑 — open-source tooling for the authorization layer.
> It discovers which services validate your JWTs, versions the contract, and fires adversarial tokens (JWT + MCP agent tokens) at them to find what they *wrongly accept* — token passthrough, no-expiry agent creds, unbound audience.
> https://keywayhq.com

**Show HN first comment (skeleton):**
> I kept seeing the same bug: a service *says* it validates a token, but its config (or its JWT library's defaults) quietly accepts tokens it shouldn't. We measured it across 530 services and then fired adversarial tokens at 9 real verifiers + 2 MCP servers — most accept a non-expiring token, and the reference MCP SDK's default config accepts token passthrough. Keyway packages the discovery + the adversarial harness so you can check your own stack. Apache-2.0, Go + a React UI. Happy to answer questions.

---

*Assets: `web/public/{logo.svg, logo-512.png, apple-touch-icon.png, og.png, favicon.svg}`.
Regenerate the PNGs from SVG with `rsvg-convert` (see git history for the source SVGs).*
