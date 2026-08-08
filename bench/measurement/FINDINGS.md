# Preliminary measurement run — NOT a publishable result

> **Do not cite these numbers.** This is a scaled-up but still-preliminary run of
> the Paper A pipeline on an **unvalidated** corpus, recorded to show the
> instrument works on real data and to expose exactly what the real study must
> fix. Treat everything here as a pilot, not a finding.

## What was run (2026‑08‑08)

- **Corpus:** 603 real config files from **428 public GitHub repos**, fetched by
  `crawl.sh` (Istio `RequestAuthentication`, Envoy `jwt_authn` / `remote_jwks`,
  Istio `request.auth.claims` / `AuthorizationPolicy`; ≤5 files/repo; public only;
  repo + blob SHA + license recorded in [`sources.tsv`](sources.tsv)). This was a
  partial crawl (rate-limited), not the target N.
- **Measurement:** `--per-file --exclude-examples --dedup` →
  347 consumers discovered, 156 validate JWTs, 26 excluded as examples,
  **103 distinct configs** in the denominator.

## Prevalence (preliminary, n = 103 distinct configs)

| Check | k/n | Prevalence | 95% CI (Wilson) |
|---|---|---|---|
| P3 no algorithm pinning **in config** | 103/103 | 100.0% | [96.4%, 100%] |
| P2 no required claims | 91/103 | 88.3% | [80.7%, 93.2%] |
| P1 unbound audience | 43/103 | 41.7% | [32.7%, 51.4%] |
| P4 multi-issuer trust (descriptive) | 3/103 | 2.9% | [1.0%, 8.2%] |
| P5 clock skew > 5 min | 0/103 | 0.0% | [0.0%, 3.6%] |

(Raw, before exclusion/dedup: 156 consumers — P3 100%, P2 89.7%, P1 48.1%.)

## Discovery validation vs an independent parser (`validate.py`)

Ground truth = an independent YAML parse of each file (declared issuers /
audiences / claim names); captured = what Keyway discovered. Over **229 files with
declared auth config**:

| Field | Recall | Precision | TP | FP | FN |
|---|---|---|---|---|---|
| issuers | 85.9% | 100.0% | 152 | 0 | 25 |
| audiences | 97.1% | 100.0% | 102 | 0 | 3 |
| **claims** | **15.4%** | 100.0% | 18 | 0 | 99 |

**This is the most valuable output so far, and it is not flattering.** Discovery
is **perfectly precise** (never captures a spurious value) but **claims recall is
low (15.4%)** and issuer recall is 85.9%. Either discovery is missing claim
extractions (a real gap to fix) **or** the independent parser over-counts
`request.auth.claims[...]` references (e.g., in `AuthorizationPolicy` conditions
Keyway models differently). Disambiguating this is precisely why the paper needs a
**hand-labelled** sample — `validate.py` emits `out/labeling-worksheet.jsonl` to
do exactly that.

## Why none of this is a result yet

1. **No human validation.** The independent-parse proxy above is a signal, not
   gold-standard ground truth. The claims-recall gap must be adjudicated by hand.
2. **Partial corpus, selection bias.** 428 repos, public only, one rate-limited
   crawl — not the target 10³–10⁴, and public config skews to samples/OSS.
3. **P3 = "not pinned in config" ≠ "vulnerable."** Almost certainly the verifying
   library's default — the static-vs-runtime gap (RQ4), not a 100% vuln rate.
4. **Dedup is by exact config signature; near-misses remain.** Fork/near-duplicate
   handling needs strengthening.

## Reproduce

```bash
GH_TOKEN=<token> MAX_PAGES=2 bash bench/measurement/crawl.sh
go run ./bench/measurement --path bench/measurement/corpus --per-file --exclude-examples --dedup
python3 bench/measurement/validate.py bench/measurement/corpus bench/measurement/out
```

(`corpus/` and `out/` are gitignored; only `sources.tsv` is committed.)
