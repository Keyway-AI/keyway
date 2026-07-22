// Package libdefaults ships Keyway's library-behaviour database and looks up
// known JWKS behaviour by library name and version. This is the only source for
// RefreshesOnUnknownKID when probing is unavailable (PRD §7.5).
package libdefaults

import (
	_ "embed"
	"fmt"

	"github.com/architsharma/keyway/internal/model"
	"gopkg.in/yaml.v3"
)

//go:embed data/defaults.yaml
var embedded []byte

// VersionEntry is one version-constrained behaviour record.
type VersionEntry struct {
	Constraint   string           `yaml:"constraint"`
	JWKSBehavior jwksBehaviorYAML `yaml:"jwks_behavior"`
	Notes        string           `yaml:"notes"`
	Risk         string           `yaml:"risk"`
}

type jwksBehaviorYAML struct {
	CacheTTLSec           *int  `yaml:"cache_ttl_sec"`
	RefreshIntervalSec    *int  `yaml:"refresh_interval_sec"`
	RefreshesOnUnknownKID *bool `yaml:"refreshes_on_unknown_kid"`
}

// Library groups version entries for one library.
type Library struct {
	Name     string         `yaml:"name"`
	Lang     string         `yaml:"lang"`
	Versions []VersionEntry `yaml:"versions"`
}

// DB is the loaded library database.
type DB struct {
	byName map[string]Library
}

type fileShape struct {
	Libraries []Library `yaml:"libraries"`
}

// Load parses the embedded database.
func Load() (*DB, error) {
	var f fileShape
	if err := yaml.Unmarshal(embedded, &f); err != nil {
		return nil, fmt.Errorf("libdefaults: parse embedded db: %w", err)
	}
	db := &DB{byName: make(map[string]Library, len(f.Libraries))}
	for _, lib := range f.Libraries {
		db.byName[lib.Name] = lib
	}
	return db, nil
}

// Match returns the JWKS behaviour and metadata for a library at a version.
//
// TODO(M5): apply the semver constraint in VersionEntry.Constraint. For now the
// first entry for a known library is returned so callers can wire the flow; the
// constraint solver lands with the detection code.
func (db *DB) Match(name, version string) (model.JWKSBehavior, VersionEntry, bool) {
	lib, ok := db.byName[name]
	if !ok || len(lib.Versions) == 0 {
		return model.JWKSBehavior{}, VersionEntry{}, false
	}
	_ = version
	entry := lib.Versions[0]
	behavior := model.JWKSBehavior{
		CacheTTLSec:           entry.JWKSBehavior.CacheTTLSec,
		RefreshIntervalSec:    entry.JWKSBehavior.RefreshIntervalSec,
		RefreshesOnUnknownKID: entry.JWKSBehavior.RefreshesOnUnknownKID,
		Source:                model.SrcLibraryDefault,
	}
	return behavior, entry, true
}

// Names lists the libraries in the database (useful for diagnostics).
func (db *DB) Names() []string {
	out := make([]string, 0, len(db.byName))
	for n := range db.byName {
		out = append(out, n)
	}
	return out
}
