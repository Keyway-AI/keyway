// Package libdefaults ships Keyway's library-behavior database and looks up
// known JWKS behavior by library name and version. This is the only source for
// RefreshesOnUnknownKID when probing is unavailable (PRD §7.5).
package libdefaults

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"

	"github.com/Keyway-AI/keyway/internal/model"
)

//go:embed data/defaults.yaml
var embedded []byte

// VersionEntry is one version-constrained behavior record.
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

// Match returns the JWKS behavior and metadata for a library at a version. The
// library name is matched exactly or by path suffix (so a go.mod module path
// "github.com/MicahParks/keyfunc" matches the DB entry "MicahParks/keyfunc").
// The version is matched against each entry's semver constraint. If no
// version-specific constraint matches, an explicit catch-all entry (empty
// Constraint) is used when the library defines one; otherwise the version is out
// of the known range and Match returns ok=false — Keyway must not label an
// unknown/out-of-range version with a specific version's known-bad behavior.
func (db *DB) Match(name, version string) (model.JWKSBehavior, VersionEntry, bool) {
	lib, ok := db.lookup(name)
	if !ok || len(lib.Versions) == 0 {
		return model.JWKSBehavior{}, VersionEntry{}, false
	}

	ver, verErr := semver.NewVersion(strings.TrimPrefix(version, "v"))

	catchAll := -1
	for i, entry := range lib.Versions {
		if entry.Constraint == "" {
			if catchAll < 0 {
				catchAll = i // remember it, but prefer a version-specific match
			}
			continue
		}
		c, err := semver.NewConstraint(entry.Constraint)
		if err != nil {
			continue
		}
		if verErr == nil && c.Check(ver) {
			return behaviorOf(entry), entry, true
		}
	}
	if catchAll >= 0 {
		entry := lib.Versions[catchAll]
		return behaviorOf(entry), entry, true
	}
	// Known library, but the version matches no constraint (or is unparseable):
	// unknown behavior, not a specific known-bad one.
	return model.JWKSBehavior{}, VersionEntry{}, false
}

// lookup finds a library by exact name or by "/"-suffix match.
func (db *DB) lookup(name string) (Library, bool) {
	if lib, ok := db.byName[name]; ok {
		return lib, true
	}
	for dbName, lib := range db.byName {
		if strings.HasSuffix(name, "/"+dbName) || strings.EqualFold(name, dbName) {
			return lib, true
		}
	}
	return Library{}, false
}

func behaviorOf(entry VersionEntry) model.JWKSBehavior {
	return model.JWKSBehavior{
		CacheTTLSec:           entry.JWKSBehavior.CacheTTLSec,
		RefreshIntervalSec:    entry.JWKSBehavior.RefreshIntervalSec,
		RefreshesOnUnknownKID: entry.JWKSBehavior.RefreshesOnUnknownKID,
		Source:                model.SrcLibraryDefault,
	}
}

// Names lists the libraries in the database (useful for diagnostics).
func (db *DB) Names() []string {
	out := make([]string, 0, len(db.byName))
	for n := range db.byName {
		out = append(out, n)
	}
	return out
}
