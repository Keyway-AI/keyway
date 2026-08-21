// Package render prepares a raw config corpus for discovery by resolving the two
// templating systems that leave scraped manifests under-resolved: Helm/Go
// templates ({{ ... }}) and Kustomize overlays. It exists because ~1 in 5 files in
// the Paper A corpus carry template markers, and reading those raw makes discovery
// miss real values (a templated `issuer:` looks like no issuer at all), which
// understates recall. See bench/measurement/FINDINGS.md ("Templating limitation").
//
// It does two things, and it fabricates nothing:
//
//   - When a real renderable unit is present (a directory with a Chart.yaml or a
//     kustomization.yaml), it shells out to `helm template` / `kustomize build` and
//     uses the rendered YAML. This is the correct path once the crawler fetches
//     whole trees (G2).
//   - For a standalone templated file (the current single-blob corpus), it cannot
//     render — there is no chart context or values — so it does NOT invent values.
//     It neutralizes template expressions into a visible sentinel so the YAML
//     parses, and records which auth-relevant fields were templated. A templated
//     value becomes a recognizable placeholder, never a plausible-looking fake.
package render

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Placeholder is what a templated scalar becomes after Neutralize. It is
// deliberately conspicuous and syntactically a plain YAML scalar, so a templated
// `issuer: {{ .Values.x }}` parses as a present-but-unresolved issuer rather than
// as an absent one. Downstream code can recognize it to caveat the measurement.
const Placeholder = "__KEYWAY_TEMPLATED__"

// Engine is the templating system detected in a file.
type Engine string

const (
	EnginePlain     Engine = "plain"
	EngineHelm      Engine = "helm"      // Go/Helm text templates: {{ ... }}
	EngineKustomize Engine = "kustomize" // kustomization.yaml or $(VAR) refs
)

// tmplRe matches a single Helm/Go template action, e.g. {{ .Values.x }} or
// {{- if .enabled -}}. Non-greedy so multiple actions on one line resolve
// independently; `.` does not cross newlines, matching how templates are written.
var tmplRe = regexp.MustCompile(`{{-?.*?-?}}`)

// controlKW are the template actions that emit no value (control flow / helpers).
// A line whose only template content is one of these is structural and is dropped;
// a line with a value action keeps a placeholder.
var controlKW = map[string]bool{
	"if": true, "else": true, "end": true, "range": true, "with": true,
	"define": true, "template": true, "block": true,
}

// authKeyRe matches the config keys whose value, if templated and read literally,
// would corrupt a prevalence check (a templated issuer/audience must not be counted
// as absent). Used only to *report* which auth fields were under-resolved.
var authKeyRe = regexp.MustCompile(`(?i)^\s*-?\s*(issuer|audiences?|jwks_?uri|jwks|remote_?jwks|from_?headers?|output_?payload_?to_?header)\s*:`)

// FileClass is the result of inspecting one file's bytes (and name).
type FileClass struct {
	Engine              Engine
	Templated           bool     // has {{ }} actions
	AuthFieldsTemplated []string // auth keys whose value contains a template action
}

// Classify inspects a file without modifying it. name is used only to spot a
// kustomization file by convention; the decision is otherwise content-based.
func Classify(name string, b []byte) FileClass {
	fc := FileClass{Engine: EnginePlain}
	base := strings.ToLower(filepath.Base(name))
	content := string(b)

	if base == "kustomization.yaml" || base == "kustomization.yml" ||
		strings.Contains(content, "apiVersion: kustomize.config.k8s.io") {
		fc.Engine = EngineKustomize
	}
	if tmplRe.MatchString(content) {
		fc.Templated = true
		if fc.Engine == EnginePlain {
			fc.Engine = EngineHelm
		}
	}
	if fc.Templated {
		for _, line := range strings.Split(content, "\n") {
			if authKeyRe.MatchString(line) && strings.Contains(line, "{{") {
				if key := authKeyRe.FindStringSubmatch(line); key != nil {
					fc.AuthFieldsTemplated = appendUnique(fc.AuthFieldsTemplated,
						strings.ToLower(key[1]))
				}
			}
		}
	}
	return fc
}

// Neutralize makes a Helm/Go-templated manifest parse as YAML without inventing
// values. Value actions ({{ .Values.issuer }}) become the sentinel Placeholder;
// structural control lines ({{- if x }}, {{ end }}) are removed entirely. It is a
// best-effort resolver for standalone files that cannot be `helm template`-d
// because their chart context was not fetched.
func Neutralize(b []byte) []byte {
	if !tmplRe.Match(b) {
		return b
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.Contains(line, "{{") {
			out = append(out, line)
			continue
		}
		resolved := tmplRe.ReplaceAllStringFunc(line, func(m string) string {
			inner := strings.Trim(m, "{}-")
			if controlKW[firstWord(inner)] {
				return "" // control action emits nothing
			}
			return Placeholder
		})
		// If nothing but list dashes / whitespace remains, the line was pure
		// control flow (e.g. "{{- if .enabled }}") — drop it so it can't break
		// the YAML. A value line ("issuer: {{ ... }}") keeps its placeholder.
		if strings.TrimSpace(strings.TrimLeft(resolved, " \t-")) == "" {
			continue
		}
		out = append(out, resolved)
	}
	return []byte(strings.Join(out, "\n"))
}

// RenderDir renders a real chart/kustomize directory to plain YAML by shelling out
// to the standard tools. It is the correct path when the crawler has fetched a
// whole tree. Returns a clear error (not a fake) when the tool is unavailable or
// the directory is not a renderable unit, so callers can fall back to Neutralize.
func RenderDir(dir string) ([]byte, error) {
	switch {
	case fileExists(filepath.Join(dir, "kustomization.yaml")),
		fileExists(filepath.Join(dir, "kustomization.yml")):
		bin, err := exec.LookPath("kustomize")
		if err != nil {
			return nil, fmt.Errorf("kustomize not on PATH: %w", err)
		}
		return runRenderer(bin, "build", dir)
	case fileExists(filepath.Join(dir, "Chart.yaml")):
		bin, err := exec.LookPath("helm")
		if err != nil {
			return nil, fmt.Errorf("helm not on PATH: %w", err)
		}
		return runRenderer(bin, "template", dir)
	default:
		return nil, fmt.Errorf("%s is not a chart or kustomize directory", dir)
	}
}

// Report is the corpus-wide templating coverage, replacing the paper's estimated
// "~17% Helm, ~2% kustomize" with a computed, reproducible number.
type Report struct {
	Files              int `json:"files"`
	Plain              int `json:"plain"`
	Helm               int `json:"helm_templated"`
	Kustomize          int `json:"kustomize"`
	AuthFieldTemplated int `json:"auth_field_templated"` // files with a templated issuer/aud/claim
	Neutralized        int `json:"neutralized"`          // files rewritten so they parse
	RenderedTrees      int `json:"rendered_trees"`       // dirs rendered via helm/kustomize
}

// PrepareCorpus walks srcDir, resolves templating, and writes a discovery-ready
// copy into dstDir (which it creates), returning the coverage Report. Plain files
// are copied verbatim; templated files are neutralized. It never drops a file: an
// under-resolved auth config still contributes to the denominator, tagged in the
// Report so the measurement can caveat it honestly.
func PrepareCorpus(srcDir, dstDir string) (Report, error) {
	var rep Report
	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		return rep, err
	}
	err := filepath.WalkDir(srcDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		low := strings.ToLower(p)
		if !strings.HasSuffix(low, ".yaml") && !strings.HasSuffix(low, ".yml") {
			return nil
		}
		b, rerr := os.ReadFile(p) // #nosec G304 -- caller-provided corpus dir
		if rerr != nil {
			return nil // a single unreadable file must not abort the run
		}
		rep.Files++
		fc := Classify(p, b)
		switch fc.Engine {
		case EngineKustomize:
			rep.Kustomize++
		case EngineHelm:
			rep.Helm++
		default:
			rep.Plain++
		}
		if len(fc.AuthFieldsTemplated) > 0 {
			rep.AuthFieldTemplated++
		}
		outBytes := b
		if fc.Templated {
			outBytes = Neutralize(b)
			rep.Neutralized++
		}
		dst := filepath.Join(dstDir, filepath.Base(p)) // #nosec G304 -- fixed dst dir
		return os.WriteFile(dst, outBytes, 0o600)
	})
	return rep, err
}

func runRenderer(bin string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...) // #nosec G204 -- bin resolved via LookPath, args fixed
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", filepath.Base(bin), strings.Join(args, " "), err)
	}
	return out, nil
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

func appendUnique(xs []string, x string) []string {
	for _, e := range xs {
		if e == x {
			return xs
		}
	}
	return append(xs, x)
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
