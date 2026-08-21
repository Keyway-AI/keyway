package render

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const helmRA = `{{- if .Values.auth.enabled }}
apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata:
  name: {{ .Values.name }}
spec:
  jwtRules:
  - issuer: {{ .Values.oidc.issuer }}
    audiences:
    - {{ .Values.oidc.audience }}
{{- end }}
`

func TestNeutralize(t *testing.T) {
	out := string(Neutralize([]byte(helmRA)))

	if strings.Contains(out, "{{") {
		t.Fatalf("template actions survived neutralization:\n%s", out)
	}
	// Value templates become the sentinel; the resource structure is preserved.
	for _, want := range []string{
		"kind: RequestAuthentication",
		"issuer: " + Placeholder,
		"- " + Placeholder, // templated audience list item kept, not dropped
	} {
		if !strings.Contains(out, want) {
			t.Errorf("neutralized output missing %q:\n%s", want, out)
		}
	}
	// Control-flow lines are dropped entirely (no stray sentinels for them).
	if strings.Contains(out, "if ") || strings.Contains(out, "end") {
		t.Errorf("control-flow line survived:\n%s", out)
	}
	// A templated issuer must NOT read as an absent one: the sentinel is present.
	if strings.Count(out, Placeholder) != 3 {
		t.Errorf("expected 3 placeholders (name, issuer, audience), got %d:\n%s",
			strings.Count(out, Placeholder), out)
	}
}

func TestNeutralizePlainUnchanged(t *testing.T) {
	plain := "kind: RequestAuthentication\nspec:\n  jwtRules:\n  - issuer: https://accounts.example.com\n"
	if got := string(Neutralize([]byte(plain))); got != plain {
		t.Errorf("plain YAML was modified:\ngot:  %q\nwant: %q", got, plain)
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		body       string
		wantEngine Engine
		wantTmpl   bool
		wantAuth   []string
	}{
		{"plain", "ra.yaml", "kind: RequestAuthentication\nspec:\n  jwtRules:\n  - issuer: https://x\n", EnginePlain, false, nil},
		{"helm-issuer", "ra.yaml", "spec:\n  jwtRules:\n  - issuer: {{ .Values.iss }}\n", EngineHelm, true, []string{"issuer"}},
		{"kustomization-name", "kustomization.yaml", "resources:\n- ra.yaml\n", EngineKustomize, false, nil},
		{"kustomize-apiversion", "k.yaml", "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n", EngineKustomize, false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fc := Classify(tc.file, []byte(tc.body))
			if fc.Engine != tc.wantEngine {
				t.Errorf("engine = %q, want %q", fc.Engine, tc.wantEngine)
			}
			if fc.Templated != tc.wantTmpl {
				t.Errorf("templated = %v, want %v", fc.Templated, tc.wantTmpl)
			}
			if strings.Join(fc.AuthFieldsTemplated, ",") != strings.Join(tc.wantAuth, ",") {
				t.Errorf("authFields = %v, want %v", fc.AuthFieldsTemplated, tc.wantAuth)
			}
		})
	}
}

func TestPrepareCorpus(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(src, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("plain__ra.yaml", "kind: RequestAuthentication\nspec:\n  jwtRules:\n  - issuer: https://x\n")
	write("chart__ra.yaml", helmRA)
	write("kustomization.yaml", "resources:\n- ra.yaml\n")
	write("notyaml.txt", "ignored")

	rep, err := PrepareCorpus(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Files != 3 { // the .txt is skipped
		t.Errorf("Files = %d, want 3", rep.Files)
	}
	if rep.Helm != 1 || rep.Plain != 1 || rep.Kustomize != 1 {
		t.Errorf("classification = helm %d / plain %d / kustomize %d, want 1/1/1",
			rep.Helm, rep.Plain, rep.Kustomize)
	}
	if rep.AuthFieldTemplated != 1 {
		t.Errorf("AuthFieldTemplated = %d, want 1", rep.AuthFieldTemplated)
	}
	if rep.Neutralized != 1 {
		t.Errorf("Neutralized = %d, want 1", rep.Neutralized)
	}
	// The helm file in dst must be template-free and keep its structure.
	got, err := os.ReadFile(filepath.Join(dst, "chart__ra.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "{{") {
		t.Errorf("prepared helm file still contains template actions:\n%s", got)
	}
	if !strings.Contains(string(got), "kind: RequestAuthentication") {
		t.Errorf("prepared helm file lost its resource:\n%s", got)
	}
}

// TestRenderDir exercises the real helm/kustomize shell-out path. It skips when
// the tool is not installed, so CI without helm/kustomize stays green while a
// developer with them gets real coverage.
func TestRenderDir(t *testing.T) {
	if _, err := exec.LookPath("kustomize"); err != nil {
		t.Skip("kustomize not installed; skipping real-render path")
	}
	dir := t.TempDir()
	must := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	must("ra.yaml", "apiVersion: security.istio.io/v1\nkind: RequestAuthentication\nmetadata:\n  name: r\n")
	must("kustomization.yaml", "resources:\n- ra.yaml\n")

	out, err := RenderDir(dir)
	if err != nil {
		t.Fatalf("RenderDir: %v", err)
	}
	if !strings.Contains(string(out), "RequestAuthentication") {
		t.Errorf("rendered output missing the resource:\n%s", out)
	}
}

func TestRenderDirNotARenderableUnit(t *testing.T) {
	if _, err := RenderDir(t.TempDir()); err == nil {
		t.Error("expected an error for a directory with no chart/kustomization, got nil")
	}
}
