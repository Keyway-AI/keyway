# Zero‑False‑Alarm Drift Classification for Auth Contracts

*Keyway Research Note 02 · 2026‑08‑07 · reproduces against `v0.1.0`*

> **TL;DR** — Text‑diffing config produces an alarm on every redeploy, so it gets
> muted. We classify *semantic* drift between two auth‑contract versions with a
> small fixed rule table and measure it as a binary detection problem. On 800
> before/after pairs — half real changes, half redeploy noise — the classifier
> achieves **100% recall and 0% false‑positive rate** (Youden J = 1.0). Because a
> perfect score on a self‑authored corpus is suspicious, we show it survives
> **mutation testing** (24 mutants, 100% killed modulo 2 equivalent) and drops to
> **J = 0.75** on a held‑out adversarial corpus.

## 1. Problem

A naive `git diff` of deployment YAML lights up on list reordering, replica
bumps, label renames, image‑tag changes, and comments — none of which change what
tokens are accepted. Engineers mute the noisy signal, and then the *one* diff that
widened an audience or dropped a required claim is muted with it. The detector's
job is therefore **asymmetric**: never miss a real contract change, and never fire
on noise.

## 2. Method

Keyway diffs two `ContractVersion`s field‑by‑field and classifies each change with
a **fixed rule table** (`internal/diff`) — deliberately *not* a learned model, so
there are no weights to overfit (§4). The taxonomy:

| Class | Semantics | Security reading |
|---|---|---|
| **widened** | accepts *more* than before (extra audience, added issuer, longer JWKS cache, `alg=none` allowed) | potential regression — high attention |
| **narrowed** | accepts *less* (required claim added, audience removed) | may break consumers — blast‑radius attention |
| **added / removed** | a consumer or edge appeared/disappeared | contextual |

Each change carries a severity (e.g. *audience widened* = medium, *required claim
removed* = critical) and its provenance from the discovery layer, so it is
attributable to a specific line.

**Evaluation as detection.** For each of N before/after pairs we ask: did the
classifier alert? A *real change* pair should alert (positive); a *noise* pair
should not (negative). We report recall (TPR), false‑positive rate (FPR),
precision, and Youden's J = TPR − FPR (1.0 is perfect).

The corpus's noise half is the load‑bearing part. `0202-noisy-redeploy` mutates
six unrelated fields at once while holding the login contract fixed; the detector
must stay silent.

## 3. Results

Reproduced with `make bench` (`bench/out/scorecard.json`, 2026‑08‑07):

| Corpus | TP | FP | TN | FN | Recall | FPR | Youden J |
|---|---|---|---|---|---|---|---|
| L3 (800 pairs: 400 real / 400 noise) | 400 | 0 | 400 | 0 | **1.000** | **0.000** | **1.000** |
| L3‑realistic (rendered YAML → real discovery) | 240 | 0 | 160 | 0 | 1.000 | 0.000 | 1.000 |

Zero false alarms across 560 negative (noise) pairs; zero missed real changes.

## 4. Is it overfit?

**A rule table can't memorise training points**, but a corpus that only exercises
paths the rules already handle proves nothing. Two stress tests:

- **Mutation testing** (`make mutation`): inject faults into `internal/diff`
  (`<`→`<=`, negated booleans) and re‑run the whole suite per mutant. Result: **24
  mutants, 100% mutator coverage, every behaviour‑changing mutant killed**; the 2
  survivors are provably equivalent (a boundary flip guarded by a `!=` above it).
  A mutation score cannot be inflated by adding agreeable corpus rows.
- **Held‑out adversarial corpus** (not gated): **TP=4, FP=1, TN=3, FN=0 → J =
  0.75.** The classifier still misses nothing but produces one false positive on
  the hard negatives. This is the honest generalisation number.

## 5. Threats to validity

- The 1.0 is on a self‑authored corpus; 0.75 (§4) is the held‑out estimate.
- Severity assignments encode our judgement of impact; they are documented rules
  (PRD §9.2), not empirical measurements of exploitability.
- Classification operates on the *derived* contract, so discovery errors (Note 01)
  would propagate — the perfect L3‑realistic score bounds that risk on‑corpus.

## 6. Reproduce

```bash
make bench        # L3 + L3-realistic + L3-adversarial
make mutation     # mutation score (installs gremlins if absent)
```

Prev: **[Note 01 — Contract Discovery](01-contract-discovery.md)** ·
Next: **[Note 03 — Adversarially Verifying JWT Verifiers](03-adversarial-verification.md)**.
