# Post-submission roadmap

Both SSR 2026 papers are submitted (2026-08-19, EasyChair):

- **Submission 20** — *SoK: Authorization for Autonomous Agents* (SoK track)
- **Submission 21** — *Vision: Measuring JWT Authorization Contracts in the Wild* (Vision track)

Next dates: notification **2026-10-05**, camera-ready **2026-10-15**, conference
**Dec 13–15**. Submissions are editable on EasyChair until **2026-09-15**.

This file tracks what remains **from our side**: evaluation/research to turn the
preliminary results into citable ones, engineering features that support that, and
manual/ops work. The two research gates (**G1** hand-labelling, **G2** corpus
scaling) are the critical path: every full-research venue (USENIX, IMC, NDSS, S&P)
is blocked on them. See [`submissions/TRACKER.md`](submissions/TRACKER.md) for
venue deadlines and [`../../bench/measurement/FINDINGS.md`](../../bench/measurement/FINDINGS.md)
for the methodology gaps this addresses.

Legend: ☐ todo · ◑ in progress · ☑ done · ⧗ blocked (needs a human/token)

---

## A. Evaluation / research

| # | Task | Owner | Blocked on | Status |
|---|---|---|---|---|
| A1 | **G1 — hand-label the validation sample** (~85 rows; packet ready via `make_labeling_packet.py`). Gold-standard precision/recall. Decide single- vs two-annotator (κ). | Human | — | ⧗ |
| A2 | **G2 — scale corpus 428 → 10³–10⁴ repos** (crawl + re-run); add OIDC/MCP sources; report selection bias. | Me (drive) | fresh GH token | ⧗ |
| A3 | Re-run prevalence + validation on the scaled, labelled corpus. | Me | A1, A2 | ☐ |
| A4 | Co-occurrence analysis of weaknesses (e.g. unbound-audience ∧ no-claims). Shipped the `cooccur` package: pairwise joint prevalence, conditionals P(B\|A)/P(A\|B), and lift, printed and in `summary.json`. | Me | — | ☑ |
| A5 | Quantify the static-vs-runtime frontier (RQ4): what fraction is decidable from config vs needs live probing. Shipped: each check tagged `static_decidable`, and a rollup (`static_runtime_frontier` in summary.json + printed) reporting how many check types and raised flags are config-conclusive vs carry the library-default caveat (P3). | Me | — | ☑ |
| A6 | Drift-in-the-wild: longitudinal analysis over real commit history (classifier already validated). Shipped the `drift` package + `bench/measurement/drift/cmd`: over a timeline of contract versions it tallies widened/narrowed/net/breaking and the widen:narrow ratio, answering "do contracts widen more than they narrow?". Runs on any timeline; the real commit-history corpus still needs G2 to crawl. | Me | ◑ |
| A7 | **Paper B empirical**: fire the adversarial agent-token corpus at real **MCP-server-class** verifiers and measure prevalence of the agent-auth weaknesses. **Done (local, no token):** `bench/verifiers/run-mcp-eval.sh` + targets `mcp-fastmcp` (FastMCP `JWTVerifier`) and `mcp-python-sdk` (reference SDK `BearerAuthBackend`, default config). Result ([MCP-FINDINGS.md](../../bench/verifiers/MCP-FINDINGS.md)): the **reference SDK default accepts MCP-01 token passthrough + MCP-02 unbound audience** (resource binding is opt-in, `resource_server_url` default None); **both MCP verifiers accept a non-expiring token** (SCOPE-02, 2/2); FastMCP binds audience correctly. Connects to the 9-library eval (a base-SDK server inherits its JWT library's defaults). *Open:* config-corpus **prevalence** at scale (many deployed MCP configs) still wants the G2 crawl token or a user decision on fetching live `.well-known/oauth-protected-resource` endpoints. | Me | ◑ |

## B. Features / engineering

| # | Task | Owner | Status |
|---|---|---|---|
| B1 | **Render Helm/kustomize before discovery** — ~1 in 5 corpus configs are under-resolved templates (17% Helm, ~2% kustomize). Biggest recall lever. Shipped as the `render` package + `--resolve-templates` (neutralizes standalone templated files today; `helm template`/`kustomize build` runs on whole trees once G2 fetches them) + `out/templating.json` coverage report. | Me | ☑ |
| B2 | Near-duplicate / fork-aware dedup (current dedup is exact-signature only). Shipped the `dedup` package: canonical signature (folds issuer host/slash, algorithm case, order) + a reported Jaccard near-duplicate diagnostic. Fork-of-upstream collapse still needs GitHub fork metadata from the G2 crawl. | Me | ☑ |
| B3 | Broaden discovery sources (more gateways / OIDC / MCP) for G2 representativeness. **Done:** added `internal/discovery/mcp` (RFC 9728 Protected Resource Metadata → the MCP agent-auth contract: `resource`→audience, `authorization_servers`→issuers), `internal/discovery/kong` (openid-connect + jwt plugins, global/service/route scope merge) and `internal/discovery/apisix` (openid-connect + jwt-auth), all wired into `keyway discover`. nginx deferred (non-YAML DSL, issuer/audience not first-class). | Me | ☑ |
| B6 | **Productize the verifier-conformance eval.** Done: `internal/conformance` turns a `keyway probe --harness` run into a per-verifier / per-threat conformance report (findings = adversarial tokens a verifier wrongly accepted; prevalence across verifiers that trusted the issuer), via `--report conformance` / `--report conformance-json`. `bench/verifiers/run-eval.sh` now emits `conformance.{txt,json}` instead of a hand-written table. | Me | ☑ |
| B7 | **Cloud/hosted persistence.** Done: `cloud/postgres` implements `cloud.Store` on pgx (tenant-scoped; analyses as JSONB + denormalized columns; FK cascade), with embedded golang-migrate migrations under a `schema_migrations_cloud` table so it can share a DB with the engine store. `keyway-cloud` selects it when `KEYWAY_DB_URL` is set, else the in-memory store. Verified against a live Postgres. | Me | ☑ |
| B4 | Package the agent-auth analyzer as a public, documented CLI/Action (Paper B "we release an open-source analyzer"). Done: static DEL-02 check closes the paper/tool gap (analyzer now covers all 6); `keyway agent inspect --fail-on` gates CI; composite GitHub Action `actions/agent-inspect`; README + `docs/ci.md` quickstart. | Me | ☑ |
| B5 | One-command reproducible artifact bundle with pinned deps (for the DOI + Artifact Evaluation). Done: `make reproduce` (`scripts/reproduce.sh`) runs every offline-verifiable claim into `artifact/` with a claim→output map; `REPRODUCE.md` documents it and the token-gated corpus steps. | Me | ☑ |

## C. Manual / ops / calendar

| # | Task | Owner | When | Status |
|---|---|---|---|---|
| C1 | **Rotate the GitHub PAT** pasted earlier in chat (exposed in transcript). | Human | now | ⧗ |
| C2 | Confirm EasyChair confirmation emails arrived (both papers). | Human | now | ☐ |
| C3 | Overleaf visual proof of both PDFs; optionally swap a polished PDF before the deadline. | Human | before 2026-09-15 | ☐ |
| C4 | If accepted (notify 2026-10-05) → camera-ready by 2026-10-15: de-anonymize the author block (uncomment the real author/institute lines in the SSR wrappers). | Me | Oct | ☐ |
| C5 | DOI-pinned Zenodo snapshot of instrument + harness + corpus manifest. | Human upload; I prep | camera-ready | ☐ |
| C6 | Artifact-Evaluation "Available/Reproduced" badges; register ≥1 author. | Human | if accepted | ☐ |
| C7 | Extended full Paper A → **USENIX Sec '27 Cy2 (2027-01-26)** or IMC, once G1+G2 are done. | Me (draft) | after A1–A3 | ☐ |

---

**Critical path:** A1 (G1) + A2 (G2). Everything that makes the papers
full-research grade hangs off those. The B-tasks improve the numbers regardless of
the gates and need no token, so they run in parallel.
