package libdefaults

import (
	"testing"

	"github.com/architsharma/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrich(t *testing.T) {
	db, err := Load()
	require.NoError(t, err)

	c := model.Consumer{Library: &model.LibraryInfo{Name: "MicahParks/keyfunc", Version: "v1.9.0"}}
	require.True(t, db.Enrich(&c))
	require.NotNil(t, c.JWKSBehavior.RefreshesOnUnknownKID)
	assert.False(t, *c.JWKSBehavior.RefreshesOnUnknownKID)
	assert.Equal(t, model.SrcLibraryDefault, c.JWKSBehavior.Source)

	// A config-provided value is not overwritten by library defaults.
	tr := true
	c2 := model.Consumer{
		Library:      &model.LibraryInfo{Name: "MicahParks/keyfunc", Version: "v1.9.0"},
		JWKSBehavior: model.JWKSBehavior{RefreshesOnUnknownKID: &tr, Source: model.SrcConfig},
	}
	require.True(t, db.Enrich(&c2))
	assert.True(t, *c2.JWKSBehavior.RefreshesOnUnknownKID, "config value must win over library default")
	assert.Equal(t, model.SrcConfig, c2.JWKSBehavior.Source)
}

func TestSemverMatching(t *testing.T) {
	db, err := Load()
	require.NoError(t, err)

	// keyfunc v1.9.0 is in the >=1.0.0 <2.0.0 band -> refresh false.
	b1, e1, ok := db.Match("MicahParks/keyfunc", "v1.9.0")
	require.True(t, ok)
	require.NotNil(t, b1.RefreshesOnUnknownKID)
	assert.False(t, *b1.RefreshesOnUnknownKID)
	assert.Equal(t, "high", e1.Risk)

	// keyfunc v2.1.0 is in the >=2.0.0 band -> refresh true.
	b2, _, ok := db.Match("MicahParks/keyfunc", "v2.1.0")
	require.True(t, ok)
	require.NotNil(t, b2.RefreshesOnUnknownKID)
	assert.True(t, *b2.RefreshesOnUnknownKID)
}

func TestLookupBySuffix(t *testing.T) {
	db, err := Load()
	require.NoError(t, err)
	// go.mod module path form must match the DB's short name.
	_, _, ok := db.Match("github.com/MicahParks/keyfunc", "v1.9.0")
	assert.True(t, ok)
}

func TestDetectGoMod(t *testing.T) {
	libs := DetectDir("../../testdata/libdefaults/go-keyfunc")
	names := map[string]string{}
	for _, l := range libs {
		names[l.Name] = l.Version
	}
	require.Contains(t, names, "github.com/MicahParks/keyfunc")
	assert.Equal(t, "v1.9.0", names["github.com/MicahParks/keyfunc"])
	// indirect deps are excluded.
	assert.NotContains(t, names, "github.com/golang-jwt/jwt/v5")
}

func TestDetectPython(t *testing.T) {
	libs := DetectDir("../../testdata/libdefaults/python-app")
	found := false
	for _, l := range libs {
		if l.Name == "PyJWT" && l.Version == "2.8.0" {
			found = true
		}
	}
	assert.True(t, found, "PyJWT should be detected from requirements.txt")
}

func TestDetectNode(t *testing.T) {
	libs := DetectDir("../../testdata/libdefaults/node-app")
	found := false
	for _, l := range libs {
		if l.Name == "jsonwebtoken" {
			found = true
			assert.Equal(t, "9.0.2", l.Version)
		}
	}
	assert.True(t, found, "jsonwebtoken should be detected from package.json")
}

// TestAC6_ZeroProbeFinding is AC-6: a repo using keyfunc v1.9.0 yields
// refreshes_on_unknown_kid=false from library defaults alone — no probes.
func TestAC6_ZeroProbeFinding(t *testing.T) {
	db, err := Load()
	require.NoError(t, err)

	lib, behavior, ok := db.DetectFor("../../testdata/libdefaults/go-keyfunc")
	require.True(t, ok, "keyfunc must be detected and matched")
	require.NotNil(t, lib)
	assert.Equal(t, "github.com/MicahParks/keyfunc", lib.Name)
	require.NotNil(t, behavior.RefreshesOnUnknownKID)
	assert.False(t, *behavior.RefreshesOnUnknownKID, "keyfunc v1.9.0 does not refresh on unknown kid")
	assert.Equal(t, "library_default", string(behavior.Source))
}
