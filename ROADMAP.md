# Keyway roadmap

This is a direction, not a promise — priorities shift with real-world use and
contributions. Dates are deliberately omitted. Coverage numbers track the
[threat taxonomy](docs/threat-coverage.md); every gap below is a cited threat, not
a vague aspiration.

## Shipped (v0.1.0)

- Automatic JWT consumer discovery (Istio / Envoy / Kubernetes / OIDC), contract
  versioning + diff, and the 13-probe verification engine.
- Blast radius + measured grace period; canary key flow.
- Cited threat coverage report; generative attack harness.
- AI-agent auth: static token analyzer (MCP audience, delegation, scope, expiry).
- Web dashboard, CLI, single-binary/container deploy, zero-config demo.

## Next — raise detection coverage

The honest denominator is the [threat taxonomy](docs/threat-coverage.md) (currently
**27/50**). Closing named gaps is the top priority. High-value targets:

- **JWT header attacks** — `jku` / `x5u` / `x5c` trust-source checks, `kid`
  injection (HDR-01/02/04/06), algorithm downgrade (ALG-02).
- **Agent-auth flow attacks** — confused deputy via static client ID (CD-01),
  dynamic-client-registration abuse (CD-02), `may_act` enforcement on token
  exchange (DEL-03), impersonation-vs-delegation (DEL-04), agent reusing the
  human's root credential (SCOPE-03), tool poisoning / rug-pull (AGT-02).

## Next — discovery & modeling

- Live in-cluster **Envoy / Kubernetes** reads (Istio already supported).
- First-class **namespace-wide RequestAuthentication** modeling (KI-33).
- Broader OIDC/registry coverage and cross-source identity merge refinements.

## Later

- **Degraded mode for SaaS IdPs** (Auth0 / Okta / Entra) where Keyway does not
  control the signing key — shadow issuer + library-defaults inference (KI-08).
- **Managed cloud**: hosted, always-on scanning with historical coverage trends
  and Slack / PagerDuty alerting.

## Non-goals

Keyway verifies that auth is enforced correctly; it is **not** an issuer, a
gateway, or an enforcement point. It will never mutate your configuration, block a
deploy, or require you to author a model file. Features that need any of those are
out of scope by definition.

---

Want to move something up the list? Open or upvote an
[issue](https://github.com/Keyway-AI/keyway/issues), or start a
[discussion](https://github.com/Keyway-AI/keyway/discussions). Contributions toward
any "Next" item are especially welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
