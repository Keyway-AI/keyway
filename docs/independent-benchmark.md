# Independent benchmark: real configs we didn't write

The main corpus and the realistic generator are ours. This benchmark is the
opposite: **real JWT-authentication configs pulled from external open-source
projects that Keyway has never seen** — to measure how it actually generalises,
not how consistent it is with itself.

_Date: 2026-07-25._

## Sources (all public, unmodified except where noted)

The manifests live under [`bench/oss/manifests/`](../bench/oss/manifests). None
were authored for Keyway; each is copied verbatim from its source:

| File | Source | Real-world features it exercises |
|---|---|---|
| `01-istio-httpbin.yaml` | [Istio docs — Authorization / JWT task](https://istio.io/latest/docs/tasks/security/authorization/authz-jwt/) | RequestAuthentication **+** a separate AuthorizationPolicy requiring `request.auth.claims[groups]` |
| `02-istio-ingressgateway.yaml` | [Istio docs — Authentication Policy task](https://istio.io/latest/docs/tasks/security/authentication/authn-policy/) | selector uses the `istio:` label, **not** `app:` |
| `03-envoy-bookstore.yaml` | [Envoy docs — jwt_authn filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/jwt_authn_filter) | Envoy provider with **two audiences** + `cache_duration` (fragment wrapped in its standard `http_filters` envelope) |
| `04-istio-graphql-auth0.yaml` | [istio/istio#35941](https://github.com/istio/istio/issues/35941) | a real Auth0-backed service: `v1beta1`, `fromHeaders`, `outputPayloadToHeader` |
| `05-istio-delivery-platform.yaml` | [istio/istio#50246](https://github.com/istio/istio/issues/50246) | a real identity-provider config: **no selector** (namespace-wide), an `audience`, `forwardOriginalToken` |

## Method

```bash
go build -o /tmp/keyway ./cmd/keyway
/tmp/keyway discover --path bench/oss/manifests --output json     # L1
make bench-oss                                                    # L1 + a real-world L3 diff
```

## Result — discovery (L1)

Keyway found **5 / 5 consumers** and extracted **100% of the security-relevant
token contract correctly** — every issuer, every audience, the cross-resource
required claim, and the cache TTL. Nothing it emitted was wrong.

| Consumer (as Keyway named it) | Issuer(s) | Audiences | Req. claims | Cache TTL | Contract correct? |
|---|---|---|---|---|---|
| `k8s://local/foo/httpbin` | testing@secure.istio.io | — | **groups** ✅ (merged from the separate AuthorizationPolicy) | — | ✅ |
| `k8s://local/istio-system/jwt-example` | testing@secure.istio.io | — | — | — | ✅ (see naming caveat) |
| `k8s://local/graphql/graphql` | https://login.mycompany.io/ | — | — | — | ✅ (`v1beta1` parsed) |
| `k8s://local/istio-system/request-authentication` | https://app.platform-identity-provider…​ | **delivery-platform-testing** ✅ | — | — | ✅ (see naming caveat) |
| `route://envoy/provider_name1` | https://example.com | bookstore_android…, bookstore_web… ✅ | — | **300s** ✅ | ✅ |

**Contract-field accuracy: 5/5 (100%).** The standout is `httpbin`: the required
`groups` claim came from a *different* Kubernetes resource (an AuthorizationPolicy)
and Keyway correctly merged it onto the workload the RequestAuthentication defines.

## Result — diff (L3)

Using the real `delivery-platform` RequestAuthentication as the baseline and a
plausible change (onboarding a second audience, `delivery-platform-prod`), Keyway
classified it **widened / medium** — correct. Scored via `make bench-oss`, it is
the extra true positive: `L3-all` = TP 401 / FP 0 / FN 0.

## Honest gaps this surfaced

Real configs found real limitations — exactly the point of the exercise:

1. **Naming falls back to the policy name when the workload isn't obvious (2/5).**
   `02` selects by the `istio:` label (not `app:`), and `05` has *no* selector
   (a namespace-wide policy), so Keyway named both after the RequestAuthentication
   (`jwt-example`, `request-authentication`) instead of the workload. The token
   *contract* is still correct and the ID is stable; only the human-facing name is
   off. Tracked as **KI-33** (and it feeds the KI-28 merge-axis issue: a
   policy-named consumer won't merge with a `Deployment`-derived one).
2. **`jwksUri` is discarded on all five** (KI-32) — the rotation endpoint is
   parsed then dropped. No effect on the contract fields, but it's the single most
   useful thing we throw away.
3. **A namespace-wide (selector-less) policy is modelled as one policy-named
   consumer** rather than "all workloads in the namespace" (KI-30 for the
   AuthorizationPolicy case; the RequestAuthentication case is new under KI-33).

## Bottom line

On five real configs it had never seen — spanning Istio `v1`/`v1beta1`, Envoy,
Auth0, and a real identity provider, with `fromHeaders`/`forwardOriginalToken`/
cross-resource claims — Keyway extracted **100% of the security contract
correctly** and classified a real change correctly. The imperfections are in
**naming**, not in what tokens each service accepts. That is the honest measure:
the part that matters for "who can log into my service" generalises cleanly; the
cosmetic identity of two edge-case policies does not, and is now tracked.
