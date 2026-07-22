# Rollover — Implementation PRD v0.2

**Purpose of this document:** buildable specification. Every technology choice is made, every data structure is typed, every algorithm is specified. Where a decision is genuinely open it is marked `OPEN:` and a default is given so implementation is never blocked.

**One-line product definition:** Rollover tells you which consumers will break before you rotate a signing key, change an issuer, or drop a claim — by deriving the consumer inventory automatically and verifying it with real tokens against real endpoints.

---

## 1. Scope

### 1.1 v1 build scope

| In | Out |
|---|---|
| Self-hosted issuers where we control the private key: Keycloak, Kubernetes service accounts, generic OIDC with local keys | Auth0 / Okta / Entra canary probing (private key inaccessible — see §1.3) |
| Consumer discovery: Istio, Envoy, Kubernetes manifests, OIDC discovery, library introspection | Code-level AST extraction |
| 13 probes, staging only | Production traffic interception |
| Contract versioning, diff, classification, attribution | Enforcement, deploy blocking, auto-remediation |
| Blast radius query, grace period calculation | Authorization contract (RLS, policy engines) — v2 |
| CLI + HTTP API + JSON reports | Web UI — v1.1 |

### 1.2 Non-goals, permanently

Rollover never mutates customer configuration, never blocks a deploy, never judges whether an authorization rule is *correct*, and never requires the user to author a model file. If a feature needs the user to describe their own system, it is out of scope by definition.

### 1.3 The private-key constraint

Probes 1, 10 and 13 require signing tokens with a key controlled by the operator. SaaS IdPs hold their private keys, so those probes are unavailable there.

- **`ControlsPrivateKey == true`** (Keycloak, K8s SA, generic local) → full probe suite including canary.
- **`ControlsPrivateKey == false`** (Auth0, Okta, Entra) → degraded mode: library-defaults inference + probes 2–9, 11, 12 using a Rollover-operated shadow issuer that staging consumers additionally trust.

v1 implements full mode only. Degraded mode is v1.2.

---

## 2. Technology decisions

| Concern | Choice | Rationale |
|---|---|---|
| Language | **Go 1.22+** | Mature JOSE libraries, native Kubernetes client, single-binary runner for in-VPC deployment, good probe concurrency |
| JOSE | `github.com/go-jose/go-jose/v4` | Key generation, signing, JWKS handling, supports `alg=none` construction needed for probe 6 |
| Kubernetes | `k8s.io/client-go` | Discovery of services, Istio CRDs via dynamic client |
| CLI | `github.com/spf13/cobra` | Standard |
| HTTP server | `github.com/go-chi/chi/v5` | Minimal, stdlib-compatible |
| Storage | **PostgreSQL 15+**, `github.com/jackc/pgx/v5` | JSONB for provenance blobs, good enough graph queries at this scale |
| Migrations | `github.com/golang-migrate/migrate/v4` | Standard |
| YAML | `gopkg.in/yaml.v3` | Config parsing |
| Testing | stdlib `testing` + `github.com/stretchr/testify` | Standard |
| Container | distroless base | Runner ships into customer VPC |

**Deployment shape:** single static binary. `rollover` is the CLI; `rollover-runner` is the same binary in daemon mode. No external dependencies beyond Postgres.

---

## 3. Repository layout

```
rollover/
├── cmd/
│   ├── rollover/main.go              # CLI entrypoint
│   └── rollover-runner/main.go       # daemon entrypoint
├── internal/
│   ├── model/                        # core types, no dependencies
│   │   ├── issuer.go
│   │   ├── consumer.go
│   │   ├── contract.go
│   │   ├── probe.go
│   │   └── change.go
│   ├── issuer/                       # issuer adapters
│   │   ├── adapter.go                # interface
│   │   ├── keycloak/
│   │   ├── k8ssa/
│   │   └── generic/
│   ├── discovery/                    # consumer discovery
│   │   ├── discoverer.go             # interface
│   │   ├── istio/
│   │   ├── envoy/
│   │   ├── k8s/
│   │   └── oidcclient/
│   ├── libdefaults/                  # library behaviour database
│   │   ├── db.go
│   │   └── data/defaults.yaml
│   ├── probe/
│   │   ├── engine.go
│   │   ├── mint.go                   # token construction
│   │   └── probes.go                 # the 13 definitions
│   ├── contract/
│   │   ├── build.go                  # assemble graph from discovery
│   │   ├── hash.go                   # canonical hash
│   │   └── version.go
│   ├── diff/
│   │   ├── diff.go
│   │   └── classify.go
│   ├── attribution/
│   ├── blastradius/
│   │   ├── query.go
│   │   └── graceperiod.go
│   ├── store/
│   │   ├── store.go                  # interface
│   │   └── postgres/
│   ├── notify/
│   │   ├── slack.go
│   │   └── webhook.go
│   └── api/
│       ├── server.go
│       └── handlers.go
├── pkg/apitypes/                     # public request/response types
├── bench/                            # benchmark harness (§13)
│   ├── harness/
│   ├── mutations/
│   └── corpus/
├── migrations/
├── testdata/
└── docs/
```

---

## 4. Data model

### 4.1 Issuer

```go
package model

type IssuerType string

const (
    IssuerKeycloak    IssuerType = "keycloak"
    IssuerK8sSA       IssuerType = "k8s_sa"
    IssuerGenericOIDC IssuerType = "generic_oidc"
    IssuerAuth0       IssuerType = "auth0"
    IssuerOkta        IssuerType = "okta"
    IssuerEntra       IssuerType = "entra"
)

type KeyStatus string

const (
    // Published in JWKS but not yet used for signing. The canary state.
    KeyAnnounced KeyStatus = "announced"
    // Currently used for signing.
    KeyActive KeyStatus = "active"
    // No longer signing, still published for validation of outstanding tokens.
    KeyRetiring KeyStatus = "retiring"
    // Removed from JWKS.
    KeyRetired KeyStatus = "retired"
)

type Key struct {
    KID               string     `json:"kid"`
    Alg               string     `json:"alg"`
    Use               string     `json:"use"`
    PublicKeyPEM      string     `json:"public_key_pem"`
    Status            KeyStatus  `json:"status"`
    FirstSeenInJWKS   time.Time  `json:"first_seen_in_jwks"`
    InSigningUseSince *time.Time `json:"in_signing_use_since,omitempty"`
    RetiredAt         *time.Time `json:"retired_at,omitempty"`
}

type Issuer struct {
    ID                 string          `json:"id"`
    Name               string          `json:"name"`
    Type               IssuerType      `json:"type"`
    IssuerURL          string          `json:"issuer_url"`
    JWKSURI            string          `json:"jwks_uri"`
    DiscoveryDoc       json.RawMessage `json:"discovery_doc,omitempty"`
    Keys               []Key           `json:"keys"`
    ControlsPrivateKey bool            `json:"controls_private_key"`
    ClaimSchema        []ClaimObs      `json:"claim_schema"`
}

// ClaimObs records claims observed in tokens issued by this issuer.
type ClaimObs struct {
    Name         string    `json:"name"`
    FirstSeen    time.Time `json:"first_seen"`
    LastSeen     time.Time `json:"last_seen"`
    PresenceRate float64   `json:"presence_rate"` // 0..1 across sampled tokens
}
```

### 4.2 Consumer

```go
type ConsumerKind string

const (
    ConsumerService      ConsumerKind = "service"
    ConsumerGatewayRoute ConsumerKind = "gateway_route"
    ConsumerEdgeFunction ConsumerKind = "edge_function"
    ConsumerClient       ConsumerKind = "client" // mobile/SPA — not probeable
)

type Expectations struct {
    Issuers        []string `json:"issuers"`
    Audiences      []string `json:"audiences"`
    Algorithms     []string `json:"algorithms"`
    RequiredClaims []string `json:"required_claims"`
    ClockSkewSec   int      `json:"clock_skew_sec"`
}

type BehaviorSource string

const (
    SrcConfig         BehaviorSource = "config"
    SrcLibraryDefault BehaviorSource = "library_default"
    SrcObserved       BehaviorSource = "observed"
    SrcProbed         BehaviorSource = "probed"
)

type JWKSBehavior struct {
    CacheTTLSec           *int           `json:"cache_ttl_sec,omitempty"`
    RefreshIntervalSec    *int           `json:"refresh_interval_sec,omitempty"`
    RefreshesOnUnknownKID *bool          `json:"refreshes_on_unknown_kid,omitempty"`
    LastObservedRefresh   *time.Time     `json:"last_observed_refresh,omitempty"`
    Source                BehaviorSource `json:"source"`
}

type LibraryInfo struct {
    Name    string `json:"name"`    // e.g. "MicahParks/keyfunc"
    Version string `json:"version"` // e.g. "v1.9.0"
    Lang    string `json:"lang"`
}

type Endpoint struct {
    URL          string   `json:"url"`
    Method       string   `json:"method"`
    // A request known to succeed with a valid token. Used as the probe target.
    SafeProbePath string  `json:"safe_probe_path"`
}

type ProvenanceRecord struct {
    Source     string    `json:"source"`      // "istio:RequestAuthentication/foo"
    Locator    string    `json:"locator"`     // file path, resource ref, or URL
    ObservedAt time.Time `json:"observed_at"`
    Confidence float64   `json:"confidence"`  // 0..1
}

type Consumer struct {
    ID           string                        `json:"id"`
    StableID     string                        `json:"stable_id"`
    Kind         ConsumerKind                  `json:"kind"`
    Name         string                        `json:"name"`
    Namespace    string                        `json:"namespace,omitempty"`
    OwnerTeam    string                        `json:"owner_team,omitempty"`
    Endpoints    []Endpoint                    `json:"endpoints"`
    Expects      Expectations                  `json:"expects"`
    JWKSBehavior JWKSBehavior                  `json:"jwks_behavior"`
    Library      *LibraryInfo                  `json:"library,omitempty"`
    Provenance   map[string][]ProvenanceRecord `json:"provenance"`
    Confidence   map[string]float64            `json:"confidence"`
    Probeable    bool                          `json:"probeable"`
}
```

**`StableID` derivation** (must survive redeploy and IP change), first match wins:
1. `k8s://{cluster}/{namespace}/{service-account-name}`
2. `k8s://{cluster}/{namespace}/{service-name}`
3. `route://{gateway-name}/{route-name}`
4. `url://{normalized-host}{path-prefix}`

### 4.3 Edge and contract version

```go
type Edge struct {
    IssuerID   string       `json:"issuer_id"`
    ConsumerID string       `json:"consumer_id"`
    Expects    Expectations `json:"expects"`
    LastVerified *time.Time `json:"last_verified,omitempty"`
    VerifyState EdgeState   `json:"verify_state"`
}

type EdgeState string

const (
    EdgeVerified   EdgeState = "verified"   // all applicable probes passed
    EdgeDivergent  EdgeState = "divergent"  // probe result contradicts derived contract
    EdgeUnverified EdgeState = "unverified" // not probeable or not yet probed
)

type ContractVersion struct {
    ID          string     `json:"id"`
    Hash        string     `json:"hash"`         // sha256 of canonical form
    CreatedAt   time.Time  `json:"created_at"`
    IsBaseline  bool       `json:"is_baseline"`
    Issuers     []Issuer   `json:"issuers"`
    Consumers   []Consumer `json:"consumers"`
    Edges       []Edge     `json:"edges"`
    TriggerKind string     `json:"trigger_kind"` // scheduled|deploy|commit|manual
    TriggerRef  string     `json:"trigger_ref,omitempty"`
}
```

### 4.4 Change events

```go
type ChangeClass string

const (
    ChangeWidened ChangeClass = "widened" // now accepts what it previously rejected
    ChangeNarrowed ChangeClass = "narrowed"
    ChangeNeutral  ChangeClass = "neutral"
    ChangeUnknown  ChangeClass = "unknown"
)

type ChangeEvent struct {
    ID           string      `json:"id"`
    FromVersion  string      `json:"from_version"`
    ToVersion    string      `json:"to_version"`
    ConsumerID   string      `json:"consumer_id"`
    Field        string      `json:"field"`      // dotted path e.g. "expects.audiences"
    OldValue     any         `json:"old_value"`
    NewValue     any         `json:"new_value"`
    Class        ChangeClass `json:"class"`
    Confidence   float64     `json:"confidence"`
    Evidence     []string    `json:"evidence"`   // probe IDs or provenance locators
    Attribution  *Attribution `json:"attribution,omitempty"`
    DetectedAt   time.Time   `json:"detected_at"`
}

type Attribution struct {
    Kind      string    `json:"kind"` // commit|pr|deploy|idp_audit|unattributed
    Ref       string    `json:"ref"`
    Actor     string    `json:"actor,omitempty"`
    Team      string    `json:"team,omitempty"`
    Timestamp time.Time `json:"timestamp"`
    Confidence float64  `json:"confidence"`
}
```

---

## 5. Component interfaces

```go
// internal/issuer/adapter.go
type Adapter interface {
    Describe(ctx context.Context) (model.Issuer, error)
    // MintToken produces a signed JWT with the given claims using the named key.
    // Returns ErrNoPrivateKey if ControlsPrivateKey is false.
    MintToken(ctx context.Context, kid string, claims map[string]any) (string, error)
    // AnnounceKey publishes a new key in JWKS WITHOUT using it for signing.
    // This is the canary operation. Returns ErrUnsupported where not possible.
    AnnounceKey(ctx context.Context, alg string) (model.Key, error)
    PromoteKey(ctx context.Context, kid string) error // announced -> active
    RetireKey(ctx context.Context, kid string) error
}

// internal/discovery/discoverer.go
type Discoverer interface {
    Name() string
    Discover(ctx context.Context, scope Scope) ([]model.Consumer, error)
}

type Scope struct {
    KubeContext string
    Namespaces  []string
    ConfigPaths []string
    IssuerURLs  []string
}

// internal/store/store.go
type Store interface {
    SaveContractVersion(ctx context.Context, v model.ContractVersion) error
    GetContractVersion(ctx context.Context, id string) (model.ContractVersion, error)
    LatestVersion(ctx context.Context) (model.ContractVersion, error)
    BaselineVersion(ctx context.Context) (model.ContractVersion, error)
    SaveChangeEvents(ctx context.Context, events []model.ChangeEvent) error
    ListChangeEvents(ctx context.Context, since time.Time) ([]model.ChangeEvent, error)
    SaveProbeResults(ctx context.Context, results []model.ProbeResult) error
    ProbeHistory(ctx context.Context, consumerID string, limit int) ([]model.ProbeResult, error)
}
```

---

## 6. The probe engine

### 6.1 Probe definition

```go
type Probe struct {
    ID                 string
    Name               string
    Description        string
    RequiresPrivateKey bool
    AppliesTo          func(c model.Consumer) bool
    // Mutate returns the Authorization header value and any extra headers.
    Mutate             func(ctx MintContext) (authHeader string, extraHeaders map[string]string, err error)
    Expect             Expectation
    Severity           Severity
}

type Expectation struct {
    // AcceptStatuses is the set of HTTP status codes considered a PASS.
    AcceptStatuses []int
    // Semantic: does the contract say this token SHOULD be accepted?
    ShouldAccept bool
}

type ProbeResult struct {
    ID          string
    ProbeID     string
    ConsumerID  string
    EndpointURL string
    StatusCode  int
    LatencyMs   int
    Passed      bool
    RawResponse string // truncated to 512 bytes
    RunAt       time.Time
}
```

### 6.2 The thirteen probes — exact specification

Baseline claim set for every minted token unless overridden:

```go
claims := map[string]any{
    "iss": issuer.IssuerURL,
    "aud": consumer.Expects.Audiences[0],
    "sub": "rollover-synthetic-principal",
    "iat": now.Unix(),
    "exp": now.Add(5 * time.Minute).Unix(),
    "nbf": now.Unix(),
    "jti": uuid.NewString(),
    // plus every claim in consumer.Expects.RequiredClaims with a placeholder value
}
```

| # | ID | Mutation | Expect | Needs key |
|---|---|---|---|---|
| 1 | `valid_token` | none — baseline claims, active key | **2xx** | yes |
| 2 | `expired` | `exp = now - 3600` | 401 | yes |
| 3 | `not_yet_valid` | `nbf = now + 3600` | 401 | yes |
| 4 | `wrong_issuer` | `iss = "https://issuer.rollover.invalid"` | 401 | yes |
| 5 | `wrong_audience` | `aud = "rollover-invalid-audience"` | 401 | yes |
| 6 | `alg_none` | header `{"alg":"none","typ":"JWT"}`, empty signature segment | 401 | no |
| 7 | `alg_confusion` | header `alg=HS256`, HMAC key = issuer's RSA public key PEM bytes | 401 | no |
| 8 | `tampered_signature` | mint valid token, flip final byte of signature before base64 | 401 | yes |
| 9 | `missing_required_claim` | one sub-probe per required claim, omitting that claim | 401 or 403 | yes |
| 10 | `retired_key` | sign with a key whose `Status == KeyRetired` | 401 | yes |
| 11 | `sibling_client_token` | mint with `aud` set to a *different* known consumer's audience | 401 | yes |
| 12 | `header_bypass` | **no Authorization header**; set `X-User-Id: rollover-test`, `X-Forwarded-User: rollover-test`, `X-Authenticated-User: rollover-test` | 401 | no |
| 13 | `canary_key` | sign with the key whose `Status == KeyAnnounced` | **2xx** | yes |

Implementation notes the generator must honour:

- Probe 6 must construct the token as raw base64url segments. Most JOSE libraries refuse to emit `alg=none`; build the string manually as `base64url(header) + "." + base64url(payload) + "."`.
- Probe 7's HMAC secret is the exact PEM-encoded public key **including** header, footer and trailing newline. This is the classic mismatch; do not trim.
- Probe 9 expands to N sub-results, one per required claim. Report individually.
- Probe 12 sends no bearer token at all.
- Probe 13 is a no-op returning `ErrNoAnnouncedKey` when no key is in `announced` state.

### 6.3 Execution rules

```go
type EngineConfig struct {
    MaxConcurrentPerConsumer int           // default 2
    MaxConcurrentGlobal      int           // default 20
    RequestTimeout           time.Duration // default 10s
    InterProbeDelay          time.Duration // default 200ms
    AbortOnConsecutive5xx    int           // default 3 — stop probing that consumer
    DryRun                   bool
}
```

- **Staging guard:** refuse to run unless the target host matches an allowlist in config, or `--i-know-this-is-production` is passed. Default deny.
- **Kill switch:** the runner polls a `probe_enabled` flag before each consumer batch.
- A consumer returning 5xx on the baseline valid-token probe is marked `unverified` and skipped; a broken service is not a contract finding.

---

## 7. Discovery adapters

### 7.1 Istio (`internal/discovery/istio`)

Read `security.istio.io/v1` `RequestAuthentication` and `AuthorizationPolicy` via dynamic client.

From `RequestAuthentication.spec.jwtRules[]` extract per rule: `issuer` → `Expects.Issuers`, `audiences` → `Expects.Audiences`, `jwksUri`, `forwardOriginalToken`, and `fromHeaders`. Map `spec.selector.matchLabels` to the workloads it applies to; each becomes a Consumer of kind `service`.

Confidence: **1.0** — declarative and unambiguous.

### 7.2 Envoy (`internal/discovery/envoy`)

Parse `envoy.filters.http.jwt_authn` from static config files or the admin `/config_dump`. Extract `providers[].issuer`, `audiences`, `remote_jwks.http_uri`, `cache_duration` (→ `JWKSBehavior.CacheTTLSec`, source `config`), `forward`, `payload_in_metadata`, and `rules[].requires`.

`cache_duration` is a direct high-confidence read of JWKS behaviour. Prioritise this adapter for that reason.

Confidence: **1.0** for config-file source, **0.9** for admin dump (may be stale).

### 7.3 Kubernetes (`internal/discovery/k8s`)

Enumerate Services and their backing Deployments/StatefulSets. For each:
- Detect projected service-account token volumes (`serviceAccountToken` projection) → consumer of a K8s SA issuer, with `audience` read directly from the projection.
- Read container env vars matching `^(OIDC|JWT|AUTH|OAUTH)_` for issuer/audience hints. Confidence **0.5**, marked as hint only.
- Extract `OwnerTeam` from labels in priority order: `team`, `owner`, `app.kubernetes.io/part-of`.
- Build `Endpoint` list from Service ports plus any Ingress/HTTPRoute paths.

### 7.4 OIDC client registry (`internal/discovery/oidcclient`)

For Keycloak: `GET /admin/realms/{realm}/clients` yields registered clients with audience mappers and protocol mappers. Each client that validates tokens becomes a Consumer.

### 7.5 Library defaults (`internal/libdefaults`)

A YAML database shipped with the binary. **This is a core strategic asset — treat it as product data, not config.**

```yaml
# internal/libdefaults/data/defaults.yaml
libraries:
  - name: MicahParks/keyfunc
    lang: go
    versions:
      - constraint: ">=1.0.0 <2.0.0"
        jwks_behavior:
          refresh_interval_sec: 3600
          refreshes_on_unknown_kid: false   # documented outage cause
          cache_ttl_sec: null
        notes: "RefreshUnknownKID defaults false; unknown kid rejected without refetch"
        risk: high
  - name: auth0/node-jsonwebtoken
    lang: node
    versions:
      - constraint: ">=9.0.0"
        jwks_behavior:
          refreshes_on_unknown_kid: false
        notes: "Requires jwks-rsa; caching configured separately"
        risk: medium
  - name: spring-security-oauth2-jose
    lang: java
    versions:
      - constraint: ">=5.7.0"
        jwks_behavior:
          cache_ttl_sec: 300
          refreshes_on_unknown_kid: true
        risk: low
```

Detection sources: `go.mod`, `package.json`, `pom.xml`, `build.gradle`, `requirements.txt`, `Cargo.toml`, plus container image SBOM if present.

When probing is unavailable, this is the *only* source for `RefreshesOnUnknownKID`, which is the mechanism behind the OpenFGA outage. It must be able to produce a finding with zero probes.

---

## 8. Contract versioning

### 8.1 Canonical hash

```go
func Hash(v ContractVersion) string
```

Algorithm, exactly:
1. Build a reduced struct excluding all volatile fields: `ID`, `CreatedAt`, `LastVerified`, `LastObservedRefresh`, `FirstSeenInJWKS`, `Provenance[].ObservedAt`, all `ProbeResult`s.
2. Sort `Issuers` by `IssuerURL`; `Consumers` by `StableID`; `Edges` by `(IssuerID, ConsumerID)`; every string slice inside `Expectations` lexicographically.
3. Marshal with sorted map keys (custom encoder — `encoding/json` sorts map keys already, but structs must be field-ordered deterministically).
4. `sha256.Sum256(canonicalBytes)`, hex encode.

Two runs against an unchanged system **must** produce an identical hash. This is the first test to write.

### 8.2 Baseline flow — mandatory

```
if store.BaselineVersion() == nil:
    v.IsBaseline = true
    save(v)
    emit ZERO change events
    report "Baseline established: N consumers, M edges, K unresolved"
else:
    prev := store.LatestVersion()
    if v.Hash == prev.Hash: save(v); return   // no events
    events := diff.Compute(prev, v)
    save(v); save(events); notify(events)
```

Violating this produces a wall of findings on first run, which is the documented pilot-killer.

---

## 9. Diff and classification

### 9.1 Diff

Match consumers across versions by `StableID`. Produce per-field changes on dotted paths. A consumer present in one version and absent in the other yields `consumer_added` / `consumer_removed`.

### 9.2 Classification rules — exact table

| Field | Change | Class |
|---|---|---|
| `expects.audiences` | value added | widened |
| `expects.audiences` | value removed | narrowed |
| `expects.issuers` | added | widened |
| `expects.issuers` | removed | narrowed |
| `expects.algorithms` | added | widened |
| `expects.algorithms` | removed | narrowed |
| `expects.algorithms` | `none` added | **widened, severity critical** |
| `expects.required_claims` | added | narrowed |
| `expects.required_claims` | removed | **widened** |
| `expects.clock_skew_sec` | increased | widened |
| `expects.clock_skew_sec` | decreased | narrowed |
| `jwks_behavior.refreshes_on_unknown_kid` | true → false | **narrowed, severity high** |
| `jwks_behavior.cache_ttl_sec` | increased | narrowed (raises required grace period) |
| `consumer_added` | — | neutral |
| `consumer_removed` | — | neutral |
| any field, confidence < 0.6 | — | unknown |

`unknown` never pages. It appears in reports only.

### 9.3 Severity

```
critical — alg=none accepted; header_bypass probe passed; required claim removed
high     — refreshes_on_unknown_kid became false; issuer widened; retired key accepted
medium   — audience or algorithm widened; cache TTL increased
low      — narrowed changes, consumer added/removed
info     — unknown-class changes
```

---

## 10. Blast radius and grace period

### 10.1 Query interface

```go
type ChangeProposal struct {
    Kind string // rotate_key|remove_claim|change_issuer|drop_algorithm|retire_key
    IssuerID   string
    KID        string   // rotate_key, retire_key
    ClaimName  string   // remove_claim
    NewIssuerURL string // change_issuer
    Algorithm  string   // drop_algorithm
}

type BlastRadiusResult struct {
    Proposal        ChangeProposal
    Affected        []AffectedConsumer
    Unknown         []model.Consumer
    RecommendedGracePeriod time.Duration
    GraceBasis      string // which consumer set the bound
    GeneratedAt     time.Time
}

type AffectedConsumer struct {
    Consumer   model.Consumer
    Verdict    string  // will_break | ready | unknown
    Reason     string
    Evidence   []string
    Confidence float64
}
```

### 10.2 Resolution per proposal kind

**`rotate_key`** — for each consumer of the issuer:
1. If a `canary_key` probe result exists within 24h: pass → `ready`, fail → `will_break`, evidence = probe ID, confidence 1.0.
2. Else if `JWKSBehavior.RefreshesOnUnknownKID == false`: `will_break`, confidence 0.8, reason cites the library.
3. Else if `CacheTTLSec` known: `ready` with required wait = `CacheTTLSec`, confidence 0.7.
4. Else `unknown`, confidence 0.

**`remove_claim`** — consumer affected if `ClaimName ∈ Expects.RequiredClaims` (confidence 1.0) or observed in probe 9 sub-results as causing rejection (confidence 1.0). Consumers merely *reading* the claim without requiring it are `unknown` in v1.

**`change_issuer`** — every consumer whose `Expects.Issuers` contains the old URL and not the new one → `will_break`, confidence 1.0.

**`drop_algorithm`** — consumer affected if `Expects.Algorithms == [thatAlg]` only → `will_break`.

### 10.3 Grace period

```
For each consumer marked ready:
    window(c) =
        if measured:   time between key AnnouncedAt and first passing canary probe
        else if config: CacheTTLSec
        else if lib:    library default refresh_interval_sec
        else:           EXCLUDE from calculation, add to Unknown

RecommendedGracePeriod = max(window(c)) * 1.5, floor 1h, ceiling 30d
GraceBasis = StableID of the argmax consumer
```

If `len(Unknown) > 0` the result must state that the grace period is a lower bound only.

**Measured windows are strictly preferred.** Run the canary on a schedule (default hourly) and each run tightens the estimate. This is the compounding asset.

---

## 11. CLI surface

```
rollover init                       # write config, test connectivity
rollover issuer add --type keycloak --url ... --admin-credential-env ...
rollover discover [--namespace ...] [--output json|table]
rollover snapshot                   # build + store a contract version
rollover probe [--consumer ID] [--probe ID] [--dry-run]
rollover diff [--from VERSION] [--to VERSION]
rollover blast-radius rotate-key --issuer ID --kid KID
rollover blast-radius remove-claim --issuer ID --claim dept
rollover canary start --issuer ID --alg RS256
rollover canary status --issuer ID
rollover canary promote --issuer ID --kid KID
rollover report [--since 7d] [--format json|md]
rollover serve                      # API + scheduler
```

`blast-radius` must return in under 10 seconds and print a table plus, with `--output json`, the full `BlastRadiusResult`.

### Example output — the demo

```
$ rollover blast-radius rotate-key --issuer keycloak-prod --kid rsa-2026-01

Rotating rsa-2026-01 on keycloak-prod affects 47 consumers.

WILL BREAK (3)
  payments-api          48h JWKS cache, RefreshUnknownKID=false   [probe:canary_key #8812]
                        owner: team-payments
  legacy-reporting      no JWKS refresh configured                [lib:keyfunc v1.9.0]
                        owner: team-data
  mobile-gateway        cached key pinned in config               [istio:RequestAuthentication/mobile-gw]
                        owner: team-mobile

READY (41)   run `rollover blast-radius ... --verbose` to list
UNKNOWN (3)  insufficient evidence — not probeable

RECOMMENDED GRACE PERIOD: 9d 6h
  bound by payments-api (48h cache, measured 6d4h to pick up canary, x1.5 margin)
  NOTE: 3 consumers unknown — treat as a lower bound.
```

---

## 12. HTTP API

```
POST   /v1/snapshots                       -> {version_id, hash, is_baseline}
GET    /v1/snapshots/{id}
GET    /v1/snapshots/latest
GET    /v1/consumers                       ?issuer_id=&probeable=
GET    /v1/consumers/{stable_id}
POST   /v1/probes/run                      {consumer_ids?, probe_ids?} -> {run_id}
GET    /v1/probes/runs/{run_id}
GET    /v1/changes                         ?since=&class=&severity=
POST   /v1/blast-radius                    {proposal} -> BlastRadiusResult
POST   /v1/canary/announce                 {issuer_id, alg} -> Key
GET    /v1/canary/status                   ?issuer_id=
POST   /v1/canary/promote                  {issuer_id, kid}
GET    /v1/health
GET    /v1/coverage                        -> {total, resolved, low_confidence, unresolved}
```

Auth: bearer token from config. All write endpoints idempotent by client-supplied `Idempotency-Key`.

---

## 13. Benchmark harness

Build this **alongside** the product, not after. It is how L1 and L3 get measured, and it is the marketing artifact.

```
bench/
├── harness/
│   ├── runner.go       # executes a scenario, scores results
│   └── score.go        # TPR, FPR, precision, recall, F1, Youden
├── mutations/
│   └── mutate.go       # inject known contract changes
└── corpus/
    ├── scenarios/      # each = docker-compose + expected.json
    └── expected/
```

### 13.1 Scenario format

```yaml
# bench/corpus/scenarios/0042-audience-widened/scenario.yaml
id: "0042"
name: audience-widened
compose: docker-compose.yaml
issuer: { type: keycloak, realm: bench }
consumers:
  - stable_id: "k8s://bench/default/api-a"
    expects: { issuers: ["http://kc:8080/realms/bench"], audiences: ["api-a"], algorithms: ["RS256"] }
mutation:
  target: "k8s://bench/default/api-a"
  field: expects.audiences
  operation: add
  value: "api-b"
expected:
  detected: true
  class: widened
  severity: medium
  consumer: "k8s://bench/default/api-a"
```

### 13.2 Composition — mirror OWASP Benchmark

Roughly **50% true positives** (a real contract change was made) and **50% false positives** (a change was made that does *not* affect the contract — dependency bump, comment edit, unrelated env var, replica count, resource limits). Without the false half the FPR number is meaningless.

Target corpus for v1: 400 scenarios — 200 mutations across all classification-table rows, 200 no-op changes.

### 13.3 Scoring

```go
type Scorecard struct {
    Layer string // L1|L2|L3|L4
    TP, FP, TN, FN int
    TPR, FPR, Precision, Recall, F1, Youden float64
    PerClass map[string]ClassScore
}
```

Emit both JSON and an OWASP-style ROC chart. Youden = TPR − FPR is the headline.

### 13.4 Gate thresholds — CI fails below these

| Layer | Metric | Target | Fail below |
|---|---|---|---|
| L1 derivation recall | consumers found / actual | ≥85% | 70% |
| L2 probe accuracy | correct verdicts | ≥99% | 95% |
| L3 diff FPR | alerts on no-op commits | ≤2% | >5% |
| L3 Youden | TPR − FPR | ≥0.85 | <0.70 |
| L4 attribution | correctly bound | ≥80% | <60% |

Calibration reference: published OWASP Benchmark scores put Snyk Code at 97.18% TPR, Semgrep at 87.06%, SonarQube at 50.36%; Kiuwan reports 100% TPR at 16% FPR; the historical static-analyzer Youden ceiling was around 0.39. Our targets are higher because probing observes rather than infers.

---

## 14. Acceptance criteria for v1

1. `rollover snapshot` run twice against an unchanged system produces **identical hashes**.
2. First snapshot emits **zero** change events and reports a baseline.
3. Against the reference Keycloak + Istio + 5-service stack in `bench/corpus/scenarios/reference`, discovery finds ≥85% of consumers with no configuration file written.
4. All 13 probes execute; probe 12 (`header_bypass`) correctly flags a deliberately misconfigured service that trusts `X-User-Id`.
5. `rollover canary start` announces a key without it being used to sign; probe 13 correctly separates ready from not-ready consumers.
6. A consumer configured with `keyfunc v1.9.0` defaults is reported as `will_break` for `rotate_key` **without any probe**, from library defaults alone.
7. Editing an Istio `RequestAuthentication` to add an audience produces exactly **one** change event, class `widened`, attributed to the commit.
8. Bumping an unrelated dependency and redeploying produces **zero** change events.
9. `blast-radius rotate-key` returns in <10s on a 50-consumer graph with a grace period and a named bounding consumer.
10. Benchmark harness runs the 400-scenario corpus and emits a scorecard meeting §13.4 targets.

---

## 15. Build order

| Milestone | Deliverable | Gate |
|---|---|---|
| **M1** | `model`, `store/postgres`, migrations, `contract/hash` | AC-1 |
| **M2** | `issuer/keycloak` (describe + mint), `discovery/istio`, `discovery/k8s` | AC-3 |
| **M3** | `probe` engine, probes 1–12, CLI `probe` | AC-4 |
| **M4** | `contract/build`, snapshot, baseline flow, `diff` + `classify` | AC-1, 2, 7, 8 |
| **M5** | `libdefaults` DB + detection | AC-6 |
| **M6** | `AnnounceKey`, probe 13, `canary` CLI | AC-5 |
| **M7** | `blastradius` + grace period, CLI + API | AC-9 |
| **M8** | `bench/harness`, 400-scenario corpus, scorecard | AC-10 |
| **M9** | `attribution` (git + K8s deploy), `notify/slack`, `api/server` | full |

M1–M4 is the smallest thing that produces value: a versioned, diffable contract with zero-noise alerting. M6–M7 is the demo that sells.

---

## 16. Open decisions

`OPEN-1` **Mobile and SPA clients are not probeable.** v1 marks them `Probeable: false` and excludes them from grace-period calculation while listing them under `Unknown`. If mobile is commonly the slowest verifier this leaves a hole. *Default: exclude and flag loudly.*

`OPEN-2` **Multi-tenant issuer support.** v1 assumes one realm per issuer record. *Default: model tenants as separate Issuers.*

`OPEN-3` **Probe 9 combinatorics.** With 12 required claims this is 12 requests per consumer. *Default: cap at 8 claims, prioritise claims appearing in authorization decisions.*

`OPEN-4` **Storage of minted tokens.** Never persist. Log only `jti` and probe ID. *Default: no token bodies at rest or in logs.*

`OPEN-5` **Attribution when IdP changes originate outside git.** Keycloak admin events are available via the admin events API; Okta and Entra need System Log polling. *Default: v1 covers git and Keycloak admin events; others `unattributed`.*
