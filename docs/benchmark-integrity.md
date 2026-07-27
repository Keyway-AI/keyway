# Benchmark integrity: is the 100% real, or overfit?

A headline of **100% on a corpus we wrote ourselves is, correctly, a red flag.**
It proves the corpus is internally consistent — not that the detector
generalises. This page is the adversarial answer to "prove you're not just
grading your own homework." It reports three things that a rigged 100% could not
survive: **mutation testing**, a **held-out adversarial corpus** (where we score
**0.75 Youden, not 1.0**), and an **honest industry comparison**.

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

**Result: TPR = 1.00, FPR = 0.25, Youden = 0.75** (TP 4, FP 1, TN 3, FN 0).

**Not 100% — by design, and honestly reported.** Breakdown:

| Case | What's hard | Outcome |
|---|---|---|
| `adv-01 audience-swap` | one audience removed **and** one added at once | ✅ correct (widened) |
| `adv-02 reorder-hides-add` | real add buried in a list reorder | ✅ correct (widened) |
| `adv-04 claim-value-changed` | required claim's **value** changes, name stays | ✅ correct (silent) |
| `adv-05 duplicate-audience` | same audience listed twice (dedup) | ✅ correct (silent) |
| `adv-07 mixed-multi-consumer` | one service churns noise, another really widens | ✅ correct (flags only the real one) |
| `adv-08 double-narrow` | two narrowings on one consumer at once | ✅ correct (narrowed) |
| `adv-03 issuer-trailing-slash` | issuer gains a trailing `/` | ❌ **flagged** — a documented judgment call (real change under strict OIDC) |
| `adv-06 issuer-whitespace` | issuer gains a trailing space | ✅ correct (silent) — whitespace is now trimmed (KI-26) |

The one remaining miss is a **documented judgment call**: Keyway compares issuer
strings **verbatim**, so `…/main` vs `…/main/` reads as an issuer change (KI-26).
We count it **against ourselves** even though the "right" answer is genuinely
debatable — under strict OIDC the `iss` claim must byte-match, so a validator that
changed its configured issuer to `…/main/` really would start rejecting `…/main`
tokens, and flagging is defensible. (The sister case, a *whitespace* difference,
was a clear false positive and is now fixed — whitespace is trimmed, since a
token's `iss` can never carry it.)

**This is the number that matters for a skeptic:** on hard, out-of-distribution
cases, Keyway is **not perfect (0.75 Youden)**, and the one remaining miss
(`adv-03`, trailing slash) is a named, documented judgment call rather than a
hidden failure.

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
| **Keyway (adversarial)** | hardest config edge cases | **0.75 Youden** |

The honest reading is **not** "Keyway beats Snyk" — they solve different
problems. It is: *inference from code has a well-documented accuracy ceiling;
checking structured config against explicit rules does not infer, so it can be
near-exact on the cases the rules cover — and the adversarial set shows where the
rules stop covering.*

## 4b. The honest denominator: coverage of the *threat space*

The scores above answer "does the classifier do what the rules say?" They do
**not** answer the more important question a security tool must face: **of all
the ways a JWT verifier can be attacked, what fraction does Keyway even check
for?** A tool that catches 100% of the handful of issue types it knows about, and
is silent on the rest, is not comprehensive — and a corpus/CVE list we curated
around our own probes cannot reveal that gap, because it only contains things we
already detect.

So coverage is now measured against an **external threat taxonomy**
([`internal/threats`](../internal/threats)) — every documented JWT/JWKS/OIDC
verifier threat we could source from RFC 8725, OWASP, CVEs, CWE, and the
PortSwigger catalog — not against our own corpus. The generated report is
[threat-coverage.md](./threat-coverage.md) (`keyway threats coverage`).

**Current coverage: 21 of 35 documented threats (60%).** The
[generative attack harness](../internal/attack) (§4c) closed nine gaps at once
— empty-signature, `alg` case/whitespace variants, the ECDSA `(0,0)` "psychic
signature", embedded-`jwk` self-signing, `kid` path traversal, `aud` array/string
confusion, missing-`exp`, and two encoding/structural classes. The remaining 14
gaps are still named and cited in the report: JWKS delivery (`http`/redirect,
0/2), the callback-dependent `jku`/`x5u` header injections, weak keys, and a few
claim-hardening cases. The `100%`/`8-of-8` figures elsewhere in this repo are
recall *against what we check* — this 60% is recall *against the known universe*,
and it is the number that should keep going up.

The taxonomy is kept honest three ways: a test asserts every "covered" mapping
points at a probe/generator that actually exists; a bridge test asserts the
harness credit matches exactly the threats the corpus can detect end-to-end
(self-contained), so callback-dependent checks like `jku` do **not** inflate the
number; and a test fails if the catalog ever claims 100% of the whole space (the
exact overclaim this page exists to prevent).

## 4c. The generative harness (escaping the checklist)

The original 13 probes were a hand-written list — comprehensive only about the
things someone thought to add. The [attack harness](../internal/attack) replaces
that with **taxonomy-driven generators**: each emits concrete attack tokens
tagged with the threat they exercise and the verdict a *correct* verifier must
return, and a **reference oracle** (built on the `go-jose` library, configured
correctly) encodes the security invariants.

Two properties make this trustworthy rather than circular:

- **The corpus self-validates.** A known-correct verifier (the oracle) must
  reject every attack token and accept the control. If a generator ever produces
  a token that doesn't actually have the property it claims, the test fails. So
  "this token is an alg=none bypass" is *proven*, not asserted.
- **Detection is proven end-to-end.** One test fires the whole corpus at a
  deliberately-broken target (accepts everything) and requires a finding for
  every attack; another fires it at an oracle-backed target and requires **zero**
  false positives.

Because the generators are driven by the taxonomy, adding a new threat entry and
a generator for it is the whole change — coverage goes up by construction, and
the bridge test refuses to let the number claim more than the corpus can actually
detect at a single endpoint. (Live scanning of real services reuses the probe
engine's issuer minting so claim-level attacks are signed by the real key; that
integration is the next step and is why `jku`/`x5u` are generated but not yet
counted.)

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
- The **adversarial corpus scores 0.75 Youden**, names its one remaining miss,
  and counts the ambiguity against itself — the opposite of a rigged benchmark.
- Reproduce all of it: `make mutation`, `make bench`, `make validate`.
