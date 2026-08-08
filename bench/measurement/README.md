# Measurement harness (Paper A instrument)

The instrument for **[Paper A](../../docs/research/academic/paper-a-measurement-study.md)** —
"A Measurement Study of Token-Auth Contracts and Drift in the Wild." It runs
Keyway's discovery + contract engine **read-only** over a corpus of real
deployment config and reports the **prevalence** of concrete, cited
auth-verification weaknesses across the population, with Wilson 95% confidence
intervals.

It turns the tool from *the claim* into *the instrument*: instead of grading
Keyway on a corpus we wrote, we point it at the world and report what's there.

## Run

```bash
make measure                                   # pilot on the bundled real manifests
go run ./bench/measurement --path <dir> --out bench/measurement/out
```

Outputs:
- `out/dataset.jsonl` — one labelled observation per JWT-validating consumer
  (issuers, audiences, algorithms, required claims, per-check flags, provenance).
- `out/summary.json` — per-check prevalence with Wilson 95% CIs.

## The checks (each grounded in a normative source)

| ID | What it measures | Source |
|---|---|---|
| **P1** unbound audience | validates a token but declares no audience → not bound to a resource | RFC 8707 / RFC 9728 |
| **P2** no required claims | requires nothing beyond issuer/audience → no authorization constraint | RFC 8725 §3.9 |
| **P3** no algorithm pinning | pins no signing algorithm in config → relies on the library default | RFC 8725 §3.1 |
| **P4** multi-issuer trust | trusts >1 issuer (descriptive, wider trust surface) | — |
| **P5** wide clock skew | accepts >5 min skew | RFC 8725 §3.11 / operational norm |

The **denominator** is consumers that actually validate JWTs (≥1 expected issuer),
not all discovered services.

## Honesty caveats (read before citing anything)

- **This is a PILOT until run at scale.** The bundled corpus (`bench/oss/manifests`,
  n≈5) exists only to prove the pipeline. **Do not cite pilot numbers as a study
  result** — with n≈5 the confidence intervals are (correctly) enormous. The paper
  needs 10³–10⁴ distinct services (see the paper design).
- **"Absent in config" ≠ "absent at runtime."** P3 in particular: a missing
  algorithm list often means the verifying *library* pins a safe default. This is
  not a false positive — it is precisely the **static-vs-runtime gap** the paper
  measures as RQ4. Report it as "not pinned in config," never as "vulnerable."
- **Public config is not representative** of private enterprise config; selection
  bias must be reported, not hidden.
- **This is measurement, not a verdict** on any single project. No project is named
  as insecure; only population prevalence is reported.
- **Discovery has error.** The paper requires a hand-labelled validation sample to
  measure discovery precision/recall on *real* data — that replaces the
  self-authored benchmark as the accuracy claim.

## Building the corpus (`crawl.sh`)

[`crawl.sh`](crawl.sh) crawls **public** GitHub for real auth config (Istio
`RequestAuthentication`, Envoy `jwt_authn`, Istio claim-based authz), fetches each
file, and records repo + blob SHA + license in [`sources.tsv`](sources.tsv). It is
read-only and reads the token from `GH_TOKEN` — **never hardcode or commit a
token.**

```bash
GH_TOKEN=<your-token> MAX_PAGES=1 MAX_PER_REPO=5 bash bench/measurement/crawl.sh
go run ./bench/measurement --path bench/measurement/corpus --per-file
```

`--per-file` runs discovery on each file independently so same-named services
across unrelated repos aren't merged by a colliding StableID — correct for a
multi-repo population measurement. The fetched `corpus/` and `out/` are gitignored;
only `sources.tsv` (the reproducible, attributable manifest) is committed.

### Study-grade flags

- `--exclude-examples` drops tutorial/sample/vendored copies (by source path).
- `--dedup` collapses identical configs (issuers+audiences+algorithms+required_claims)
  so a RequestAuthentication pasted across many repos counts once.

```bash
make measure           # per-file + exclude-examples + dedup
make measure-validate  # discovery precision/recall vs an independent parse
```

### Validation (`validate.py`)

`validate.py` measures Keyway's discovery **precision/recall** against an
independent YAML parse of the same files, and emits `out/labeling-worksheet.jsonl`
to bootstrap the hand-labelled gold-standard sample the paper needs. The
independent parse is a proxy, not human ground truth.

A scaled preliminary run (603 files / 428 repos → 103 distinct configs; discovery
precision 100%, but claims recall only ~15% — a real gap to adjudicate) is
recorded in **[FINDINGS.md](FINDINGS.md)** — explicitly **not** a publishable
result. Scaling to the target N and hand-labelling the validation sample is the
remaining research work.

## Ethics

Only public config, read only. **No probing of third-party live endpoints** and
**no real user tokens** — those would be unauthorised. Responsible disclosure for
any identifiable high-severity finding. See the paper design's ethics section.
