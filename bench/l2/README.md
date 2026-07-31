# L2 — live-probe benchmark rig

The offline corpus (`bench/harness`) scores discovery (L1) and diff (L3). This
rig scores the one layer they can't: **L2 — probing real services with real
tokens over the network** (PRD §13, §6).

It stands up, in containers:

- **`issuer`** — a minimal OIDC issuer (Keyway's own local-key issuer) serving
  `/.well-known/openid-configuration`, `/jwks`, and a test-only `/mint`.
- **eight validator services** — each a real HTTP JWT-validating service, one
  correctly configured (`secure`) and one per documented weakness
  (`alg=none`, RS256→HS256 confusion, unverified signature, missing
  `aud`/`iss`/`exp`, identity-header trust).

Then `l2 score` runs Keyway's **actual probe engine** against every validator
(minting through the issuer's `/mint`) and checks Keyway returns the correct
verdict for each: the `secure` service must come back clean, and each vulnerable
service must be flagged by exactly the matching probe.

## Run it

```bash
make bench-l2
```

That builds the image, brings the compose topology up, waits for the issuer,
runs the scorer with the §13.4 gate (fail below 95% correct verdicts), and tears
everything down. Output is a table plus `bench/out/l2-scorecard.json`.

## Why the issuer stands in for Keycloak

The validators consume a standard OIDC JWKS, and Keyway's `issuer/keycloak`
adapter already does real OIDC discovery + JWKS fetch — so a real Keycloak would
plug in at the same endpoints. The lightweight issuer is used here because it is
deterministic and fast to boot; swapping in Keycloak is a compose change, not a
code change. See [docs/known-issues.md](../../docs/known-issues.md) KI-15.

**Safety note:** the issuer's `/mint` endpoint is a test rig convenience (it
signs arbitrary claims on request). It exists only in this benchmark image and
must never ship in a real deployment.
