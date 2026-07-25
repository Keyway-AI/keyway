# Benchmark methodology

This document is the technical companion to [BENCHMARK.md](../BENCHMARK.md). It
describes exactly what the harness measures, how the corpus is built, and the
honest limits of the numbers. It follows PRD §13.

## Layers measured

Keyway is four measurable layers; the harness scores the two that need no live
cluster or staging endpoints:

| Layer | What | Measured here? |
|---|---|---|
| **L1 — Derivation** | building the consumer inventory from manifests | ✅ via file-based scenarios (real discovery) |
| **L2 — Verification** | 13 real-token probes vs. a live service | ✅ `bench/l2/` docker rig (`make bench-l2`) — probes 8 real containerized services, **100% correct verdicts** |
| **L3 — Diff** | classifying a contract change (widened/narrowed) with no false positives | ✅ the headline number |
| **L4 — Attribution** | binding a change to a commit | covered by `internal/attribution` tests |

## Corpus composition (PRD §13.2)

Three complementary sources, reported as three scorecards (`L3` for the
generated struct corpus, `L3-realistic` for the rendered-YAML corpus, and
`L3-all` for the generated + hand-authored file scenarios):

1. **Generated corpus** (`bench/mutations`): for every row of the classification
   table (PRD §9.2) it emits a true-positive change, and for every
   contract-neutral field it emits a no-op. At the default 50 rounds that is
   **400 true positives + 400 true negatives** — the ~50/50 split OWASP
   recommends, so the false-positive rate is meaningful. It mutates model structs
   directly, so it proves the classifier is self-consistent (see the honesty note
   for what that does and doesn't establish).

2. **Realistic corpus** (`bench/harness/realistic.go`, `--realistic N`, default
   **400**): deterministically renders real Istio / Envoy / Kubernetes YAML
   across parameterized names, namespaces, issuers, audiences, claims, and cache
   TTLs, applies a catalog of realistic mutations (12 true-positive risk classes,
   8 no-op churn patterns), and runs each pair through the **actual discovery
   pipeline** into `diff.Compute`. Because ground truth comes from the manifest
   mutation — not the diff's own field semantics — a perfect score here is
   end-to-end evidence that derive→classify is correct at breadth, not just
   self-consistent. Generation is index-driven (no RNG) so CI is reproducible.
   Current split: **240 true-positive / 160 true-negative**.

3. **File-based scenarios** (`bench/corpus/scenarios/*/`): each has a `before/`
   and `after/` directory of real Istio/Envoy/K8s manifests plus a
   `scenario.yaml` ground truth, hand-authored to pin specific, named cases.
   These also run the **actual discovery pipeline** (istio + k8s + envoy
   adapters) into `diff.Compute`, so they exercise L1 as well as L3. The current
   set (**26 scenarios**, 15 true-positive / 11 true-negative, spanning the
   Istio, Envoy, and K8s discovery sources):

   | ID | Kind | Tests |
   |---|---|---|
   | `0042-audience-widened` | TP | Istio: new audience → widened/medium |
   | `0043-issuer-migration` | TP | Istio: second issuer added → widened/high |
   | `0044-cache-ttl-raised` | TP | Envoy: `cache_duration` 5m→1h → narrowed/medium |
   | `0045-audience-removed` | TP | Istio: audience removed → narrowed/low |
   | `0046-issuer-removed` | TP | Istio: old issuer removed → narrowed/low |
   | `0047-cache-ttl-lowered` | TP | Envoy: `cache_duration` 1h→5m → narrowed/low |
   | `0048-required-claim-dropped` | TP | Istio AuthPolicy: required claim removed → widened/critical |
   | `0049-issuer-added` | TP | Istio: second trusted issuer added → widened/high |
   | `0050-required-claim-added` | TP | Istio AuthPolicy: new required claim → narrowed/low |
   | `0051-consumer-added` | TP | Istio: new service onboarded → neutral/low |
   | `0052-consumer-removed` | TP | Istio: service decommissioned → neutral/low |
   | `0053-envoy-audience-widened` | TP | Envoy: provider gains an audience → widened/medium |
   | `0054-envoy-issuer-migration` | TP | Envoy: issuer cut over → widened/high |
   | `0055-audience-narrowed-multi` | TP | Istio: one of two consumers narrows; neighbour stays silent |
   | `0056-widen-and-claim` | TP | Istio: audience widened + required claim dropped on one consumer |
   | `0201-dependency-bump-noop` | TN | only an owner label changes → silent |
   | `0202-noisy-redeploy` | TN | Istio+K8s: 6 things churn (reorder, replicas, labels, image, env, comments) → silent |
   | `0203-k8s-noisy-redeploy` | TN | K8s: scale + relabel + image bump, SA audience unchanged → silent |
   | `0204-comment-only-noop` | TN | Istio: comments/formatting only → silent |
   | `0205-audience-reordered` | TN | Istio: audience list reordered, same set → silent |
   | `0206-jwtrules-reordered` | TN | Istio: two jwtRules reordered → silent |
   | `0207-annotation-added` | TN | Istio: GitOps annotations added → silent |
   | `0208-envoy-noisy` | TN | Envoy: unrelated fields churn, contract fields unchanged → silent |
   | `0209-unrelated-service-added` | TN | K8s: ConfigMap + metrics Service added (non-consumers) → silent |
   | `0210-authpolicy-values-reordered` | TN | Istio: claim values reordered, same set → silent |
   | `0211-whitespace-reformat` | TN | Istio: flow→block YAML reformat → silent |

## Scoring (PRD §13.3)

Each scenario runs through the real `diff.Compute`. A true positive is counted
only when the emitted event matches the expected **class** (and consumer, when
specified) — a detection with the wrong classification is **not** a hit. The
confusion matrix yields TPR, FPR, precision, recall, F1 and **Youden = TPR −
FPR** (the headline).

## CI gate (PRD §13.4)

`go run ./bench/harness --ci-gate` exits non-zero if any threshold is breached:

| Layer | Metric | Fail below |
|---|---|---|
| L1 | derivation recall | < 70% |
| L2 | probe accuracy | < 95% |
| L3 | diff FPR | > 5% |
| L3 | Youden | < 0.70 |
| L4 | attribution | < 60% |

This runs on every push/PR (`.github/workflows/ci.yml`), so the published
accuracy cannot silently regress.

## Is the 100% overfit? (integrity)

A 100% on a self-authored corpus proves consistency, not generalisation. Two
checks that a rigged number could not survive live in
[benchmark-integrity.md](benchmark-integrity.md): **mutation testing**
(`make mutation` — inject faults into the classifier, 100% mutator coverage,
every behaviour-changing fault killed) and a **held-out adversarial corpus**
(`bench/corpus/adversarial/`, scored as `L3-adversarial`, **0.5 Youden** — not
perfect, failures named). Read that page before trusting the headline.

## Honest limitations

- **The generated corpus is deterministic.** Ground truth is derived from the
  same field semantics the diff uses, so a passing score there proves
  *consistency and regression-safety*, not that the classification rules are the
  "right" ones — that judgement lives in the classification table itself
  (PRD §9.2) and its unit tests. Mutation testing (above) is how we confirm the
  corpus actually exercises those rules.
- **The realistic signal is in the file-based scenarios**, especially the no-op
  ones: those are where a naive differ would generate false positives, and where
  the "zero-noise" claim is actually earned. Growing that set (toward the PRD's
  400-scenario target) is the main way to harden the number further.
- **L2 (probing) and L4 (attribution) are not in the offline corpus** because
  they need a live service and a git history respectively; they are covered by
  package tests instead. A full docker-compose corpus (PRD §13.1) that also
  scores L2 is future work.
- **The comparison figures are not a head-to-head.** Static analysers and Keyway
  do different jobs; their OWASP numbers are cited for calibration only.
