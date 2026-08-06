package issuerregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/internal/keystore"
	"github.com/Keyway-AI/keyway/internal/model"
)

// TestCanaryStateSurvivesRestart is the end-to-end KI-09 guarantee: a canary key
// announced before a restart is still present after the daemon rebuilds its
// registry from the same store.
func TestCanaryStateSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 7)
	}
	store, err := keystore.NewFileStore(dir, encKey)
	require.NoError(t, err)

	specs := []Spec{{Name: "default", Type: model.IssuerGenericOIDC, URL: "https://issuer.test"}}

	reg, err := NewRegistryWithStore(specs, store)
	require.NoError(t, err)
	iss, ok := reg.Get("default")
	require.True(t, ok)

	announced, err := iss.AnnounceKey(context.Background(), "RS256")
	require.NoError(t, err)
	require.NotEmpty(t, announced.KID)

	// Rebuild the registry from the same store — simulates `keyway serve` restart.
	reg2, err := NewRegistryWithStore(specs, store)
	require.NoError(t, err)
	iss2, ok := reg2.Get("default")
	require.True(t, ok)

	assert.Equal(t, announced.KID, iss2.KeySet().AnnouncedKID(),
		"announced canary key must survive a restart")

	// And the restored key is usable: promoting then signing works.
	require.NoError(t, iss2.PromoteKey(context.Background(), announced.KID))
	tok, err := iss2.KeySet().Sign(announced.KID, map[string]any{"sub": "svc"})
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

// TestNoStoreIsInMemory verifies the default (nil store) path keeps working and
// simply does not persist.
func TestNoStoreIsInMemory(t *testing.T) {
	specs := []Spec{{Name: "default", Type: model.IssuerGenericOIDC, URL: "https://issuer.test"}}
	reg, err := NewRegistry(specs)
	require.NoError(t, err)
	iss, ok := reg.Get("default")
	require.True(t, ok)
	_, err = iss.AnnounceKey(context.Background(), "RS256")
	require.NoError(t, err)
}
