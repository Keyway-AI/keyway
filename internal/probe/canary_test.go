package probe_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/issuer/generic"
	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/probe"
)

// pinnedValidator accepts only tokens signed by a specific kid (a consumer that
// has NOT yet picked up the canary key). trustAll accepts any JWKS key (a
// consumer that HAS refreshed and trusts the announced key).
func canaryValidator(iss *generic.Adapter, pinnedKID string, trustAll bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		kid := tok.Headers[0].KeyID
		if !trustAll && kid != pinnedKID {
			w.WriteHeader(http.StatusUnauthorized) // hasn't picked up the new key
			return
		}
		set := iss.KeySet().JWKS()
		keys := set.Key(kid)
		if len(keys) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var claims map[string]any
		if err := tok.Claims(keys[0].Key, &claims); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

// TestAC5_CanarySeparatesReadyFromNotReady is AC-5: after announcing a canary
// key (which is NOT used for signing), probe 13 accepts on a consumer that
// trusts it (ready) and is rejected by one that only trusts the old key.
func TestAC5_CanarySeparatesReadyFromNotReady(t *testing.T) {
	iss, err := generic.New(generic.Options{Name: "kc", IssuerURL: expectedIssuer})
	require.NoError(t, err)
	ctx := context.Background()

	originalActive := iss.KeySet().ActiveKID()

	// Announce a canary key. It must NOT become the active signer.
	canary, err := iss.AnnounceKey(ctx, "RS256")
	require.NoError(t, err)
	require.Equal(t, originalActive, iss.KeySet().ActiveKID(), "announced key must not sign")
	require.Equal(t, canary.KID, iss.KeySet().AnnouncedKID())

	ready := httptest.NewServer(canaryValidator(iss, originalActive, true))     // trusts new key
	notReady := httptest.NewServer(canaryValidator(iss, originalActive, false)) // pinned to old key
	defer ready.Close()
	defer notReady.Close()

	results := runEngine(t, iss, []model.Consumer{
		named(testConsumer(ready.URL, nil), "ready-svc", "k8s://t/d/ready"),
		named(testConsumer(notReady.URL, nil), "notready-svc", "k8s://t/d/notready"),
	})

	// The engine keys results by ProbeID (last write wins across consumers), so
	// re-run per consumer to inspect the canary verdict individually.
	readyRes := probeOne(t, iss, testConsumer(ready.URL, nil), probe.ProbeCanaryKey)
	require.NotNil(t, readyRes)
	assert.Equal(t, 200, readyRes.StatusCode)
	assert.True(t, readyRes.Passed, "ready consumer accepts the canary key")

	notReadyRes := probeOne(t, iss, testConsumer(notReady.URL, nil), probe.ProbeCanaryKey)
	require.NotNil(t, notReadyRes)
	assert.Equal(t, 401, notReadyRes.StatusCode)
	assert.False(t, notReadyRes.Passed, "not-ready consumer rejects the canary key")

	_ = results
}

func named(c model.Consumer, name, stableID string) model.Consumer {
	c.Name = name
	c.StableID = stableID
	c.ID = stableID
	return c
}

func probeOne(t *testing.T, iss *generic.Adapter, c model.Consumer, probeID string) *model.ProbeResult {
	t.Helper()
	eng := probe.NewEngine(probe.EngineConfig{
		Allowlist:      []string{"127.0.0.1"},
		RequestTimeout: 5 * time.Second,
	})
	issModel, _ := iss.Describe(context.Background())
	results, _, err := eng.Run(context.Background(), issModel, mintOf(iss), []model.Consumer{c})
	require.NoError(t, err)
	for i := range results {
		if results[i].ProbeID == probeID {
			return &results[i]
		}
	}
	return nil
}
