# Keyway: Deriving, Versioning, and Adversarially Verifying Token‑Auth Contracts

**A technical whitepaper.** Version 1.0 — 2026‑08‑07. Corresponds to Keyway `v0.1.0`.

> Every quantitative claim in this document is reproduced from the repository with
> a named command. Where a result is favourable *and* self‑authored, the honest
> counter‑evidence is reported alongside it. Nothing here is asserted that
> `make bench`, `make mutation`, or `keyway threats coverage` cannot regenerate.

---

## Abstract

Distributed systems authenticate requests with bearer tokens — JWTs between
services, and increasingly OAuth/MCP tokens carried by AI agents. Whether a token
is *accepted* depends on a verifier's configuration: the issuers it trusts, the
audiences it binds to, the algorithms it allows, the claims it requires. That
configuration is an implicit **contract**, and it is almost never written down.
When it drifts — a rotated key, a new issuer, a widened audience, a dropped claim
— services break in production, or worse, silently accept tokens they should
reject.

Keyway makes that contract explicit. It (1) **discovers** what each service
expects from a token by parsing real deployment configuration and, optionally,
live cluster state; (2) **versions** the derived contract as a content‑addressed
snapshot; (3) **classifies drift** between versions into a small, severity‑graded
taxonomy; (4) **adversarially probes** staging verifiers with minted attack
tokens to prove a correct verifier is enforced; and (5) extends the same model to
**AI‑agent authorization** (MCP audience binding, delegation, scope, expiry).

On a benchmark of **1,226** before/after configuration pairs, Keyway's drift
classifier achieves **100% recall and 0% false‑positive rate** (Youden's J = 1.0)
on the gated corpus. Because 100% on a self‑authored corpus proves consistency and
not generalisation, we stress‑test the claim two ways: **mutation testing** kills
every behaviour‑changing fault injected into the classifier, and a **held‑out
adversarial corpus** scores **J = 0.75**, with each miss named. Detection is
measured against a cited taxonomy of **50** documented threats, of which Keyway
covers **27 (54%)**; every gap is enumerated. We argue this honest, reproducible
framing is the appropriate standard for a security tool.

---

## 1. The problem: auth correctness is a runtime property

Consider a mesh of services that accept JWTs. Service A trusts issuer *I₁* with
audience `a.svc`; service B trusts *I₁* and *I₂*; a gateway strips and re‑signs.
An engineer rotates *I₁*'s signing key, or migrates B to a new identity provider,
or shortens a token's lifetime. **Which running services now reject valid users?
Which now accept tokens they shouldn't?** The answer is not in any one file — it
is an emergent property of every verifier's configuration *and* its runtime
behaviour (how long it caches JWKS, whether it actually enforces `exp`).

Two classes of tooling exist today, and neither answers the question:

- **Static code scanners** (Snyk, Semgrep, SonarQube) find *bug patterns in
  source*. They are valuable, but they reason about code, not about which
  *deployed* verifier trusts which issuer; they cannot enumerate the blast radius
  of a key rotation, and pattern‑matching auth logic has a well‑documented
  false‑positive ceiling.
- **Gateways and IdPs** (Envoy, Istio, Keycloak, Auth0) *issue and enforce* auth.
  They are the thing under test, not a test of it. "Is enforcement correct?" is
  precisely the question they cannot answer about themselves.

Keyway occupies the empty third position: **verification of the auth contract**,
treated as a first‑class, versioned, testable artifact.

---

## 2. Approach

Keyway is a pipeline of five stages. Each is a package with a narrow interface, so
the CLI, the self‑hosted server, and the hosted Cloud layer all run identical
logic (`internal/discovery`, `internal/contract`, `internal/diff`,
`internal/probe`, `internal/agentauth`).

```
 discovery ──┐
             ├─▶ contract build ─▶ hash/version ─▶ diff ─▶ classify ─▶ notify
 issuers  ───┤          │                                     ▲
             │          ▼                                     │
 libdefaults │       probe engine (13 probes) ────────────────┘
             │          │
             └──────────┴─▶ blast radius + grace period
```

### 2.1 Discovery — deriving the consumer inventory

A **consumer** is anything that validates tokens. Keyway derives consumers from
configuration rather than asking humans to declare them:

- **Istio** `RequestAuthentication` (issuer, audiences, JWKS URI) and
  `AuthorizationPolicy` (required claims);
- **Envoy** JWT‑authentication filters;
- **Kubernetes** workload metadata for stable identity;
- **OIDC discovery** documents and **Keycloak** client registries;
- optionally, **live cluster state** via client‑go for in‑cluster CRDs.

Each discovered field carries **provenance** (source file, locator, confidence),
so every downstream finding is attributable to the exact line that produced it. A
**library‑defaults database** fills in verifier behaviour that config omits (e.g.
a library's default clock skew or JWKS cache TTL), because the contract includes
what the verifier does *by default*, not only what is written.

### 2.2 Contract & versioning

Discovered consumers are assembled into a `ContractVersion`: a normalised graph of
issuers, consumers, and the edges between them, reduced to a **content hash**. A
new snapshot that hashes identically to the previous one is a no‑op; a different
hash is a new version. This makes "did the auth contract change?" a cheap,
deterministic equality check, and makes history a sequence of immutable,
diffable versions. The first snapshot for a project is a **baseline** (nothing to
compare); every subsequent one is diffed against its predecessor.

### 2.3 Drift classification

The `diff` stage walks two contract versions field‑by‑field and classifies each
change. The classifier is a **fixed rule table**, not a machine‑learning model —
there are no learned weights (see §4.1 for why that matters to the accuracy
claim). The taxonomy is deliberately small:

| Class | Meaning | Example | Typical severity |
|---|---|---|---|
| **widened** | verifier now accepts *more* | audience `+"*"`, second issuer added | medium–critical |
| **narrowed** | verifier now accepts *less* | required claim added, audience removed | low–high |
| **added / removed** | a consumer or edge appeared/disappeared | new service adopts auth | contextual |

The key design goal is **asymmetric attention with zero noise**: a *widened*
contract (a service that now trusts something new) is a potential security
regression and must never be missed; a routine redeploy that reorders a list or
bumps a replica count must never fire. §3 shows this is achieved on the benchmark.

### 2.4 Blast radius & grace period

Given a proposed change (rotate this key, remove this claim, add this issuer),
Keyway resolves **exactly which consumers break** and which are safe, with a
per‑consumer verdict and reason. Where a consumer's JWKS‑refresh behaviour has
been observed, the safe **grace window** for a rotation is *measured*, not
guessed.

### 2.5 Adversarial probing — proving enforcement

Discovery and diff are static; they describe what *should* happen. The probe
engine tests what *does*. A taxonomy‑driven harness **mints real tokens** for
`alg=none`, HS/RS key confusion, `jku`/`kid` injection, expiry and forged‑claim
attacks, and fires them at **staging** endpoints — proving a correct verifier
rejects each one. An *accepted* attack token is a confirmed, live vulnerability,
not a heuristic guess. The engine is **deny‑by‑default**: it refuses to run
outside a staging allowlist, and a hard guard blocks production targets.

### 2.6 Agent authorization

The same contract model extends to AI agents, where the auth layer is new and
largely unverified. `keyway agent inspect` statically checks an agent/MCP token
for: **audience binding** (is the token bound to the resource, per RFC 8707 /
RFC 9728, or replayable?), **delegation** (does an on‑behalf‑of token carry a
verifiable `act` chain, per RFC 8693?), **scope minimisation** (omnibus `admin:*`
grants), and **expiry** (non‑expiring credentials). Every finding cites its
normative source.

---

## 3. Evaluation

### 3.1 Benchmark design

The accuracy question is a binary detection problem: for each before/after pair,
should Keyway alert? A good detector must **catch every real change** (recall) and
**stay silent on noise** (low false‑positive rate) — because a noisy detector gets
muted, and the alert that mattered is muted with it.

The corpus (`bench/corpus`, reproduced with `make bench`) contains **1,226**
pairs across three layers:

| Layer | Pairs | What it exercises |
|---|---|---|
| **L1** (file discovery) | 26 | real YAML → discovery correctness |
| **L3** (diff) | 800 | 400 real changes + 400 noise pairs |
| **L3‑realistic** | 400 | rendered YAML → **real discovery engine** → diff |

The **noise** half is the hard part. The headline noise scenario,
`0202-noisy-redeploy`, changes *six* things at once (list order, replica count, a
team label, an image tag, an env var, a comment) while keeping the login contract
identical — Keyway must stay completely silent.

### 3.2 Results (gated corpus)

Reproduced 2026‑08‑07 with `make bench` (`bench/out/scorecard.json`):

| Layer | TP | FP | TN | FN | Recall (TPR) | FPR | Precision | Youden J |
|---|---|---|---|---|---|---|---|---|
| L1 (discovery) | 26 | 0 | 0 | 0 | 1.000 | — | 1.000 | — |
| **L3 (diff)** | 400 | 0 | 400 | 0 | **1.000** | **0.000** | **1.000** | **1.000** |
| L3‑realistic | 240 | 0 | 160 | 0 | 1.000 | 0.000 | 1.000 | 1.000 |

On the gated corpus the classifier misses no real change and raises no false
alarm. **This proves the rule table is internally consistent — not that it
generalises.** §4 is the adversarial answer.

---

## 4. Is the 100% real, or overfit?

A headline of 100% on a corpus we wrote ourselves is, correctly, a red flag. We
report three things a rigged number could not survive.

### 4.1 There are no learned weights

Keyway's classifier is a fixed rule table, so the machine‑learning sense of
overfitting (memorising training points) does not apply. The *analogous* risk is
real: a corpus that only exercises the paths the rules already handle proves
nothing. The right question is therefore: **would a bug introduced into the
detector be caught?**

### 4.2 Mutation testing (the code‑side proof)

`make mutation` uses [`gremlins`](https://github.com/go-gremlins/gremlins) to
inject faults into the classifier (`internal/diff`) — flip every `<` to `<=`,
negate every boolean operator — and re‑runs the **entire suite** against each
mutant. A surviving mutant is a detector bug the tests failed to catch.

| Metric | Result |
|---|---|
| Mutants generated | 24 |
| Mutator coverage | 100% |
| Behaviour‑changing mutants killed | all |
| Survivors | 2, both provably **equivalent** |

The two survivors are the boundary flip `<`→`<=` at the clock‑skew and cache‑TTL
comparisons, each guarded by a `!=` on the line above, so the mutation cannot
change behaviour. Mutation score cannot be gamed by adding agreeable corpus rows.

### 4.3 Held‑out adversarial corpus (the data‑side proof)

A separate, **not‑gated** corpus of deliberately hard cases is scored to show what
*failure* looks like:

| Layer | TP | FP | TN | FN | Recall | FPR | Youden J |
|---|---|---|---|---|---|---|---|
| **L3‑adversarial** | 4 | 1 | 3 | 0 | 1.000 | 0.250 | **0.750** |

Keyway still misses nothing, but produces **one false positive** on the hard
negatives — **J = 0.75, not 1.0.** This is the honest number for generalisation,
and it is why the corpus is kept out of the CI gate: it is a research signal, not
a pass/fail.

### 4.4 Industry comparison, honestly framed

Published accuracy ranges for static analyzers are cited for *calibration*, not as
a head‑to‑head — the tasks differ (source‑bug detection vs. runtime‑contract
drift). We make no claim of superiority on a shared benchmark, because none
exists. See `docs/benchmark-integrity.md`.

---

## 5. Threat coverage as a denominator

Detection quality is meaningless without a denominator. Keyway measures coverage
against a **cited taxonomy of documented auth‑verifier threats** (RFC 8725, the
MCP spec, OAuth RFCs 8693/8707/9728, OWASP LLM & Agentic Top 10, CVEs, CWE,
PortSwigger) — *not* against the corpus it grades itself on. Reproduced with
`keyway threats coverage`:

| Domain | Covered | Total | % |
|---|---|---|---|
| jwt | 21 | 35 | 60% |
| agent | 6 | 15 | 40% |
| **Total** | **27** | **50** | **54%** |

Every one of the **23 gaps** is a named, cited threat Keyway does not yet detect
(e.g. JWKS‑endpoint SSRF, consent‑phishing, agent‑identity spoofing) — the roadmap
kept honest. Coverage is the metric we most want to move.

---

## 6. Limitations & threats to validity

- **Self‑authored corpus.** The 1.0 figures are on a corpus we wrote; §4.3's 0.75
  on held‑out data is the honest generalisation estimate. Independent corpora are
  welcome (`docs/independent-benchmark.md`).
- **Static discovery sees configuration, not intent.** If a verifier's true
  behaviour diverges from its config and its library defaults, only the *probe*
  layer (which requires a reachable staging endpoint) will catch it.
- **Probing needs the customer's environment.** Minting and firing attack tokens
  requires real issuers and staging traffic, so the live half is self‑hosted by
  design; the hosted Cloud layer runs only the static half and never handles
  signing keys.
- **Agent‑auth is early.** 40% coverage of a young, fast‑moving threat surface;
  treat agent findings as high‑value but non‑exhaustive.
- **Coverage counts threats, not exploitability.** A covered threat means a
  detector exists, not that every instance in the wild is caught.

---

## 7. Reproducing every number here

```bash
make bench        # §3.2 + §4.3 accuracy scorecard  → bench/out/scorecard.json
make mutation     # §4.2 mutation score (needs gremlins)
keyway threats coverage   # §5 coverage table
```

Real‑world CVE/incident validation: `docs/realworld-validation.md`. Benchmark
methodology: `docs/benchmark.md`. Integrity write‑up: `docs/benchmark-integrity.md`.

---

## 8. Conclusion

Auth correctness is a property of what is *running*, not of what is written, and
it drifts. Keyway makes the implicit token‑auth contract explicit, versioned, and
testable, and — crucially — reports its own accuracy the way a security tool
should: reproducibly, with the favourable numbers and their counter‑evidence in
the same table. The honest headline is **100% recall / 0% false alarms on a
1,226‑pair gated corpus, 0.75 Youden on held‑out hard cases, 54% of a cited
50‑threat taxonomy covered with every gap named.**

*Keyway is open source under the MIT license: <https://github.com/Keyway-AI/keyway>.*
