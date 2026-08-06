package cloud

import (
	"context"
	"testing"
)

const istioHTTPBin = `apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata:
  name: httpbin
  namespace: foo
spec:
  selector:
    matchLabels:
      app: httpbin
  jwtRules:
  - issuer: "testing@secure.istio.io"
    audiences:
    - "api"
`

// widened audience — a real, meaningful drift the diff must catch.
const istioHTTPBinWidened = `apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata:
  name: httpbin
  namespace: foo
spec:
  selector:
    matchLabels:
      app: httpbin
  jwtRules:
  - issuer: "testing@secure.istio.io"
    audiences:
    - "api"
    - "internal-admin"
`

func TestAnalyzeDiscoversConsumers(t *testing.T) {
	v, changes, err := Analyze(context.Background(), map[string]string{"httpbin.yaml": istioHTTPBin}, nil)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(v.Consumers) == 0 {
		t.Fatalf("expected at least one consumer, got 0")
	}
	if v.Hash == "" {
		t.Fatal("expected a contract hash")
	}
	if changes != nil {
		t.Fatalf("first run should have no drift, got %d changes", len(changes))
	}
}

func TestAnalyzeIsDeterministic(t *testing.T) {
	m := map[string]string{"httpbin.yaml": istioHTTPBin}
	a, _, err := Analyze(context.Background(), m, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := Analyze(context.Background(), m, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Hash != b.Hash {
		t.Fatalf("same config should hash identically: %s != %s", a.Hash, b.Hash)
	}
}

func TestAnalyzeDetectsDrift(t *testing.T) {
	ctx := context.Background()
	v1, _, err := Analyze(ctx, map[string]string{"httpbin.yaml": istioHTTPBin}, nil)
	if err != nil {
		t.Fatal(err)
	}
	v2, changes, err := Analyze(ctx, map[string]string{"httpbin.yaml": istioHTTPBinWidened}, &v1)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Hash == v1.Hash {
		t.Fatal("widened audience should change the contract hash")
	}
	if len(changes) == 0 {
		t.Fatal("expected a drift event for the widened audience")
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	if _, _, err := Analyze(context.Background(), nil, nil); err == nil {
		t.Fatal("expected an error for no manifests")
	}
}
