package libdefaults

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nometria/keyway/internal/model"
)

// DetectDir scans a directory (non-recursively, then one level into common
// module roots) for dependency manifests and returns the JWT libraries it can
// identify. It reads go.mod, package.json, pom.xml, build.gradle,
// requirements.txt and Cargo.toml (PRD §7.5).
func DetectDir(dir string) []model.LibraryInfo {
	var libs []model.LibraryInfo
	add := func(more []model.LibraryInfo) { libs = append(libs, more...) }

	add(parseGoMod(filepath.Join(dir, "go.mod")))
	add(parsePackageJSON(filepath.Join(dir, "package.json")))
	add(parsePomXML(filepath.Join(dir, "pom.xml")))
	add(parseGradle(filepath.Join(dir, "build.gradle")))
	add(parseRequirements(filepath.Join(dir, "requirements.txt")))
	add(parseCargo(filepath.Join(dir, "Cargo.toml")))
	return libs
}

// DetectFor scans a directory and returns the first library present in the DB
// together with its resolved JWKS behavior. This is the AC-6 path: a repo using
// keyfunc v1.9.0 yields RefreshesOnUnknownKID=false with zero probes.
func (db *DB) DetectFor(dir string) (*model.LibraryInfo, model.JWKSBehavior, bool) {
	for _, lib := range DetectDir(dir) {
		if behavior, _, ok := db.Match(lib.Name, lib.Version); ok {
			l := lib
			return &l, behavior, true
		}
	}
	return nil, model.JWKSBehavior{}, false
}

// Enrich fills a consumer's unknown JWKS behavior fields from its library's
// documented defaults. Config/observed/probed values already present take
// precedence — library defaults only fill gaps. Returns true if it matched.
func (db *DB) Enrich(c *model.Consumer) bool {
	if c.Library == nil {
		return false
	}
	b, _, ok := db.Match(c.Library.Name, c.Library.Version)
	if !ok {
		return false
	}
	if c.JWKSBehavior.RefreshesOnUnknownKID == nil {
		c.JWKSBehavior.RefreshesOnUnknownKID = b.RefreshesOnUnknownKID
	}
	if c.JWKSBehavior.CacheTTLSec == nil {
		c.JWKSBehavior.CacheTTLSec = b.CacheTTLSec
	}
	if c.JWKSBehavior.RefreshIntervalSec == nil {
		c.JWKSBehavior.RefreshIntervalSec = b.RefreshIntervalSec
	}
	if c.JWKSBehavior.Source == "" {
		c.JWKSBehavior.Source = model.SrcLibraryDefault
	}
	return true
}

func readFile(path string) ([]byte, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return b, true
}

var goModRe = regexp.MustCompile(`^\s*([^\s]+/[^\s]+)\s+(v[0-9][^\s/]*)`)

func parseGoMod(path string) []model.LibraryInfo {
	b, ok := readFile(path)
	if !ok {
		return nil
	}
	var libs []model.LibraryInfo
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "// indirect") {
			continue
		}
		line = strings.TrimPrefix(strings.TrimSpace(line), "require ")
		if m := goModRe.FindStringSubmatch(line); m != nil {
			libs = append(libs, model.LibraryInfo{Name: m[1], Version: m[2], Lang: "go"})
		}
	}
	return libs
}

func parsePackageJSON(path string) []model.LibraryInfo {
	b, ok := readFile(path)
	if !ok {
		return nil
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return nil
	}
	var libs []model.LibraryInfo
	for _, deps := range []map[string]string{pkg.Dependencies, pkg.DevDependencies} {
		for name, ver := range deps {
			libs = append(libs, model.LibraryInfo{Name: name, Version: cleanVersion(ver), Lang: "node"})
		}
	}
	return libs
}

func parsePomXML(path string) []model.LibraryInfo {
	b, ok := readFile(path)
	if !ok {
		return nil
	}
	var pom struct {
		Dependencies struct {
			Dependency []struct {
				ArtifactID string `xml:"artifactId"`
				Version    string `xml:"version"`
			} `xml:"dependency"`
		} `xml:"dependencies"`
	}
	if xml.Unmarshal(b, &pom) != nil {
		return nil
	}
	var libs []model.LibraryInfo
	for _, d := range pom.Dependencies.Dependency {
		libs = append(libs, model.LibraryInfo{Name: d.ArtifactID, Version: cleanVersion(d.Version), Lang: "java"})
	}
	return libs
}

var gradleRe = regexp.MustCompile(`['"]([\w.\-]+):([\w.\-]+):([\w.\-]+)['"]`)

func parseGradle(path string) []model.LibraryInfo {
	b, ok := readFile(path)
	if !ok {
		return nil
	}
	var libs []model.LibraryInfo
	for _, m := range gradleRe.FindAllStringSubmatch(string(b), -1) {
		libs = append(libs, model.LibraryInfo{Name: m[2], Version: m[3], Lang: "java"})
	}
	return libs
}

var reqRe = regexp.MustCompile(`^([A-Za-z0-9_.\-]+)\s*(?:[=<>!~]=?\s*([0-9][\w.\-]*))?`)

func parseRequirements(path string) []model.LibraryInfo {
	b, ok := readFile(path)
	if !ok {
		return nil
	}
	var libs []model.LibraryInfo
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := reqRe.FindStringSubmatch(line); m != nil {
			libs = append(libs, model.LibraryInfo{Name: m[1], Version: m[2], Lang: "python"})
		}
	}
	return libs
}

var cargoRe = regexp.MustCompile(`^([A-Za-z0-9_\-]+)\s*=\s*(?:"([^"]+)"|\{[^}]*version\s*=\s*"([^"]+)")`)

func parseCargo(path string) []model.LibraryInfo {
	b, ok := readFile(path)
	if !ok {
		return nil
	}
	var libs []model.LibraryInfo
	inDeps := false
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			inDeps = strings.Contains(line, "dependencies")
			continue
		}
		if !inDeps {
			continue
		}
		if m := cargoRe.FindStringSubmatch(line); m != nil {
			ver := m[2]
			if ver == "" {
				ver = m[3]
			}
			libs = append(libs, model.LibraryInfo{Name: m[1], Version: ver, Lang: "rust"})
		}
	}
	return libs
}

// cleanVersion strips common range prefixes (^, ~, >=, v) leaving a bare semver.
func cleanVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimLeft(v, "^~>=<")
	v = strings.TrimSpace(v)
	return v
}
