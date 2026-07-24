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
| **L2 — Verification** | 13 real-token probes vs. a live service | covered by unit tests (`internal/probe`), not the offline corpus |
| **L3 — Diff** | classifying a contract change (widened/narrowed) with no false positives | ✅ the headline number |
| **L4 — Attribution** | binding a change to a commit | covered by `internal/attribution` tests |

## Corpus composition (PRD §13.2)

Two complementary sources, combined into one scorecard (`L3-all`):

1. **Generated corpus** (`bench/mutations`): for every row of the classification
   table (PRD §9.2) it emits a true-positive change, and for every
   contract-neutral field it emits a no-op. At the default 50 rounds that is
   **400 true positives + 400 true negatives** — the ~50/50 split OWASP
   recommends, so the false-positive rate is meaningful.

2. **File-based scenarios** (`bench/corpus/scenarios/*/`): each has a `before/`
   and `after/` directory of real Istio/Envoy/K8s manifests plus a
   `scenario.yaml` ground truth. These run the **actual discovery pipeline**
   (istio + k8s + envoy adapters) into `diff.Compute`, so they exercise L1 as
   well as L3. The seed set:

   | ID | Kind | Tests |
   |---|---|---|
   | `0042-audience-widened` | TP | new audience → widened/medium |
   | `0043-issuer-migration` | TP | second issuer added → widened/high |
   | `0044-cache-ttl-raised` | TP | Envoy `cache_duration` 5m→1h → narrowed/medium |
   | `0201-dependency-bump-noop` | TN | only an owner label changes → silent |
   | `0202-noisy-redeploy` | TN | 6 things churn (list reorder, replicas, labels, image, env, comments) → silent |

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

## Honest limitations

- **The generated corpus is deterministic.** Ground truth is derived from the
  same field semantics the diff uses, so a passing score there proves
  *consistency and regression-safety*, not that the classification rules are the
  "right" ones — that judgement lives in the classification table itself
  (PRD §9.2) and its unit tests.
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
