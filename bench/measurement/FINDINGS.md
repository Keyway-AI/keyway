# Preliminary measurement run — NOT a publishable result

> **Do not cite these numbers.** This is a first end-to-end run of the Paper A
> pipeline on a **small, unvalidated** corpus, recorded to show the instrument
> works on real data and to expose the limitations that the actual study must
> fix. Treat everything here as a pilot, not a finding.

## What was run

- **Corpus:** 192 real config files fetched from **127 public GitHub repos** by
  `crawl.sh` (queries: Istio `RequestAuthentication`, Envoy `jwt_authn`, Istio
  `request.auth.claims`; ≤5 files/repo; public repos only; blob SHA + license
  recorded in [`sources.tsv`](sources.tsv)). Fetched 2026‑08‑08.
- **Measurement:** `go run ./bench/measurement --path corpus --per-file` →
  130 consumers discovered, **50 validate JWTs** (the denominator).

## Numbers (preliminary, n = 50)

| Check | k/n | Prevalence | 95% CI (Wilson) |
|---|---|---|---|
| P3 no algorithm pinning **in config** | 50/50 | 100.0% | [92.9%, 100%] |
| P2 no required claims | 40/50 | 80.0% | [67.0%, 88.8%] |
| P1 unbound audience | 24/50 | 48.0% | [34.8%, 61.5%] |
| P4 multi‑issuer trust (descriptive) | 1/50 | 2.0% | [0.4%, 10.5%] |
| P5 clock skew > 5 min | 0/50 | 0.0% | [0.0%, 7.1%] |

## Why these are NOT results yet (the honest gaps)

1. **Tiny n.** 50 consumers → wide CIs. The paper targets 10³–10⁴ (this was one
   crawl pass, 1 page/query).
2. **Near‑duplicate examples not removed.** Only **41 of the 50** are distinct
   `(issuers, audiences)` configs; the canonical Istio sample
   (`testing@secure.istio.io`) appears **6×**. The study must dedup by config
   content and exclude vendored/tutorial copies — copying the docs is not a
   finding about production.
3. **"No algorithm pinning in config" ≠ "vulnerable."** P3 = 100% almost
   certainly reflects that operators rely on the verifying **library's default**
   algorithm handling, which config doesn't restate. This is the **static‑vs‑runtime
   gap (RQ4)**, not a 100% vulnerability rate. Reported as "not pinned in config."
4. **No labeled validation.** Discovery precision/recall on this real data is
   unmeasured; the paper needs a hand‑labeled sample (this replaces the
   self‑authored benchmark as the accuracy claim).
5. **Selection bias.** Public config skews to samples, demos, and OSS — not
   representative of private enterprise config. Must be stated.

## Reproduce

```bash
GH_TOKEN=<your-token> MAX_PAGES=1 bash bench/measurement/crawl.sh   # → corpus/ + sources.tsv
go run ./bench/measurement --path bench/measurement/corpus --per-file
```

(The fetched `corpus/` and `out/` are gitignored; only `sources.tsv` — the
reproducible, attributable manifest — is committed.)
