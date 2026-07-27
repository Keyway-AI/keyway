package probe

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nometria/keyway/internal/attack"
	"github.com/nometria/keyway/internal/issuer/generic"
	"github.com/nometria/keyway/internal/model"
)

// TestRunHarness_EndToEnd stands up a real issuer and fires the generative attack
// corpus through the probe engine at two targets: a correctly-configured verifier
// (which must reject every attack, accept the control) and a broken accept-all
// verifier (which must be flagged on every attack). This proves the harness works
// live, with claim attacks signed by the real issuer.
func TestRunHarness_EndToEnd(t *testing.T) {
	const issURL = "https://issuer.keyway.test"
	const aud = "payments-api"

	iss, err := generic.New(generic.Options{Name: "keyway", IssuerURL: issURL})
	if err != nil {
		t.Fatal(err)
	}
	issModel, err := iss.Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	mint := func(kid string, claims map[string]any) (string, error) {
		return iss.MintToken(context.Background(), kid, claims)
	}

	// The correct verifier: pinned RS256 against the issuer's public key + claims.
	pub := activePublicKey(t, issModel)
	oracle := attack.Oracle{Policy: attack.Policy{
		TrustedKey: pub, Issuer: issURL, Audience: aud,
		RequiredClaims: []string{"scope"}, Now: time.Now(), Skew: 60 * time.Second,
	}}

	correct := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if v, _ := oracle.Verify(r.Context(), tok); v == attack.Accept {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer correct.Close()

	vulnerable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK) // honors any token — the classic "decode, never verify"
	}))
	defer vulnerable.Close()

	cfg := DefaultEngineConfig()
	cfg.AllowProduction = true // httptest runs on 127.0.0.1
	cfg.InterProbeDelay = 0
	eng := NewEngine(cfg)

	t.Run("correct target has no findings", func(t *testing.T) {
		results, _, err := eng.RunHarness(context.Background(), issModel, mint, []model.Consumer{consumerFor(correct.URL, issURL, aud)})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) < 15 {
			t.Fatalf("expected a full corpus run, got %d results", len(results))
		}
		for _, r := range results {
			if !r.Passed {
				t.Errorf("correct target: %s should have been handled correctly (status %d)", r.ProbeID, r.StatusCode)
			}
		}
	})

	t.Run("vulnerable target is flagged on every attack", func(t *testing.T) {
		results, _, err := eng.RunHarness(context.Background(), issModel, mint, []model.Consumer{consumerFor(vulnerable.URL, issURL, aud)})
		if err != nil {
			t.Fatal(err)
		}
		attacks, findings, controlOK := 0, 0, false
		for _, r := range results {
			if strings.HasPrefix(r.ProbeID, "harness:CONTROL:") {
				controlOK = r.Passed
				continue
			}
			attacks++
			if !r.Passed {
				findings++
			}
		}
		if !controlOK {
			t.Error("control token should be accepted by the accept-all target")
		}
		if findings != attacks || attacks == 0 {
			t.Fatalf("expected every attack (%d) flagged against the accept-all target, got %d", attacks, findings)
		}
	})
}

func consumerFor(url, issURL, aud string) model.Consumer {
	return model.Consumer{
		StableID:  "test://" + url,
		Name:      "test-consumer",
		Probeable: true,
		Endpoints: []model.Endpoint{{URL: url, Method: http.MethodGet}},
		Expects: model.Expectations{
			Issuers:        []string{issURL},
			Audiences:      []string{aud},
			RequiredClaims: []string{"scope"},
			Algorithms:     []string{"RS256"},
		},
	}
}

func activePublicKey(t *testing.T, iss model.Issuer) *rsa.PublicKey {
	t.Helper()
	for _, k := range iss.Keys {
		if k.Status == model.KeyActive && k.PublicKeyPEM != "" {
			block, _ := pem.Decode([]byte(k.PublicKeyPEM))
			if block == nil {
				t.Fatal("could not PEM-decode the active public key")
			}
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			rsaPub, ok := pub.(*rsa.PublicKey)
			if !ok {
				t.Fatalf("active key is not RSA: %T", pub)
			}
			return rsaPub
		}
	}
	t.Fatal("no active public key on the issuer")
	return nil
}
