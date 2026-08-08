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
- **Measurement:** `--per-repo --exclude-examples --dedup` →
  312 consumers discovered, 149 validate JWTs, 24 excluded as examples,
  **102 distinct configs** in the denominator.

## Prevalence (preliminary, n = 102 distinct configs)

| Check | k/n | Prevalence | 95% CI (Wilson) |
|---|---|---|---|
| P3 no algorithm pinning **in config** | 102/102 | 100.0% | [96.4%, 100%] |
| P2 no required claims | 87/102 | 85.3% | [77.1%, 90.9%] |
| P1 unbound audience | 43/102 | 42.2% | [33.0%, 51.9%] |
| P4 multi-issuer trust (descriptive) | 6/102 | 5.9% | [2.7%, 12.2%] |
| P5 clock skew > 5 min | 0/102 | 0.0% | [0.0%, 3.6%] |

## Discovery validation vs an independent parser (`validate.py`)

Ground truth = an independent YAML parse (declared issuers / audiences / claim
names), unioned **per repo**; captured = what Keyway discovered per repo. Over
**183 repos with declared auth config**:

| Field | Recall | Precision | TP | FP | FN |
|---|---|---|---|---|---|
| issuers | 86.5% | 100.0% | 147 | 0 | 23 |
| audiences | 97.0% | 100.0% | 97 | 0 | 3 |
| claims (unscoped) | 27.1% | 100.0% | 23 | 0 | 62 |
| **claims (scoped to repos with a JWT consumer)** | **82.1%** | 100.0% | 23 | 0 | 5 |

### The claims-recall investigation (resolved)

The first run showed **claims recall 15.4%**, which looked alarming. Investigated
it and it is **not a discovery bug** — precision is 100% throughout. Two fixes and
a diagnosis:

1. **Measurement artifact → fixed.** The initial `--per-file` mode split each
   `AuthorizationPolicy` (where claims live) from its `RequestAuthentication`
   (where the issuer lives), so claims had no consumer to attach to. Switching to
   **`--per-repo`** (analyse a repo's files together) raised claims recall
   **15.4% → 27.1%**.
2. **Corpus completeness, not discovery.** Of the 50 repos still missing claims,
   **45 have no `RequestAuthentication` in the corpus at all** — the crawl fetched
   an `AuthorizationPolicy` but not the matching RA (query/limit gaps), so there is
   genuinely no JWT consumer to attach claims to. Scoping recall to repos where a
   consumer exists gives **82.1%** — in line with issuers/audiences.
3. **The residual 5** (RA present, claim still missed) are **kustomize overlay/base
   splits** (Keyway reads raw manifests, not rendered kustomize) and
   selector-scoping cases where the repo-level union over-counts. These are
   config-composition limitations, documented — not a defect that should be
   "fixed" by loosening attachment (which would cost the 100% precision).

**Takeaway:** discovery is precise (100%) and its true recall is issuers 86.5% /
audiences 97.0% / claims ~82%. The remaining misses are corpus and un-rendered-templating
limitations, both named. A hand-labelled sample (worksheet emitted) remains the
gold standard to confirm this.

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
go run ./bench/measurement --path bench/measurement/corpus --per-repo --exclude-examples --dedup
go run ./bench/measurement --path bench/measurement/corpus --per-repo   # raw, for validation
python3 bench/measurement/validate.py bench/measurement/corpus bench/measurement/out
```

(`corpus/` and `out/` are gitignored; only `sources.tsv` is committed.)
