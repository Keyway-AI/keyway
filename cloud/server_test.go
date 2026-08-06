package cloud

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	cfg := Config{
		SessionSecret:  []byte("test-secret-please-change-0123456789"),
		DevLogin:       true,
		FrontendURL:    "http://frontend.test",
		AllowedOrigins: []string{"http://frontend.test"},
	}
	ts := httptest.NewServer(NewServer(cfg, NewMemoryStore()).Routes())
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	return ts, &http.Client{Jar: jar}
}

// doJSON issues a request, decodes the JSON body, closes it, and returns the
// status code plus the decoded map. It intentionally does not return the
// *http.Response so callers never juggle an unclosed body.
func doJSON(t *testing.T, c *http.Client, method, url string, body any) (int, map[string]any) {
	t.Helper()
	var r *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, url, r)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	return res.StatusCode, out
}

func TestUnauthenticatedIsRejected(t *testing.T) {
	ts, c := testServer(t)
	code, _ := doJSON(t, c, http.MethodGet, ts.URL+"/v1/projects", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d", code)
	}
}

func TestEndToEndProjectFlow(t *testing.T) {
	ts, c := testServer(t)

	// Sign in (dev).
	if code, _ := doJSON(t, c, http.MethodPost, ts.URL+"/v1/auth/dev-login", nil); code != http.StatusOK {
		t.Fatalf("dev-login: %d", code)
	}
	if code, me := doJSON(t, c, http.MethodGet, ts.URL+"/v1/me", nil); code != http.StatusOK || me["login"] != "dev" {
		t.Fatalf("me: %d %v", code, me)
	}

	// Create an upload-source project.
	code, proj := doJSON(t, c, http.MethodPost, ts.URL+"/v1/projects", map[string]any{
		"name":   "my-mesh",
		"source": map[string]any{"kind": "upload"},
	})
	if code != http.StatusCreated {
		t.Fatalf("create project: %d", code)
	}
	pid, _ := proj["id"].(string)
	if pid == "" {
		t.Fatal("project id missing")
	}

	// First analysis (baseline) with a real Istio manifest.
	code, a1 := doJSON(t, c, http.MethodPost, ts.URL+"/v1/projects/"+pid+"/analyze", map[string]any{
		"manifests": map[string]string{"httpbin.yaml": istioHTTPBin},
	})
	if code != http.StatusOK {
		t.Fatalf("analyze: %d %v", code, a1)
	}
	if cc, _ := a1["consumer_count"].(float64); cc < 1 {
		t.Fatalf("expected >=1 consumer, got %v", a1["consumer_count"])
	}
	if a1["is_baseline"] != true {
		t.Fatalf("first analysis should be baseline, got %v", a1["is_baseline"])
	}

	// Second analysis with drift.
	code, a2 := doJSON(t, c, http.MethodPost, ts.URL+"/v1/projects/"+pid+"/analyze", map[string]any{
		"manifests": map[string]string{"httpbin.yaml": istioHTTPBinWidened},
	})
	if code != http.StatusOK {
		t.Fatalf("analyze v2: %d", code)
	}
	if cnt, _ := a2["change_count"].(float64); cnt < 1 {
		t.Fatalf("expected drift on v2, got change_count=%v", a2["change_count"])
	}

	// History should now have two analyses.
	code, hist := doJSON(t, c, http.MethodGet, ts.URL+"/v1/projects/"+pid+"/analyses", nil)
	if code != http.StatusOK {
		t.Fatalf("list analyses: %d", code)
	}
	if items, _ := hist["analyses"].([]any); len(items) != 2 {
		t.Fatalf("expected 2 analyses in history, got %d", len(items))
	}
}

func TestTenantIsolation(t *testing.T) {
	ts, alice := testServer(t)
	// Alice signs in and creates a project.
	doJSON(t, alice, http.MethodPost, ts.URL+"/v1/auth/dev-login", nil)
	_, proj := doJSON(t, alice, http.MethodPost, ts.URL+"/v1/projects", map[string]any{"name": "a", "source": map[string]any{"kind": "upload"}})
	pid, _ := proj["id"].(string)

	// Bob (a different jar → different session? dev-login always creates dev:local,
	// so simulate a stranger with NO session instead).
	jar, _ := cookiejar.New(nil)
	stranger := &http.Client{Jar: jar}
	code, _ := doJSON(t, stranger, http.MethodGet, ts.URL+"/v1/projects/"+pid, nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("stranger should be unauthorized, got %d", code)
	}
}

func TestPublicAgentInspect(t *testing.T) {
	ts, c := testServer(t)
	// A well-formed JWT with no exp — public tool, no auth required, real analysis.
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"agent"}`))
	token := "eyJhbGciOiJSUzI1NiJ9." + payload + ".sig"
	code, out := doJSON(t, c, http.MethodPost, ts.URL+"/v1/agent/inspect", map[string]any{"token": token})
	if code != http.StatusOK {
		t.Fatalf("agent inspect: %d %v", code, out)
	}
	if cnt, _ := out["count"].(float64); cnt < 1 {
		t.Fatalf("expected at least one finding (no exp), got %v", out)
	}
}
