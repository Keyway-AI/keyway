package libdefaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbedded(t *testing.T) {
	db, err := Load()
	require.NoError(t, err)
	require.NotNil(t, db)
	assert.NotEmpty(t, db.Names())
}

// TestKeyfuncFinding is the mechanism behind AC-6: the keyfunc default must
// surface refreshes_on_unknown_kid=false with zero probes.
func TestKeyfuncFinding(t *testing.T) {
	db, err := Load()
	require.NoError(t, err)

	behavior, entry, ok := db.Match("MicahParks/keyfunc", "v1.9.0")
	require.True(t, ok, "keyfunc must be known")
	require.NotNil(t, behavior.RefreshesOnUnknownKID)
	assert.False(t, *behavior.RefreshesOnUnknownKID, "keyfunc v1.x does not refresh on unknown kid")
	assert.Equal(t, "high", entry.Risk)
}

func TestUnknownLibrary(t *testing.T) {
	db, err := Load()
	require.NoError(t, err)
	_, _, ok := db.Match("nonexistent/library", "v1.0.0")
	assert.False(t, ok)
}
