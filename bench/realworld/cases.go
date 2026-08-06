package realworld

import (
	"context"
	"fmt"
	"net/http/httptest"
	"time"

	"github.com/Keyway-AI/keyway/internal/blastradius"
	"github.com/Keyway-AI/keyway/internal/issuer/generic"
	"github.com/Keyway-AI/keyway/internal/libdefaults"
	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/probe"
)

// Result is the outcome of validating one real-world case against Keyway.
type Result struct {
	Detected bool
	Verdict  string // human-readable statement of what Keyway concluded
}

// Case is a documented, cited real-world JWT/JWKS risk plus a reproduction that
// checks whether Keyway detects it.
type Case struct {
	ID        string // e.g. "RW-01"
	Title     string
	Category  string
	Reference string // CVE id or "org/repo#issue"
	Source    string // canonical URL
	Mechanism string // which Keyway capability catches it
	Severity  string
	Run       func() Result
}

// Cases returns the validation suite. Each Run reproduces the exact failure mode
// from its cited source and asks Keyway to flag it.
func Cases() []Case {
	return []Case{
		{
			ID: "RW-01", Title: "alg=none signature bypass",
			Category: "algorithm downgrade", Reference: "CVE-2022-23540",
			Source:    "https://nvd.nist.gov/vuln/detail/CVE-2022-23540",
			Mechanism: "probe: alg_none", Severity: "critical",
			Run: probeCase(vuln{acceptNone: true}, probe.ProbeAlgNone),
		},
		{
			ID: "RW-02", Title: "RS256→HS256 algorithm confusion (public key as HMAC secret)",
			Category: "algorithm confusion", Reference: "CVE-2022-23541",
			Source:    "https://nvd.nist.gov/vuln/detail/CVE-2022-23541",
			Mechanism: "probe: alg_confusion", Severity: "critical",
			Run: probeCase(vuln{acceptHSConfuse: true}, probe.ProbeAlgConfusion),
		},
		{
			ID: "RW-03", Title: "Signature not verified (claims trusted unsigned)",
			Category: "missing verification", Reference: "CWE-347",
			Source:    "https://cwe.mitre.org/data/definitions/347.html",
			Mechanism: "probe: tampered_signature", Severity: "high",
			Run: probeCase(vuln{skipSignature: true}, probe.ProbeTamperedSignature),
		},
		{
			ID: "RW-04", Title: "Missing audience validation",
			Category: "missing claim check", Reference: "OWASP JWT / RFC 8725 §3.9",
			Source:    "https://datatracker.ietf.org/doc/html/rfc8725",
			Mechanism: "probe: wrong_audience", Severity: "high",
			Run: probeCase(vuln{skipAudience: true}, probe.ProbeWrongAudience),
		},
		{
			ID: "RW-05", Title: "Missing issuer validation",
			Category: "missing claim check", Reference: "RFC 8725 §3.8",
			Source:    "https://datatracker.ietf.org/doc/html/rfc8725",
			Mechanism: "probe: wrong_issuer", Severity: "high",
			Run: probeCase(vuln{skipIssuer: true}, probe.ProbeWrongIssuer),
		},
		{
			ID: "RW-06", Title: "Expired token accepted",
			Category: "missing claim check", Reference: "RFC 8725 §3.9",
			Source:    "https://datatracker.ietf.org/doc/html/rfc8725",
			Mechanism: "probe: expired", Severity: "high",
			Run: probeCase(vuln{skipExpiry: true}, probe.ProbeExpired),
		},
		{
			ID: "RW-07", Title: "Trusts identity header without a token (gateway bypass)",
			Category: "auth bypass", Reference: "CWE-290",
			Source:    "https://cwe.mitre.org/data/definitions/290.html",
			Mechanism: "probe: header_bypass", Severity: "critical",
			Run: probeCase(vuln{trustHeader: true}, probe.ProbeHeaderBypass),
		},
		{
			ID: "RW-08", Title: "JWKS not refreshed on unknown kid → key-rotation outage",
			Category: "key rotation", Reference: "openfga/openfga#3099",
			Source:    "https://github.com/openfga/openfga/issues/3099",
			Mechanism: "libdefaults + blast-radius: rotate_key", Severity: "high",
			Run: rotationCase(),
		},
	}
}

// probeCase reproduces a token-level vulnerability and checks that the named
// probe flags it (a finding = the probe did NOT pass because the vulnerable
// server accepted what it should have rejected).
func probeCase(v vuln, probeID string) func() Result {
	return func() Result {
		iss, err := generic.New(generic.Options{Name: "issuer", IssuerURL: wantIssuer})
		if err != nil {
			return Result{Detected: false, Verdict: "setup error: " + err.Error()}
		}
		srv := httptest.NewServer(vulnServer(iss, v))
		defer srv.Close()

		c := realworldConsumer(srv.URL)
		res, ok := runProbe(iss, c, probeID)
		if !ok {
			return Result{Detected: false, Verdict: fmt.Sprintf("probe %s did not run", probeID)}
		}
		detected := !res.Passed // vulnerable server accepted the attack; probe fired
		return Result{
			Detected: detected,
			Verdict: fmt.Sprintf("probe %s: reproduced server responded %d to the attack token; Keyway %s",
				probeID, res.StatusCode, flaggedWord(detected)),
		}
	}
}

func rotationCase() func() Result {
	return func() Result {
		db, err := libdefaults.Load()
		if err != nil {
			return Result{Detected: false, Verdict: "libdefaults load error"}
		}
		// Reproduce OpenFGA's config: a consumer using MicahParks/keyfunc v1.x,
		// whose default RefreshUnknownKID=false is the documented root cause.
		c := model.Consumer{
			ID: "openfga", StableID: "openfga", Name: "openfga",
			Expects: model.Expectations{Issuers: []string{wantIssuer}, Audiences: []string{wantAudience}},
			Library: &model.LibraryInfo{Name: "MicahParks/keyfunc", Version: "v1.9.0", Lang: "go"},
		}
		db.Enrich(&c) // fills RefreshesOnUnknownKID=false from the library defaults

		v := model.ContractVersion{
			Issuers:   []model.Issuer{{ID: "iss", Name: "iss", IssuerURL: wantIssuer}},
			Consumers: []model.Consumer{c},
		}
		res, err := blastradius.Resolve(v,
			blastradius.ChangeProposal{Kind: blastradius.KindRotateKey, IssuerID: "iss", KID: "rsa-new"},
			nil, time.Now())
		if err != nil || len(res.Affected) != 1 {
			return Result{Detected: false, Verdict: "blast-radius produced no verdict"}
		}
		a := res.Affected[0]
		detected := a.Verdict == blastradius.VerdictWillBreak
		return Result{
			Detected: detected,
			Verdict:  fmt.Sprintf("blast-radius rotate_key → %s (%s) %v", a.Verdict, a.Reason, a.Evidence),
		}
	}
}

func realworldConsumer(url string) model.Consumer {
	return model.Consumer{
		ID: "svc", StableID: "k8s://rw/default/svc", Name: "svc", Kind: model.ConsumerService,
		Endpoints: []model.Endpoint{{URL: url, Method: "GET", SafeProbePath: "/"}},
		Expects: model.Expectations{
			Issuers:    []string{wantIssuer},
			Audiences:  []string{wantAudience},
			Algorithms: []string{"RS256"},
		},
		Probeable: true,
	}
}

func runProbe(iss *generic.Adapter, c model.Consumer, probeID string) (model.ProbeResult, bool) {
	eng := probe.NewEngine(probe.EngineConfig{
		Allowlist:      []string{"127.0.0.1"},
		RequestTimeout: 5 * time.Second,
	})
	ctx := context.Background()
	im, err := iss.Describe(ctx)
	if err != nil {
		return model.ProbeResult{}, false
	}
	mint := func(kid string, claims map[string]any) (string, error) {
		return iss.MintToken(ctx, kid, claims)
	}
	results, _, err := eng.Run(ctx, im, mint, []model.Consumer{c})
	if err != nil {
		return model.ProbeResult{}, false
	}
	for _, r := range results {
		if r.ProbeID == probeID {
			return r, true
		}
	}
	return model.ProbeResult{}, false
}

func flaggedWord(detected bool) string {
	if detected {
		return "FLAGGED it ✓"
	}
	return "missed it ✗"
}
