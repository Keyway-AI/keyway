# Secure Software Development Lifecycle (SSDLC)

Keyway is a security tool, so it holds itself to a documented secure-development
process. This page describes the controls — automated and manual — that gate and
inform every change. It maps loosely to the [NIST Secure Software Development
Framework (SSDF, SP 800-218)](https://csrc.nist.gov/pubs/sp/800/218/final) and
[OWASP SAMM](https://owaspsamm.org/).

All automated controls run in CI on every pull request and push to `main`, plus a
weekly schedule. They are defined in [`.github/workflows/`](../../.github/workflows/).

## Automated controls

| Control | Tool (open source) | Workflow | Gating? |
|---|---|---|---|
| **Dependency vulnerabilities (Go)** | [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) | `security.yml` | ✅ fails build |
| **Dependency vulnerabilities (npm, shipped)** | `npm audit --omit=dev` | `security.yml` | ✅ at critical |
| **Automated dependency updates** | Dependabot | `dependabot.yml` | PRs |
| **Secret scanning** | [gitleaks](https://github.com/gitleaks/gitleaks) | `security.yml` | ✅ fails build |
| **SAST — semantic (Go + TS)** | [CodeQL](https://codeql.github.com/) (`security-and-quality`) | `codeql.yml` | report → Security tab |
| **SAST — Go rules** | [gosec](https://github.com/securego/gosec) | `security.yml` | report → Security tab |
| **Filesystem / IaC / secret scan** | [Trivy](https://trivy.dev/) | `security.yml` | report → Security tab |
| **DAST** | [OWASP ZAP](https://www.zaproxy.org/) baseline against the zero-config demo | `security.yml` | report (artifact) |
| **Supply-chain posture** | [OpenSSF Scorecard](https://securityscorecards.dev/) | `scorecard.yml` | report + badge |
| **Accuracy / detection regression** | Keyway's own benchmark + real-world corpus | `ci.yml` | ✅ fails build |
| **SBOM (SPDX)** | [Syft](https://github.com/anchore/syft) | `release.yml` | attached to each release |
| **Signed container images** | [cosign](https://github.com/sigstore/cosign) (keyless, OIDC) | `release.yml` | every image |

**Gating vs. report-only.** We fail the build on high-confidence signals —
known-exploitable dependency vulnerabilities and committed secrets. Heuristic
SAST/DAST findings (gosec, CodeQL, Trivy, ZAP) are surfaced in the **Security →
Code scanning** tab for triage rather than blocking, so a false positive can't
wedge the pipeline. (CodeQL, Scorecard and SARIF upload require a public repo /
GitHub Advanced Security, so those steps activate when the repo is public.)

Run the local subset any time:

```bash
make security   # SAST + secret scan
make sast       # govulncheck + gosec + npm audit
make dast       # ZAP baseline against `make demo`
make sbom       # syft SPDX SBOM
```

## Manual controls

- **Code review + ownership.** Every change is reviewed; security-sensitive
  surfaces (token minting, probing, key material, the API) have explicit owners in
  [CODEOWNERS](../../.github/CODEOWNERS). See [GOVERNANCE.md](../../GOVERNANCE.md).
- **Security invariants.** Five non-negotiable invariants (never mutate customer
  config, never block a deploy, never require a model file, never persist minted
  tokens, deny-by-default probing) gate every PR — see
  [CONTRIBUTING.md](../../CONTRIBUTING.md).
- **Threat modeling.** Detection is measured against a cited
  [threat taxonomy](../threat-coverage.md); every gap is named. New auth-handling
  code is expected to reason about the relevant threats.
- **Security audits.** See [docs/security-audit.md](../security-audit.md).

## Dependency & finding triage policy

- Dependabot proposes updates; shipped-dependency vulnerabilities are addressed
  promptly, dev-tooling advisories on a best-effort basis.
- A finding that does **not** apply to Keyway's usage may be suppressed **with a
  written justification** in [`.trivyignore`](../../.trivyignore) or
  [`.gitleaks.toml`](../../.gitleaks.toml). Current suppressions:
  - **react-router "RSC Mode CSRF" (GHSA-qwww-vcr4-c8h2)** — Keyway's web app is a
    static SPA (declarative routing, embedded static assets) and does not use
    React Router's RSC/server/data-action modes, the only configuration the
    advisory affects. Tracked for a future bump.
  - **gitleaks/gosec false positives** on test fixtures, benchmark rigs, and
    documented local-dev defaults (no real credentials).

## Vulnerability disclosure

Security vulnerabilities follow coordinated disclosure per
[SECURITY.md](../../SECURITY.md) — **never** a public issue.
