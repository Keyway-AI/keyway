package probe

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Keyway-AI/keyway/internal/attack"
	"github.com/Keyway-AI/keyway/internal/model"
)

// mintSigner adapts the issuer's MintFunc + active public key to the harness's
// TrustedSigner seam: claim-level attacks are signed by the *real* issuer, so a
// live target's signature check passes and its claim validation is what's tested.
type mintSigner struct {
	mint   MintFunc
	kid    string
	pubPEM string
}

func (m mintSigner) Sign(claims map[string]any) (string, error) { return m.mint(m.kid, claims) }
func (m mintSigner) PublicKeyPEM() string                       { return m.pubPEM }

// RunHarness fires the generative attack corpus (internal/attack) at each
// applicable consumer's endpoint. It reuses the same safety machinery as Run —
// the staging allowlist, response scrubbing, and dry-run — and records one
// ProbeResult per attack token, keyed "harness:<threat>:<name>". A result with
// Passed=false whose token expected rejection is a live vulnerability.
func (e *Engine) RunHarness(ctx context.Context, iss model.Issuer, mint MintFunc, consumers []model.Consumer) ([]model.ProbeResult, []ConsumerOutcome, error) {
	active, _, _, activePEM := resolveKeys(iss)

	var results []model.ProbeResult
	var outcomes []ConsumerOutcome

	for _, c := range consumers {
		if e.cfg.KillSwitch != nil && e.cfg.KillSwitch() {
			outcomes = append(outcomes, ConsumerOutcome{ConsumerID: c.StableID, Skipped: true, Reason: "kill switch engaged"})
			continue
		}
		out := e.runHarnessConsumer(ctx, iss, mintSigner{mint, active, activePEM}, c)
		results = append(results, out.Results...)
		outcomes = append(outcomes, out)
		if ctx.Err() != nil {
			break
		}
	}
	return results, outcomes, ctx.Err()
}

func (e *Engine) runHarnessConsumer(ctx context.Context, iss model.Issuer, signer attack.TrustedSigner, c model.Consumer) ConsumerOutcome {
	out := ConsumerOutcome{ConsumerID: c.StableID}
	if !c.Probeable || len(c.Endpoints) == 0 {
		out.Skipped, out.Reason = true, "not probeable (no endpoint)"
		return out
	}
	endpoint := c.Endpoints[0]
	if !e.cfg.HostAllowed(hostOf(endpoint.URL)) {
		out.Skipped, out.Reason = true, "host not in staging allowlist (default deny)"
		return out
	}

	gc, err := attack.NewLiveContext(signer, expectedIssuer(iss, c), expectedAudience(c), c.Expects.RequiredClaims, e.now())
	if err != nil {
		out.Skipped, out.Reason = true, "harness setup failed: "+err.Error()
		return out
	}
	corpus, err := attack.Corpus(gc)
	if err != nil {
		// Generation errors are recorded but the successfully-built tokens still run.
		out.Reason = "partial corpus: " + err.Error()
	}

	for _, tok := range corpus {
		select {
		case <-ctx.Done():
			return out
		default:
		}
		if e.cfg.InterProbeDelay > 0 {
			time.Sleep(e.cfg.InterProbeDelay)
		}
		res := e.sendHarnessToken(ctx, c, endpoint, tok)
		out.Results = append(out.Results, res)
		if tok.ThreatID == "CONTROL" {
			out.Verified = res.Passed // the control being accepted means the target trusts our issuer
		}
	}
	return out
}

// sendHarnessToken sends one attack token and scores it: Passed means the target
// did the correct thing (rejected an attack, accepted the control).
func (e *Engine) sendHarnessToken(ctx context.Context, c model.Consumer, ep model.Endpoint, tok attack.Token) model.ProbeResult {
	res := model.ProbeResult{
		ID:          uuid.NewString(),
		ProbeID:     "harness:" + tok.ThreatID + ":" + tok.Name,
		ConsumerID:  c.StableID,
		EndpointURL: ep.URL,
		RunAt:       e.now(),
	}
	if e.cfg.DryRun {
		res.RawResponse = "dry-run"
		return res
	}
	status, body := e.sendToken(ctx, ep, tok.JWS)
	res.StatusCode = status
	res.RawResponse = truncate(scrubTokens(body))
	accepted := status == 200 || status == 201 || status == 202 || status == 204
	actual := attack.Reject
	if accepted {
		actual = attack.Accept
	}
	res.Passed = actual == tok.Expect
	return res
}

// sendToken issues a single bearer-token request and returns the status (0 on
// transport error) and the response body.
func (e *Engine) sendToken(ctx context.Context, ep model.Endpoint, token string) (int, string) {
	method := ep.Method
	if method == "" {
		method = http.MethodGet
	}
	reqURL := strings.TrimRight(ep.URL, "/") + ep.SafeProbePath
	reqCtx, cancel := context.WithTimeout(ctx, e.client.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, method, reqURL, nil)
	if err != nil {
		return 0, "request error: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := e.client.Do(req)
	if err != nil {
		return 0, "transport error: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return resp.StatusCode, string(body)
}

func expectedAudience(c model.Consumer) string {
	if len(c.Expects.Audiences) > 0 {
		return c.Expects.Audiences[0]
	}
	return ""
}

func expectedIssuer(iss model.Issuer, c model.Consumer) string {
	if len(c.Expects.Issuers) > 0 {
		return c.Expects.Issuers[0]
	}
	return iss.IssuerURL
}
