# Paper A — A Measurement Study of Token‑Auth Contracts and Drift in the Wild

**Author team (planned):** Archit Sharma, Garima Mann.

*Design document. Status: proposed. This is a plan — it contains **no results**,
only the study we would run. Nothing here may be cited as a finding until the
study is executed.*

## 1. Why this paper (the honest case)

Keyway's current accuracy numbers come from a **self‑authored** corpus. In
academia that is near‑disqualifying no matter how honest the caveats: 100% on a
benchmark we wrote proves internal consistency, not a fact about the world. A
measurement study fixes the root problem by turning the tool from *the claim* into
*the instrument*: we point Keyway's existing discovery engine at a large corpus of
**real, public** deployment configuration and report **what is actually out
there.** That is an empirical contribution the synthetic benchmark can never be,
and it is the most defensible first real paper.

## 2. Contribution statement

> We present the first large‑scale measurement of JWT/OIDC and AI‑agent
> authorization *contracts* as expressed in real deployment configuration. Over N
> repositories / M distinct services we quantify the prevalence of concrete
> auth‑verification weaknesses (unbound audiences, permissive algorithm sets,
> missing required claims, non‑expiring agent tokens, unverifiable delegation),
> characterise how these contracts **drift** across version history, and release
> the dataset and tooling for reproduction.

Three defensible contributions: (i) a **method** for deriving auth contracts from
heterogeneous config at scale; (ii) an **empirical characterisation** of the
population (prevalence, co‑occurrence, drift dynamics); (iii) a **released
artifact** (dataset + tooling) enabling follow‑on work.

## 3. Research questions

- **RQ1 (prevalence).** How common are specific, RFC‑defined verification
  weaknesses in public auth config? (e.g., what fraction of services binding JWTs
  declare no audience, contravening RFC 8707/9728?)
- **RQ2 (drift).** Across version history, how often do auth contracts *widen*
  (accept more) vs *narrow*, and are widenings correlated with reviewable events
  (a commit, a PR, a dependency bump)?
- **RQ3 (agents).** In the emerging MCP/agent ecosystem, what is the prevalence of
  unbound audiences, missing `act` delegation, omnibus scopes, and non‑expiring
  credentials?
- **RQ4 (detectability).** What fraction of the weaknesses in RQ1–RQ3 are
  detectable statically from config alone vs require runtime probing? (This is
  where Keyway's static/live split becomes an empirical result, not a design
  choice.)

## 4. Data & methodology

### 4.1 Corpus construction
- **Sources.** Public GitHub (code search for `RequestAuthentication`,
  `jwtRules`, Envoy `jwt_authn`, OIDC client registrations, MCP server manifests),
  Artifact Hub / public Helm charts, Istio/Envoy sample repos, and public MCP
  server registries. Record commit SHA + license for every artifact.
- **Sampling & bias.** Pre‑register inclusion criteria; report selection bias
  explicitly (public ≠ representative of private enterprise config). Deduplicate
  forks. Stratify by ecosystem (mesh vs gateway vs IdP vs MCP).
- **Scale target.** Enough for tight confidence intervals on prevalence — aim for
  10³–10⁴ distinct services; report exact N with CIs, never bare percentages.
- **Ethics.** Only public config; **no exploitation**, no probing of third‑party
  live endpoints (that would be unauthorised — see §6). Responsible‑disclosure
  plan for any high‑severity finding tied to an identifiable owner; IRB/again‑ethics
  note in the paper.

### 4.2 Extraction
Keyway's `discovery` + `contract` + `diff` packages, run read‑only over checked‑out
config. Every extracted field keeps provenance (already implemented). Manually
label a **stratified validation sample** (~300–500 services) to measure discovery
precision/recall against ground truth — this replaces the self‑authored benchmark
as the accuracy claim, because the labels are on *real* data.

### 4.3 Analysis
Prevalence with Wilson CIs; drift as a longitudinal analysis over commit history;
co‑occurrence via association rules; detectability as the static‑vs‑runtime split
from RQ4. All analysis scripts released.

## 5. Related work to position against (must‑cite)
- JWT/OIDC security analysis and known attacks (`alg=none`, key confusion, JWKS
  issues) — establish that *individual attacks are known*; our novelty is the
  **population‑scale contract + drift** view, not the attacks themselves.
- Config/IaC security scanning (Checkov, KICS, tfsec, Semgrep) — contrast:
  pattern‑matching vs contract derivation with provenance.
- Contract testing (Pact, consumer‑driven contracts) — borrow the framing, note
  it targets API shape, not auth verification.
- Software‑supply‑chain and misconfiguration measurement studies — our
  methodological template.
- Emerging MCP/agent‑authorization literature — likely sparse; being early is the
  point (feeds Paper B).

## 6. Threats to validity (state them, don't hide them)
- **External:** public config may not represent production enterprise config.
- **Construct:** static discovery sees configuration + library defaults, not the
  running verifier; RQ4 measures exactly this gap rather than pretending it away.
- **Internal:** discovery errors — bounded by the §4.2 labelled validation sample.
- **Ethical:** measurement must be non‑invasive; **no live probing of others'
  endpoints** appears in the paper as a hard constraint, not a footnote.

## 7. Artifact evaluation (a credibility multiplier we can actually bank)
Target the **Artifact Evaluation** track (USENIX/CCS/S&P/NDSS). Keyway's
`make bench` / `make mutation` / `keyway threats coverage` reproducibility ethos is
tailor‑made for **"Artifacts Available + Reproduced"** badges. Package the corpus
(licences permitting), the extraction pipeline pinned to a commit, and a one‑command
reproduction. Badges are honest, citable credibility and are where most tool papers
are weakest.

## 8. Target venues (see `venues-and-cfps.md` for dates)
- **Primary:** USENIX Security (measurement‑study friendly) → NDSS → IEEE S&P.
- **Realistic first win / de‑risk:** a measurement‑oriented workshop, or ACSAC.
- **Do not** submit to a top track until the labelled validation (accuracy on real
  data) and CIs are in hand — that is the whole point of the paper.

## 8a. Status — the instrument exists

The measurement harness is built: **[`bench/measurement`](../../../bench/measurement/)**
(`make measure`). It runs discovery + contract read-only over a config corpus and
emits a labelled `dataset.jsonl` plus `summary.json` with per-check prevalence and
**Wilson 95% CIs**, for the checks P1–P5 in §3/§4 (each cited). It has been
**pilot-run** on the bundled real manifests only (n≈5) to prove the pipeline — those
numbers are **not** a study result and are not cited anywhere. Remaining research
work: scale the corpus toward the §4.1 target N (broaden GitHub/Helm/MCP sources,
per-repo runs), build the §4.2 hand-labelled validation set, and run the §4.3
analysis.

## 9. Effort & timeline (honest)
Corpus + pipeline hardening ~4–6 weeks; labelling + analysis ~4–6 weeks; writing +
related work ~3–4 weeks. **~3 months of focused work** to a credible submission,
longer if solo. A co‑author who has shipped a measurement study before compresses
this materially (see the academic README).
