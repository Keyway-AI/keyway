package localkeys

import (
	"testing"
	"time"

	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportImportRoundTrip verifies a key set survives a serialize→deserialize
// cycle with private material intact: the imported key can still sign, and its
// status/kid are preserved (the core of KI-09).
func TestExportImportRoundTrip(t *testing.T) {
	ks := NewKeySet()
	now := time.Now()
	announced, err := ks.Generate("RS256", model.KeyAnnounced, now)
	require.NoError(t, err)
	active, err := ks.Generate("RS256", model.KeyActive, now)
	require.NoError(t, err)

	exported := ks.Export()
	require.Len(t, exported, 2)

	// Fresh set (simulating a restart) imports the persisted keys.
	restored := NewKeySet()
	require.NoError(t, restored.Import(exported))

	assert.Equal(t, announced.KID, restored.AnnouncedKID(), "announced kid preserved")
	assert.Equal(t, active.KID, restored.ActiveKID(), "active kid preserved")

	// The restored private key still signs, and produces a verifiable token kid.
	tok, err := restored.Sign(active.KID, map[string]any{"sub": "svc"})
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
	assert.Len(t, restored.JWKS().Keys, 2, "both public keys republished")

	// Import is idempotent: re-importing the same keys adds nothing.
	require.NoError(t, restored.Import(exported))
	assert.Len(t, restored.Keys(), 2)
}

// TestOnChangeFires verifies every lifecycle mutation notifies the persistence
// hook exactly once, with the full current state.
func TestOnChangeFires(t *testing.T) {
	ks := NewKeySet()
	var calls int
	var last []PersistedKey
	ks.SetOnChange(func(k []PersistedKey) { calls++; last = k })

	now := time.Now()
	k, err := ks.Generate("RS256", model.KeyAnnounced, now)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Len(t, last, 1)

	require.NoError(t, ks.Promote(k.KID, now))
	require.Equal(t, 2, calls)

	require.NoError(t, ks.Retire(k.KID, now))
	require.Equal(t, 3, calls)
	assert.Equal(t, string(model.KeyRetired), last[0].Status)
}
