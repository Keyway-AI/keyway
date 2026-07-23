package probe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/architsharma/keyway/internal/model"
	"github.com/google/uuid"
)

// EngineConfig controls probe execution (PRD §6.3).
type EngineConfig struct {
	MaxConcurrentPerConsumer int           // reserved; probes run sequentially per consumer
	MaxConcurrentGlobal      int           // default 20 — consumers probed in parallel
	RequestTimeout           time.Duration // default 10s
	InterProbeDelay          time.Duration // default 200ms
	AbortOnConsecutive5xx    int           // default 3
	DryRun                   bool

	// Allowlist of host substrings the engine may target. Empty = deny all
	// unless AllowProduction is set (the --i-know-this-is-production override).
	Allowlist       []string
	AllowProduction bool

	// KillSwitch is polled before each consumer; returning true stops probing.
	KillSwitch func() bool

	// MaxRequiredClaimProbes caps probe-9 sub-probes (PRD OPEN-3, default 8).
	MaxRequiredClaimProbes int
}

// DefaultEngineConfig returns the PRD §6.3 defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxConcurrentPerConsumer: 2,
		MaxConcurrentGlobal:      20,
		RequestTimeout:           10 * time.Second,
		InterProbeDelay:          200 * time.Millisecond,
		AbortOnConsecutive5xx:    3,
		MaxRequiredClaimProbes:   8,
	}
}

// HostAllowed reports whether the engine may target a host (staging guard).
// Default deny: an empty allowlist without the production override blocks all.
func (c EngineConfig) HostAllowed(host string) bool {
	if c.AllowProduction {
		return true
	}
	for _, allowed := range c.Allowlist {
		if allowed != "" && strings.Contains(host, allowed) {
			return true
		}
	}
	return false
}

// Engine runs probes against consumers.
type Engine struct {
	cfg    EngineConfig
	probes []Probe
	client *http.Client
	now    func() time.Time
}

// NewEngine constructs an engine with the default probe set.
func NewEngine(cfg EngineConfig) *Engine {
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if cfg.MaxConcurrentGlobal <= 0 {
		cfg.MaxConcurrentGlobal = 20
	}
	if cfg.MaxRequiredClaimProbes <= 0 {
		cfg.MaxRequiredClaimProbes = 8
	}
	return &Engine{
		cfg:    cfg,
		probes: Definitions(),
		client: &http.Client{Timeout: timeout},
		now:    time.Now,
	}
}

// WithClock injects a clock (tests).
func (e *Engine) WithClock(now func() time.Time) *Engine { e.now = now; return e }

// ConsumerOutcome summarises what happened for one consumer.
type ConsumerOutcome struct {
	ConsumerID string
	Verified   bool // baseline valid-token probe passed
	Skipped    bool // not probeable, host not allowed, or kill switch
	Reason     string
	Results    []model.ProbeResult
}

// Run probes the applicable consumers and returns all probe results plus a
// per-consumer outcome summary. iss provides the JWKS key ids; mint signs
// tokens (the issuer adapter's MintToken).
func (e *Engine) Run(ctx context.Context, iss model.Issuer, mint MintFunc, consumers []model.Consumer) ([]model.ProbeResult, []ConsumerOutcome, error) {
	active, announced, retired, activePEM := resolveKeys(iss)

	var (
		mu       sync.Mutex
		results  []model.ProbeResult
		outcomes []ConsumerOutcome
		wg       sync.WaitGroup
		sem      = make(chan struct{}, e.cfg.MaxConcurrentGlobal)
	)

	for _, c := range consumers {
		if e.cfg.KillSwitch != nil && e.cfg.KillSwitch() {
			mu.Lock()
			outcomes = append(outcomes, ConsumerOutcome{ConsumerID: c.StableID, Skipped: true, Reason: "kill switch engaged"})
			mu.Unlock()
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(c model.Consumer) {
			defer wg.Done()
			defer func() { <-sem }()

			out := e.runConsumer(ctx, iss, mint, c, consumers, keyIDs{active, announced, retired, activePEM})
			mu.Lock()
			results = append(results, out.Results...)
			outcomes = append(outcomes, out)
			mu.Unlock()
		}(c)
	}
	wg.Wait()
	return results, outcomes, ctx.Err()
}

type keyIDs struct {
	active, announced, retired, activePEM string
}

func (e *Engine) runConsumer(ctx context.Context, iss model.Issuer, mint MintFunc, c model.Consumer, all []model.Consumer, kids keyIDs) ConsumerOutcome {
	out := ConsumerOutcome{ConsumerID: c.StableID}
	if !c.Probeable || len(c.Endpoints) == 0 {
		out.Skipped, out.Reason = true, "not probeable (no endpoint)"
		return out
	}
	endpoint := c.Endpoints[0]
	host := hostOf(endpoint.URL)
	if !e.cfg.HostAllowed(host) {
		out.Skipped, out.Reason = true, "host not in staging allowlist (default deny)"
		return out
	}

	sibling := siblingAudience(c, all)

	// Baseline valid-token probe first: a 5xx here means the service is broken,
	// which is not a contract finding — mark unverified and skip the rest.
	baseline := e.probeByID(ProbeValidToken)
	baseCtx := e.mintContext(iss, c, mint, sibling, kids)
	baseRes, status := e.execute(ctx, *baseline, baseCtx, endpoint)
	out.Results = append(out.Results, baseRes)
	if status >= 500 {
		out.Skipped, out.Reason = true, "baseline valid-token returned 5xx (service broken, not a finding)"
		return out
	}
	out.Verified = baseRes.Passed

	consecutive5xx := 0
	for _, p := range e.probes {
		if p.ID == ProbeValidToken {
			continue // already run
		}
		if p.AppliesTo != nil && !p.AppliesTo(c) {
			continue
		}
		select {
		case <-ctx.Done():
			return out
		default:
		}
		if e.cfg.InterProbeDelay > 0 {
			time.Sleep(e.cfg.InterProbeDelay)
		}

		if p.ID == ProbeMissingClaim {
			claims := c.Expects.RequiredClaims
			if len(claims) > e.cfg.MaxRequiredClaimProbes {
				claims = claims[:e.cfg.MaxRequiredClaimProbes]
			}
			for _, claim := range claims {
				mc := e.mintContext(iss, c, mint, sibling, kids)
				mc.OmitClaim = claim
				res, st := e.execute(ctx, p, mc, endpoint)
				res.ProbeID = p.ID + ":" + claim
				out.Results = append(out.Results, res)
				consecutive5xx = trackAndMaybeAbort(st, consecutive5xx, e.cfg.AbortOnConsecutive5xx, &out)
				if out.Reason != "" {
					return out
				}
			}
			continue
		}

		mc := e.mintContext(iss, c, mint, sibling, kids)
		res, st := e.execute(ctx, p, mc, endpoint)
		if res.ProbeID == "" && res.StatusCode == 0 && res.RawResponse == skipMarker {
			continue // not applicable (no canary/retired/sibling)
		}
		out.Results = append(out.Results, res)
		consecutive5xx = trackAndMaybeAbort(st, consecutive5xx, e.cfg.AbortOnConsecutive5xx, &out)
		if out.Reason != "" {
			return out
		}
	}
	return out
}

const skipMarker = "\x00skip"

func trackAndMaybeAbort(status, consecutive, limit int, out *ConsumerOutcome) int {
	if status >= 500 {
		consecutive++
		if limit > 0 && consecutive >= limit {
			out.Reason = "aborted after consecutive 5xx responses"
		}
		return consecutive
	}
	return 0
}

func (e *Engine) mintContext(iss model.Issuer, c model.Consumer, mint MintFunc, sibling string, kids keyIDs) MintContext {
	now := e.now()
	return MintContext{
		Issuer:          iss,
		Consumer:        c,
		Claims:          BaselineClaims(iss, c, now),
		Now:             now,
		Mint:            mint,
		ActiveKID:       kids.active,
		AnnouncedKID:    kids.announced,
		RetiredKID:      kids.retired,
		ActiveKeyPEM:    kids.activePEM,
		SiblingAudience: sibling,
	}
}

// execute runs one probe against one endpoint and returns the result plus the
// HTTP status (0 on transport error, -1 when skipped/not-applicable).
func (e *Engine) execute(ctx context.Context, p Probe, mc MintContext, ep model.Endpoint) (model.ProbeResult, int) {
	res := model.ProbeResult{
		ID:          uuid.NewString(),
		ProbeID:     p.ID,
		ConsumerID:  mc.Consumer.StableID,
		EndpointURL: ep.URL,
		RunAt:       e.now(),
	}
	authHeader, extra, err := p.Mutate(mc)
	if errors.Is(err, ErrProbeNotApplicable) {
		return model.ProbeResult{RawResponse: skipMarker}, -1
	}
	if err != nil {
		res.RawResponse = truncate("mint error: "+err.Error(), 512)
		return res, 0
	}
	if e.cfg.DryRun {
		res.RawResponse = "dry-run"
		res.Passed = false
		return res, -1
	}

	method := ep.Method
	if method == "" {
		method = http.MethodGet
	}
	reqURL := strings.TrimRight(ep.URL, "/") + ep.SafeProbePath
	reqCtx, cancel := context.WithTimeout(ctx, e.client.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, method, reqURL, nil)
	if err != nil {
		res.RawResponse = truncate("request error: "+err.Error(), 512)
		return res, 0
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}

	start := e.now()
	resp, err := e.client.Do(req)
	res.LatencyMs = int(e.now().Sub(start).Milliseconds())
	if err != nil {
		res.RawResponse = truncate("transport error: "+err.Error(), 512)
		return res, 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	res.StatusCode = resp.StatusCode
	res.RawResponse = truncate(string(body), 512)
	res.Passed = p.Expect.Accepts(resp.StatusCode)
	return res, resp.StatusCode
}

func (e *Engine) probeByID(id string) *Probe {
	for i := range e.probes {
		if e.probes[i].ID == id {
			return &e.probes[i]
		}
	}
	return nil
}

// resolveKeys extracts the active/announced/retired kids and active PEM.
func resolveKeys(iss model.Issuer) (active, announced, retired, activePEM string) {
	for _, k := range iss.Keys {
		switch k.Status {
		case model.KeyActive:
			if active == "" {
				active, activePEM = k.KID, k.PublicKeyPEM
			}
		case model.KeyAnnounced:
			if announced == "" {
				announced = k.KID
			}
		case model.KeyRetired:
			if retired == "" {
				retired = k.KID
			}
		}
	}
	return active, announced, retired, activePEM
}

func siblingAudience(c model.Consumer, all []model.Consumer) string {
	own := map[string]struct{}{}
	for _, a := range c.Expects.Audiences {
		own[a] = struct{}{}
	}
	for _, other := range all {
		if other.StableID == c.StableID {
			continue
		}
		for _, a := range other.Expects.Audiences {
			if _, mine := own[a]; !mine && a != "" {
				return a
			}
		}
	}
	return ""
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
