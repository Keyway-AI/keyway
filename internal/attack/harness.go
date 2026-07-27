package attack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Verdict is the accept/reject decision a verifier makes about a token.
type Verdict int

const (
	Reject Verdict = iota // the token was refused (the safe outcome for an attack)
	Accept                // the token was honored
)

func (v Verdict) String() string {
	if v == Accept {
		return "accept"
	}
	return "reject"
}

// Token is one generated token: the raw JWS, the threat it exercises, and
// the verdict a correct verifier MUST return. Controls carry Expect=Accept.
type Token struct {
	ThreatID  string  // maps to internal/threats (e.g. "SIG-01")
	Name      string  // stable, e.g. "alg_none"
	Rationale string  // the invariant this probes
	JWS       string  // the compact JWS
	Expect    Verdict // what a correct verifier must do
	// SelfContained is true when firing this single token at one endpoint reveals
	// a vulnerable target with no extra infrastructure. Checks that are NOT
	// self-contained (e.g. jku/x5u, which need Keyway to host an attacker JWKS)
	// are still generated and oracle-validated, but do not count toward detection
	// coverage until their callback server lands.
	SelfContained bool
}

// Finding is a token whose actual verdict differed from the correct one — i.e. an
// attack token that a verifier wrongly accepted (or a control it wrongly rejected).
type Finding struct {
	Token  Token
	Actual Verdict
}

// Vulnerability reports whether this finding is an accepted attack (the dangerous
// direction), as opposed to a rejected control (a functional problem).
func (f Finding) Vulnerability() bool {
	return f.Token.Expect == Reject && f.Actual == Accept
}

// VerifyFunc decides a single token — implemented by the reference oracle (for
// self-validation) and by the live HTTP runner (for scanning a target).
type VerifyFunc func(ctx context.Context, token string) (Verdict, error)

// Evaluate runs every token in the corpus through verify and returns the tokens
// whose actual verdict disagreed with the correct one. Against the reference
// oracle this must be empty (the corpus is valid); against a vulnerable target it
// lists exactly what that target got wrong.
func Evaluate(ctx context.Context, corpus []Token, verify VerifyFunc) ([]Finding, error) {
	var findings []Finding
	for _, t := range corpus {
		actual, err := verify(ctx, t.JWS)
		if err != nil {
			return nil, fmt.Errorf("attack: verify %s/%s: %w", t.ThreatID, t.Name, err)
		}
		if actual != t.Expect {
			findings = append(findings, Finding{Token: t, Actual: actual})
		}
	}
	return findings, nil
}

// ThreatIDs returns the distinct, sorted set of threat IDs the corpus exercises.
func ThreatIDs(corpus []Token) []string { return threatIDs(corpus, false) }

// CoveredThreatIDs returns only the threats the harness can detect end-to-end at
// a single endpoint (self-contained). This is the honest bridge to the coverage
// taxonomy: it excludes callback-dependent checks (jku/x5u) that are generated
// but not yet actionable against a live target.
func CoveredThreatIDs(corpus []Token) []string { return threatIDs(corpus, true) }

func threatIDs(corpus []Token, selfContainedOnly bool) []string {
	seen := map[string]bool{}
	for _, t := range corpus {
		if t.Expect != Reject { // controls don't map to a threat
			continue
		}
		if selfContainedOnly && !t.SelfContained {
			continue
		}
		seen[t.ThreatID] = true
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// HTTPTarget describes a live endpoint that verifies a bearer token. AcceptCodes
// are the HTTP statuses that mean "token honored"; everything else is a reject.
type HTTPTarget struct {
	URL         string
	Method      string // default GET
	Header      string // default "Authorization"
	Prefix      string // default "Bearer "
	AcceptCodes []int  // default {200,201,202,204}
	Client      *http.Client
}

// Verify sends the token to the target and maps the response status to a verdict.
// It is the live counterpart to the reference oracle, so the same corpus that is
// proven correct offline can scan a real service.
func (t HTTPTarget) Verify(ctx context.Context, token string) (Verdict, error) {
	method := t.Method
	if method == "" {
		method = http.MethodGet
	}
	headerName := t.Header
	if headerName == "" {
		headerName = "Authorization"
	}
	prefix := t.Prefix
	if prefix == "" {
		prefix = "Bearer "
	}
	accept := t.AcceptCodes
	if len(accept) == 0 {
		accept = []int{200, 201, 202, 204}
	}
	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, method, t.URL, nil)
	if err != nil {
		return Reject, err
	}
	req.Header.Set(headerName, prefix+token)
	resp, err := client.Do(req)
	if err != nil {
		return Reject, err
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
	for _, c := range accept {
		if resp.StatusCode == c {
			return Accept, nil
		}
	}
	return Reject, nil
}
