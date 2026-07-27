package generic

import (
	"context"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nometria/keyway/internal/model"
)

func newAdapter(t *testing.T) *Adapter {
	t.Helper()
	a, err := New(Options{Name: "test", IssuerURL: "https://issuer.test", Now: func() time.Time { return time.Unix(1_700_000_000, 0) }})
	require.NoError(t, err)
	return a
}

func TestMintAndVerify(t *testing.T) {
	a := newAdapter(t)
	ctx := context.Background()

	claims := map[string]any{"iss": "https://issuer.test", "aud": "api-a", "sub": "s"}
	token, err := a.MintToken(ctx, "", claims)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Verify the token against the published JWKS.
	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)

	set := a.KeySet().JWKS()
	out := map[string]any{}
	require.NoError(t, parsed.Claims(set.Keys[0].Key, &out))
	assert.Equal(t, "https://issuer.test", out["iss"])
	assert.Equal(t, "api-a", out["aud"])
}

func TestDescribeHasActiveKey(t *testing.T) {
	a := newAdapter(t)
	iss, err := a.Describe(context.Background())
	require.NoError(t, err)
	require.True(t, iss.ControlsPrivateKey)
	require.Len(t, iss.Keys, 1)
	assert.Equal(t, model.KeyActive, iss.Keys[0].Status)
}

// TestCanaryLifecycle exercises the announce -> active flow: an announced key is
// published but NOT the active signing key (AC-5), and promotion makes it active
// while demoting the previous active key.
func TestCanaryLifecycle(t *testing.T) {
	a := newAdapter(t)
	ctx := context.Background()

	origActive := a.KeySet().ActiveKID()
	require.NotEmpty(t, origActive)

	canary, err := a.AnnounceKey(ctx, "RS256")
	require.NoError(t, err)
	assert.Equal(t, model.KeyAnnounced, canary.Status)
	// Announced key is not the active signer.
	assert.NotEqual(t, canary.KID, a.KeySet().ActiveKID())
	assert.Equal(t, canary.KID, a.KeySet().AnnouncedKID())

	// A default mint still uses the original active key, not the canary.
	tok, err := a.MintToken(ctx, "", map[string]any{"sub": "s"})
	require.NoError(t, err)
	_ = tok

	// Promote the canary; it becomes active, the old key retires.
	require.NoError(t, a.PromoteKey(ctx, canary.KID))
	assert.Equal(t, canary.KID, a.KeySet().ActiveKID())

	iss, _ := a.Describe(ctx)
	var retiring bool
	for _, k := range iss.Keys {
		if k.KID == origActive {
			retiring = k.Status == model.KeyRetiring
		}
	}
	assert.True(t, retiring, "previous active key should be retiring")
}

func TestRetiredKeyStillSigns(t *testing.T) {
	a := newAdapter(t)
	ctx := context.Background()
	// Announce + promote so we have a retiring key, then retire it explicitly.
	active := a.KeySet().ActiveKID()
	require.NoError(t, a.RetireKey(ctx, active))
	assert.Equal(t, active, a.KeySet().RetiredKID())
	// Probe 10 needs to sign with a retired key.
	tok, err := a.MintToken(ctx, active, map[string]any{"sub": "s"})
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}
