// Package blastradius answers "if I make this change, who breaks?" and derives
// a safe grace period (PRD §10).
package blastradius

import (
	"fmt"
	"sort"
	"time"

	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/probe"
)

// Proposal kinds.
const (
	KindRotateKey     = "rotate_key"
	KindRemoveClaim   = "remove_claim"
	KindChangeIssuer  = "change_issuer"
	KindDropAlgorithm = "drop_algorithm"
	KindRetireKey     = "retire_key"
)

// Verdicts.
const (
	VerdictWillBreak = "will_break"
	VerdictReady     = "ready"
	VerdictUnknown   = "unknown"
)

// ChangeProposal is a proposed change to evaluate (PRD §10.1).
type ChangeProposal struct {
	Kind         string
	IssuerID     string
	KID          string // rotate_key, retire_key
	ClaimName    string // remove_claim
	NewIssuerURL string // change_issuer
	Algorithm    string // drop_algorithm
}

// AffectedConsumer is one consumer's verdict under a proposal.
type AffectedConsumer struct {
	Consumer   model.Consumer `json:"consumer"`
	Verdict    string         `json:"verdict"`
	Reason     string         `json:"reason"`
	Evidence   []string       `json:"evidence"`
	Confidence float64        `json:"confidence"`
}

// BlastRadiusResult is the full answer (PRD §10.1).
type BlastRadiusResult struct {
	Proposal               ChangeProposal     `json:"proposal"`
	Affected               []AffectedConsumer `json:"affected"`
	Unknown                []model.Consumer   `json:"unknown"`
	RecommendedGracePeriod time.Duration      `json:"recommended_grace_period"`
	GraceBasis             string             `json:"grace_basis"`
	GeneratedAt            time.Time          `json:"generated_at"`
}

const canaryFreshness = 24 * time.Hour

// Resolve computes the blast radius of a proposal against a contract version and
// its probe history (keyed by consumer StableID). `now` anchors canary
// freshness and the generated-at timestamp.
func Resolve(v model.ContractVersion, p ChangeProposal, history map[string][]model.ProbeResult, now time.Time) (BlastRadiusResult, error) {
	res := BlastRadiusResult{Proposal: p, GeneratedAt: now}

	var (
		affected []AffectedConsumer
		unknown  []model.Consumer
		windows  []windowed
	)

	switch p.Kind {
	case KindRotateKey, KindRetireKey:
		affected, unknown, windows = resolveRotateKey(v, p, history, now)
	case KindRemoveClaim:
		affected, unknown = resolveRemoveClaim(v, p, history)
	case KindChangeIssuer:
		affected, unknown = resolveChangeIssuer(v, p)
	case KindDropAlgorithm:
		affected, unknown = resolveDropAlgorithm(v, p)
	default:
		return res, fmt.Errorf("blastradius: unknown proposal kind %q", p.Kind)
	}

	res.Affected = affected
	res.Unknown = unknown

	// Grace period from the ready consumers' refresh windows (PRD §10.3). A
	// window measured from a real announce→pickup (KI-25) is preferred to a
	// default one at the same duration, and the basis is labelled accordingly.
	if len(windows) > 0 {
		durations := make([]time.Duration, 0, len(windows))
		var maxW time.Duration
		var maxMeasured bool
		for _, w := range windows {
			durations = append(durations, w.window)
			if w.window > maxW || (w.window == maxW && w.measured && !maxMeasured) {
				maxW = w.window
				maxMeasured = w.measured
				if w.measured {
					res.GraceBasis = "measured pickup on " + w.stableID
				} else {
					res.GraceBasis = w.stableID
				}
			}
		}
		res.RecommendedGracePeriod = GracePeriod(durations)
	}
	return res, nil
}

type windowed struct {
	stableID string
	window   time.Duration
	measured bool // true when the window came from an observed announce→pickup, not a default
}

// issuerURL resolves a proposal's IssuerID to an issuer URL via the contract,
// falling back to treating the ID as the URL itself.
func issuerURL(v model.ContractVersion, issuerID string) string {
	for _, is := range v.Issuers {
		if is.ID == issuerID || is.Name == issuerID {
			return is.IssuerURL
		}
	}
	return issuerID
}

func resolveRotateKey(v model.ContractVersion, p ChangeProposal, history map[string][]model.ProbeResult, now time.Time) ([]AffectedConsumer, []model.Consumer, []windowed) {
	url := issuerURL(v, p.IssuerID)
	var affected []AffectedConsumer
	var unknown []model.Consumer
	var windows []windowed

	for _, c := range v.Consumers {
		if !usesIssuer(c, url) {
			continue
		}
		ac := AffectedConsumer{Consumer: c}

		// 1. Fresh canary probe result is the strongest evidence.
		if pr, ok := freshCanary(history[c.StableID], now); ok {
			if pr.Passed {
				ac.Verdict, ac.Confidence = VerdictReady, 1.0
				ac.Reason = "canary key accepted"
				ac.Evidence = []string{"probe:" + pr.ProbeID}
				// Prefer a window measured from the observed announce→pickup over
				// the cache-TTL/library default (KI-25).
				if mw, ok := measuredWindow(history[c.StableID], now); ok {
					windows = append(windows, windowed{c.StableID, mw, true})
					ac.Reason = fmt.Sprintf("canary key accepted (measured pickup %s)", mw.Round(time.Second))
				} else if w, ok := readyWindow(c); ok {
					windows = append(windows, windowed{c.StableID, w, false})
				}
			} else {
				ac.Verdict, ac.Confidence = VerdictWillBreak, 1.0
				ac.Reason = "canary key rejected"
				ac.Evidence = []string{"probe:" + pr.ProbeID}
			}
			affected = append(affected, ac)
			continue
		}

		// 2. Library/config says it will not refresh on an unknown kid.
		if c.JWKSBehavior.RefreshesOnUnknownKID != nil && !*c.JWKSBehavior.RefreshesOnUnknownKID {
			ac.Verdict, ac.Confidence = VerdictWillBreak, 0.8
			ac.Reason = "does not refresh JWKS on unknown kid"
			ac.Evidence = libEvidence(c)
			affected = append(affected, ac)
			continue
		}

		// 3. Cache TTL known -> ready after the cache expires.
		if c.JWKSBehavior.CacheTTLSec != nil {
			ac.Verdict, ac.Confidence = VerdictReady, 0.7
			ac.Reason = fmt.Sprintf("refreshes JWKS; %ds cache", *c.JWKSBehavior.CacheTTLSec)
			ac.Evidence = []string{"source:" + string(c.JWKSBehavior.Source)}
			affected = append(affected, ac)
			windows = append(windows, windowed{c.StableID, time.Duration(*c.JWKSBehavior.CacheTTLSec) * time.Second, false})
			continue
		}

		// 4. Insufficient evidence.
		ac.Verdict, ac.Confidence = VerdictUnknown, 0
		ac.Reason = "insufficient evidence — not probeable"
		affected = append(affected, ac)
		unknown = append(unknown, c)
	}
	return affected, unknown, windows
}

func resolveRemoveClaim(v model.ContractVersion, p ChangeProposal, history map[string][]model.ProbeResult) ([]AffectedConsumer, []model.Consumer) {
	var affected []AffectedConsumer
	var unknown []model.Consumer
	for _, c := range v.Consumers {
		if containsStr(c.Expects.RequiredClaims, p.ClaimName) {
			affected = append(affected, AffectedConsumer{
				Consumer: c, Verdict: VerdictWillBreak, Confidence: 1.0,
				Reason:   fmt.Sprintf("requires claim %q", p.ClaimName),
				Evidence: []string{"expects.required_claims"},
			})
			continue
		}
		// Probe 9 evidence: a passing missing-claim sub-probe means the claim is enforced.
		if pr, ok := missingClaimProbe(history[c.StableID], p.ClaimName); ok && pr.Passed {
			affected = append(affected, AffectedConsumer{
				Consumer: c, Verdict: VerdictWillBreak, Confidence: 1.0,
				Reason:   "rejects tokens missing this claim",
				Evidence: []string{"probe:" + pr.ProbeID},
			})
			continue
		}
		// Consumers merely reading the claim are unknown in v1.
		unknown = append(unknown, c)
	}
	return affected, unknown
}

func resolveChangeIssuer(v model.ContractVersion, p ChangeProposal) ([]AffectedConsumer, []model.Consumer) {
	old := issuerURL(v, p.IssuerID)
	var affected []AffectedConsumer
	for _, c := range v.Consumers {
		if usesIssuer(c, old) && !usesIssuer(c, p.NewIssuerURL) {
			affected = append(affected, AffectedConsumer{
				Consumer: c, Verdict: VerdictWillBreak, Confidence: 1.0,
				Reason:   fmt.Sprintf("trusts old issuer %q, not %q", old, p.NewIssuerURL),
				Evidence: []string{"expects.issuers"},
			})
		}
	}
	return affected, nil
}

func resolveDropAlgorithm(v model.ContractVersion, p ChangeProposal) ([]AffectedConsumer, []model.Consumer) {
	var affected []AffectedConsumer
	for _, c := range v.Consumers {
		algs := c.Expects.Algorithms
		if len(algs) == 1 && algs[0] == p.Algorithm {
			affected = append(affected, AffectedConsumer{
				Consumer: c, Verdict: VerdictWillBreak, Confidence: 1.0,
				Reason:   fmt.Sprintf("only accepts %q", p.Algorithm),
				Evidence: []string{"expects.algorithms"},
			})
		}
	}
	return affected, nil
}

// --- helpers ----------------------------------------------------------------

func usesIssuer(c model.Consumer, url string) bool {
	if url == "" {
		return false
	}
	return containsStr(c.Expects.Issuers, url)
}

func readyWindow(c model.Consumer) (time.Duration, bool) {
	if c.JWKSBehavior.CacheTTLSec != nil {
		return time.Duration(*c.JWKSBehavior.CacheTTLSec) * time.Second, true
	}
	if c.JWKSBehavior.RefreshIntervalSec != nil {
		return time.Duration(*c.JWKSBehavior.RefreshIntervalSec) * time.Second, true
	}
	return 0, false
}

// measuredWindow derives a real announce→pickup latency from canary probe
// history (KI-25). When the scheduler (or an operator) re-runs the canary probe
// over time, the history records the moment the consumer went from rejecting the
// announced key (fail) to accepting it (pass). The gap between the last failing
// probe and the first passing probe after it bounds how long pickup actually
// took — a measurement, not a cache-TTL guess. Returns false when the history
// shows no such transition (e.g. the key was already picked up on first probe).
func measuredWindow(results []model.ProbeResult, now time.Time) (time.Duration, bool) {
	// Collect fresh canary probes in chronological order.
	var canaries []model.ProbeResult
	for _, r := range results {
		if r.ProbeID != probe.ProbeCanaryKey {
			continue
		}
		if now.Sub(r.RunAt) > canaryFreshness {
			continue
		}
		canaries = append(canaries, r)
	}
	sort.Slice(canaries, func(i, j int) bool { return canaries[i].RunAt.Before(canaries[j].RunAt) })

	// Find the last fail→pass transition and measure the gap across it.
	var lastFail time.Time
	haveFail := false
	var best time.Duration
	found := false
	for _, r := range canaries {
		if !r.Passed {
			lastFail, haveFail = r.RunAt, true
			continue
		}
		if haveFail {
			if d := r.RunAt.Sub(lastFail); d > 0 {
				best, found = d, true
			}
			haveFail = false // consumed this transition; a later fail starts a new one
		}
	}
	return best, found
}

func freshCanary(results []model.ProbeResult, now time.Time) (model.ProbeResult, bool) {
	var best model.ProbeResult
	found := false
	for _, r := range results {
		if r.ProbeID != probe.ProbeCanaryKey {
			continue
		}
		if now.Sub(r.RunAt) > canaryFreshness {
			continue
		}
		if !found || r.RunAt.After(best.RunAt) {
			best, found = r, true
		}
	}
	return best, found
}

func missingClaimProbe(results []model.ProbeResult, claim string) (model.ProbeResult, bool) {
	want := probe.ProbeMissingClaim + ":" + claim
	for _, r := range results {
		if r.ProbeID == want {
			return r, true
		}
	}
	return model.ProbeResult{}, false
}

func libEvidence(c model.Consumer) []string {
	if c.Library != nil {
		return []string{fmt.Sprintf("lib:%s %s", c.Library.Name, c.Library.Version)}
	}
	return []string{"source:" + string(c.JWKSBehavior.Source)}
}

func containsStr(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
