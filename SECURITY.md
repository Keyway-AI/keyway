# Security Policy

Keyway mints synthetic JWTs and interacts directly with authentication infrastructure. Please
treat it accordingly.

## Operational safety

- **Staging only by default.** The probe engine refuses to run against a host unless that host is
  in the configured allowlist, or the operator passes `--i-know-this-is-production`. Default deny.
- **Kill switch.** `keyway-runner` polls a `probe_enabled` flag before every consumer batch, so a
  run can be stopped without redeploying.
- **No token bodies at rest.** Keyway never persists minted tokens or logs their contents. Only
  the `jti` and the probe ID are recorded (PRD OPEN-4).
- **Least privilege.** The Keycloak/K8s credentials Keyway uses need read access for discovery and
  the ability to publish a *canary* key. They do **not** need permission to alter live signing
  keys, and Keyway will never attempt to.
- **Bounded probes.** Probe traffic is rate-limited per consumer and globally, aborts a consumer
  after consecutive 5xx responses, and targets a known-safe probe path.

## Supported versions

Until v1.0.0, only the latest `main` receives security fixes.

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Email **security@keyway.dev** with:

- a description of the issue and its impact,
- steps to reproduce (a minimal scenario in `bench/corpus/` format is ideal),
- affected version / commit.

You will receive an acknowledgement within 3 business days and a remediation timeline after
triage. We support coordinated disclosure and will credit reporters who wish to be named.
