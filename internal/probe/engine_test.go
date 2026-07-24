package probe_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nometria/keyway/internal/issuer/generic"
	"github.com/nometria/keyway/internal/model"
	"github.com/nometria/keyway/internal/probe"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedIssuer   = "https://issuer.test"
	expectedAudience = "api-a"
)

// validator is a realistic JWT-checking server. When trustHeader is true it is
// deliberately misconfigured to trust the X-User-Id header (the header_bypass
// vulnerability probe 12 must catch).
func validator(iss *generic.Adapter, requiredClaims []string, trustHeader bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		raw := strings.TrimPrefix(auth, "Bearer ")

		if raw == "" {
			if trustHeader && r.Header.Get("X-User-Id") != "" {
				w.WriteHeader(http.StatusOK) // VULNERABLE: trusts identity header
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Only RS256 is accepted; this rejects alg=none (6) and HS256 confusion (7).
		tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		set := iss.KeySet().JWKS() // excludes retired keys -> retired-key token (10) fails
		var kid string
		if len(tok.Headers) > 0 {
			kid = tok.Headers[0].KeyID
		}
		keys := set.Key(kid)
		if len(keys) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var claims map[string]any
		if err := tok.Claims(keys[0].Key, &claims); err != nil { // verifies signature (tamper=8)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		now := time.Now()
		if !validClaims(claims, now, requiredClaims) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func validClaims(claims map[string]any, now time.Time, required []string) bool {
	if s, _ := claims["iss"].(string); s != expectedIssuer {
		return false
	}
	if s, _ := claims["aud"].(string); s != expectedAudience {
		return false
	}
	if exp, ok := claims["exp"].(float64); ok && now.Unix() > int64(exp) {
		return false
	}
	if nbf, ok := claims["nbf"].(float64); ok && now.Unix() < int64(nbf) {
		return false
	}
	for _, c := range required {
		if _, ok := claims[c]; !ok {
			return false
		}
	}
	return true
}

func testConsumer(url string, required []string) model.Consumer {
	return model.Consumer{
		ID:       "c1",
		StableID: "k8s://test/default/api-a",
		Kind:     model.ConsumerService,
		Name:     "api-a",
		Endpoints: []model.Endpoint{
			{URL: url, Method: "GET", SafeProbePath: "/"},
		},
		Expects: model.Expectations{
			Issuers:        []string{expectedIssuer},
			Audiences:      []string{expectedAudience},
			Algorithms:     []string{"RS256"},
			RequiredClaims: required,
		},
		Probeable: true,
	}
}

func newIssuer(t *testing.T) *generic.Adapter {
	t.Helper()
	a, err := generic.New(generic.Options{Name: "test", IssuerURL: expectedIssuer})
	require.NoError(t, err)
	return a
}

// mintOf adapts an issuer's context-taking MintToken to the engine's MintFunc.
func mintOf(iss *generic.Adapter) probe.MintFunc {
	return func(kid string, claims map[string]any) (string, error) {
		return iss.MintToken(context.Background(), kid, claims)
	}
}

func runEngine(t *testing.T, iss *generic.Adapter, consumers []model.Consumer) map[string]model.ProbeResult {
	t.Helper()
	eng := probe.NewEngine(probe.EngineConfig{
		MaxConcurrentGlobal:   4,
		RequestTimeout:        5 * time.Second,
		InterProbeDelay:       0,
		AbortOnConsecutive5xx: 3,
		Allowlist:             []string{"127.0.0.1"},
	})
	issModel, err := iss.Describe(context.Background())
	require.NoError(t, err)
	results, _, err := eng.Run(context.Background(), issModel, mintOf(iss), consumers)
	require.NoError(t, err)
	byID := map[string]model.ProbeResult{}
	for _, r := range results {
		byID[r.ProbeID] = r
	}
	return byID
}

// TestSecureServerAllProbesPass is AC-4: against a correctly-configured service,
// every probe's expectation holds (valid accepted, everything malicious rejected).
func TestSecureServerAllProbesPass(t *testing.T) {
	iss := newIssuer(t)
	srv := httptest.NewServer(validator(iss, []string{"dept"}, false))
	defer srv.Close()

	results := runEngine(t, iss, []model.Consumer{testConsumer(srv.URL, []string{"dept"})})

	// Baseline accepted.
	require.Contains(t, results, probe.ProbeValidToken)
	assert.Equal(t, 200, results[probe.ProbeValidToken].StatusCode)
	assert.True(t, results[probe.ProbeValidToken].Passed, "valid token must be accepted")

	// Every rejection probe must PASS (i.e. the server correctly rejected).
	for _, id := range []string{
		probe.ProbeExpired, probe.ProbeNotYetValid, probe.ProbeWrongIssuer,
		probe.ProbeWrongAudience, probe.ProbeAlgNone, probe.ProbeAlgConfusion,
		probe.ProbeTamperedSignature, probe.ProbeHeaderBypass,
	} {
		r, ok := results[id]
		require.True(t, ok, "probe %s should have run", id)
		assert.Truef(t, r.Passed, "probe %s should pass (server rejected), got status %d", id, r.StatusCode)
		assert.Equalf(t, 401, r.StatusCode, "probe %s expected 401", id)
	}

	// Probe 9 expands per required claim.
	r, ok := results[probe.ProbeMissingClaim+":dept"]
	require.True(t, ok, "missing-claim sub-probe should run")
	assert.True(t, r.Passed)
}

// TestHeaderBypassFlagged is AC-4's specific check: a service that trusts
// X-User-Id must be flagged by the header_bypass probe (it returns 200, so the
// probe does NOT pass — that failure is the finding).
func TestHeaderBypassFlagged(t *testing.T) {
	iss := newIssuer(t)
	srv := httptest.NewServer(validator(iss, nil, true)) // trusts X-User-Id
	defer srv.Close()

	results := runEngine(t, iss, []model.Consumer{testConsumer(srv.URL, nil)})

	r, ok := results[probe.ProbeHeaderBypass]
	require.True(t, ok)
	assert.Equal(t, 200, r.StatusCode, "misconfigured server accepts header identity")
	assert.False(t, r.Passed, "header_bypass must be flagged (probe fails) on a vulnerable service")
}

// TestStagingGuardDenies verifies the default-deny production guard.
func TestStagingGuardDenies(t *testing.T) {
	iss := newIssuer(t)
	srv := httptest.NewServer(validator(iss, nil, false))
	defer srv.Close()

	eng := probe.NewEngine(probe.EngineConfig{Allowlist: nil}) // deny all
	issModel, _ := iss.Describe(context.Background())
	_, outcomes, err := eng.Run(context.Background(), issModel, mintOf(iss), []model.Consumer{testConsumer(srv.URL, nil)})
	require.NoError(t, err)
	require.Len(t, outcomes, 1)
	assert.True(t, outcomes[0].Skipped)
	assert.Contains(t, outcomes[0].Reason, "allowlist")
}

// TestBaseline5xxSkips verifies a broken service is marked unverified, not a finding.
func TestBaseline5xxSkips(t *testing.T) {
	iss := newIssuer(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	eng := probe.NewEngine(probe.EngineConfig{Allowlist: []string{"127.0.0.1"}, InterProbeDelay: 0})
	issModel, _ := iss.Describe(context.Background())
	_, outcomes, err := eng.Run(context.Background(), issModel, mintOf(iss), []model.Consumer{testConsumer(srv.URL, nil)})
	require.NoError(t, err)
	require.Len(t, outcomes, 1)
	assert.True(t, outcomes[0].Skipped)
	assert.Contains(t, outcomes[0].Reason, "5xx")
}

// TestSiblingTokenRejected verifies probe 11 uses another consumer's audience.
func TestSiblingTokenRejected(t *testing.T) {
	iss := newIssuer(t)
	srv := httptest.NewServer(validator(iss, nil, false))
	defer srv.Close()

	main := testConsumer(srv.URL, nil)
	sibling := model.Consumer{StableID: "other", Name: "other", Expects: model.Expectations{Audiences: []string{"other-aud"}}}
	results := runEngine(t, iss, []model.Consumer{main, sibling})

	r, ok := results[probe.ProbeSiblingClientToken]
	require.True(t, ok, "sibling probe should run when another audience exists")
	assert.True(t, r.Passed, "token for another audience must be rejected")
}
