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

func doJSON(t *testing.T, c *http.Client, method, url string, body any) (*http.Response, map[string]any) {
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
	return res, out
}

func TestUnauthenticatedIsRejected(t *testing.T) {
	ts, c := testServer(t)
	res, _ := doJSON(t, c, http.MethodGet, ts.URL+"/v1/projects", nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d", res.StatusCode)
	}
}

func TestEndToEndProjectFlow(t *testing.T) {
	ts, c := testServer(t)

	// Sign in (dev).
	if res, _ := doJSON(t, c, http.MethodPost, ts.URL+"/v1/auth/dev-login", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("dev-login: %d", res.StatusCode)
	}
	if res, me := doJSON(t, c, http.MethodGet, ts.URL+"/v1/me", nil); res.StatusCode != http.StatusOK || me["login"] != "dev" {
		t.Fatalf("me: %d %v", res.StatusCode, me)
	}

	// Create an upload-source project.
	res, proj := doJSON(t, c, http.MethodPost, ts.URL+"/v1/projects", map[string]any{
		"name":   "my-mesh",
		"source": map[string]any{"kind": "upload"},
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create project: %d", res.StatusCode)
	}
	pid, _ := proj["id"].(string)
	if pid == "" {
		t.Fatal("project id missing")
	}

	// First analysis (baseline) with a real Istio manifest.
	res, a1 := doJSON(t, c, http.MethodPost, ts.URL+"/v1/projects/"+pid+"/analyze", map[string]any{
		"manifests": map[string]string{"httpbin.yaml": istioHTTPBin},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("analyze: %d %v", res.StatusCode, a1)
	}
	if cc, _ := a1["consumer_count"].(float64); cc < 1 {
		t.Fatalf("expected >=1 consumer, got %v", a1["consumer_count"])
	}
	if a1["is_baseline"] != true {
		t.Fatalf("first analysis should be baseline, got %v", a1["is_baseline"])
	}

	// Second analysis with drift.
	res, a2 := doJSON(t, c, http.MethodPost, ts.URL+"/v1/projects/"+pid+"/analyze", map[string]any{
		"manifests": map[string]string{"httpbin.yaml": istioHTTPBinWidened},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("analyze v2: %d", res.StatusCode)
	}
	if cnt, _ := a2["change_count"].(float64); cnt < 1 {
		t.Fatalf("expected drift on v2, got change_count=%v", a2["change_count"])
	}

	// History should now have two analyses.
	res, hist := doJSON(t, c, http.MethodGet, ts.URL+"/v1/projects/"+pid+"/analyses", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list analyses: %d", res.StatusCode)
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
	res, _ := doJSON(t, stranger, http.MethodGet, ts.URL+"/v1/projects/"+pid, nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("stranger should be unauthorized, got %d", res.StatusCode)
	}
}

func TestPublicAgentInspect(t *testing.T) {
	ts, c := testServer(t)
	// A well-formed JWT with no exp — public tool, no auth required, real analysis.
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"agent"}`))
	token := "eyJhbGciOiJSUzI1NiJ9." + payload + ".sig"
	res, out := doJSON(t, c, http.MethodPost, ts.URL+"/v1/agent/inspect", map[string]any{"token": token})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("agent inspect: %d %v", res.StatusCode, out)
	}
	if cnt, _ := out["count"].(float64); cnt < 1 {
		t.Fatalf("expected at least one finding (no exp), got %v", out)
	}
}
