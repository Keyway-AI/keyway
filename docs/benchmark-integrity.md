# Benchmark integrity: is the 100% real, or overfit?

A headline of **100% on a corpus we wrote ourselves is, correctly, a red flag.**
It proves the corpus is internally consistent — not that the detector
generalises. This page is the adversarial answer to "prove you're not just
grading your own homework." It reports three things that a rigged 100% could not
survive: **mutation testing**, a **held-out adversarial corpus** (where we score
**0.5 Youden, not 1.0**), and an **honest industry comparison**.

_Last updated: 2026-07-25._

---

## 1. Why "overfitting" needs a careful definition here

Keyway's classifier is **not a machine-learning model** — it is a fixed rule
table (PRD §9.2: `audiences +→ widened/medium`, `required_claim −→
widened/critical`, …). There are no learned weights to overfit. So the ML sense
of overfitting does not apply directly.

The *analogous* risk is real and worth attacking: **a corpus that only exercises
the paths the rules already handle** proves nothing. A test that says "the code
does what the code does" is circular. The question is therefore:

> Does the corpus actually pin down the detector's behaviour, such that a **bug
> introduced into the detector would be caught**?

That is exactly what mutation testing measures — and it cannot be gamed by adding
more agreeable corpus rows.

## 2. Mutation testing (the code-side proof)

We use [`gremlins`](https://github.com/go-gremlins/gremlins) to inject faults
into the classifier (`internal/diff`) — flip every `<` to `<=`, negate every
`&&`/`||`/`!=`, etc. — and re-run the **entire test suite (including the corpus)**
against each mutant. A mutant that "lives" is a detector bug the tests failed to
catch.

Reproduce: **`make mutation`**.

| Metric | Result |
|---|---|
| Mutation points (mutants generated) | 24 |
| **Mutator coverage** (mutants the tests actually exercise) | **100%** |
| Behaviour-changing mutants **killed** | **all of them** |
| Survivors | 2, both **provably equivalent** (see below) |

**The two survivors are equivalent mutants, not test gaps.** Both are the
boundary flip `<` → `<=` at `diff.go:73` (clock skew) and `diff.go:95` (cache
TTL). In both cases the line immediately above guards with `!=`
(`if from.X != to.X { … if to.X < from.X … }`), so the two values are **never
equal** when the `<` is evaluated — `<` and `<=` produce identical behaviour.
Equivalent mutants are unkillable by definition and are a well-known artifact of
mutation testing, not a hole in the corpus.

**What this establishes:** every fault we can inject into the classifier that
*changes its output* is caught by the test suite. The corpus is doing real work,
not rubber-stamping. (Earlier, before this pass, coverage was 74% and four
mutants lived — we found and closed those gaps with targeted unit tests. The
process itself is the evidence: the tooling finds holes, and we close them.)

## 3. Held-out adversarial corpus (the data-side proof)

Mutation testing proves the tests pin the code. It does **not** prove the rules
are *right* on inputs we never thought to generate. For that we keep a separate,
deliberately nasty corpus in
[`bench/corpus/adversarial/`](../bench/corpus/adversarial), scored **separately
and NOT gated** — its job is to expose limitations, not to pass. It includes
cases we expected to fail.

Run: `make bench` prints an `L3-adversarial` line alongside the main score.

**Result: TPR = 1.00, FPR = 0.50, Youden = 0.50** (TP 4, FP 2, TN 2, FN 0).

**Not 100% — by design, and honestly reported.** Breakdown:

| Case | What's hard | Outcome |
|---|---|---|
| `adv-01 audience-swap` | one audience removed **and** one added at once | ✅ correct (widened) |
| `adv-02 reorder-hides-add` | real add buried in a list reorder | ✅ correct (widened) |
| `adv-04 claim-value-changed` | required claim's **value** changes, name stays | ✅ correct (silent) |
| `adv-05 duplicate-audience` | same audience listed twice (dedup) | ✅ correct (silent) |
| `adv-07 mixed-multi-consumer` | one service churns noise, another really widens | ✅ correct (flags only the real one) |
| `adv-08 double-narrow` | two narrowings on one consumer at once | ✅ correct (narrowed) |
| `adv-03 issuer-trailing-slash` | issuer gains a trailing `/` | ❌ **flagged** (we mark this a false positive) |
| `adv-06 issuer-whitespace` | issuer gains a trailing space | ❌ **flagged** (false positive) |

The two failures are a **real, documented limitation**: Keyway compares issuer
strings **verbatim** and has no normalization pass, so `…/main` vs `…/main/`
reads as an issuer change (KI-26). We count these **against ourselves** even
though the "right" answer is genuinely debatable — under strict OIDC the `iss`
claim must byte-match, so a validator that changed its configured issuer to
`…/main/` really would start rejecting `…/main` tokens, and flagging is
defensible. We take the self-critical scoring (FP) rather than claim the
ambiguity in our favour.

**This is the number that matters for a skeptic:** on hard, out-of-distribution
cases, Keyway is **not perfect (0.5 Youden)**, and the specific failures are
named and tracked, not hidden.

## 4. Why the *main* corpus can legitimately hit ~1.0 (and a scanner can't)

The task matters. Keyway diffs **structured configuration** against a
**deterministic rule table**. That is a fundamentally more tractable problem than
a SAST tool inferring vulnerabilities from arbitrary source code — and it is why
a high score on the main corpus is plausible rather than suspicious.

For calibration (published third-party **OWASP Benchmark** results — *not* re-run
by us, and a *different task*):

| Tool | Task | Reported accuracy |
|---|---|---|
| Snyk Code | SAST (infer vulns from code) | 97.18% TPR |
| Semgrep | SAST | 87.06% TPR |
| SonarQube | SAST | 50.36% TPR |
| Kiuwan | SAST | 100% TPR **@ 16% FPR** |
| SAST category | — | historical **Youden ceiling ~0.39** |
| **Keyway (main corpus)** | structured config diff | 1.00 Youden |
| **Keyway (adversarial)** | hardest config edge cases | **0.50 Youden** |

The honest reading is **not** "Keyway beats Snyk" — they solve different
problems. It is: *inference from code has a well-documented accuracy ceiling;
checking structured config against explicit rules does not infer, so it can be
near-exact on the cases the rules cover — and the adversarial set shows where the
rules stop covering.*

## 5. Where the real risk actually lives

Because the classifier is deterministic, classifier accuracy is **not** where
Keyway is most likely to disappoint. The honest risk surface is:

- **Discovery recall** — do we find *every* consumer? Manifests with unusual
  shapes, Helm-templated values, or non-Istio/Envoy meshes may be missed. This is
  the true generalisation question and is bounded by documented blind spots
  (KI-19 algorithm restrictions, KI-21 low-confidence env-var hints).
- **Config diversity** — the corpus is realistic but finite (KI-18). The
  adversarial set is how we keep pushing on this; contributions of new hard cases
  are the most valuable thing you can add.
- **Issuer/string normalization** — the adversarial failures above (KI-26).

None of these are hidden by the 100%; they are why this page exists.

## 6. Bottom line

- The 100% on the main corpus is **real but narrow**: it means the deterministic
  rules are self-consistent and regression-locked.
- **Mutation testing** (100% mutator coverage, every behaviour-changing fault
  killed) shows the corpus genuinely tests the detector — the anti-overfitting
  proof that can't be gamed by adding rows.
- The **adversarial corpus scores 0.5 Youden**, names its failures, and counts
  ambiguity against itself — the opposite of a rigged benchmark.
- Reproduce all of it: `make mutation`, `make bench`, `make validate`.
