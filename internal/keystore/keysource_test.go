package keystore

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKey_PrecedenceAndDecode(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	hexKey := hex.EncodeToString(raw)

	// File source (hex-encoded).
	dir := t.TempDir()
	fp := filepath.Join(dir, "key")
	if err := os.WriteFile(fp, []byte(hexKey+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveKey(KeySource{File: fp})
	if err != nil {
		t.Fatalf("file source: %v", err)
	}
	if hex.EncodeToString(got) != hexKey {
		t.Fatalf("file key mismatch")
	}

	// Command source takes precedence over file + env.
	t.Setenv("KW_TEST_KEY", "not-a-valid-key")
	got, err = ResolveKey(KeySource{Command: "printf %s " + hexKey, File: fp, Env: "KW_TEST_KEY"})
	if err != nil {
		t.Fatalf("command source: %v", err)
	}
	if hex.EncodeToString(got) != hexKey {
		t.Fatalf("command key mismatch")
	}

	// Env source (hex-encoded).
	t.Setenv("KW_TEST_KEY", hexKey)
	got, err = ResolveKey(KeySource{Env: "KW_TEST_KEY"})
	if err != nil || hex.EncodeToString(got) != hexKey {
		t.Fatalf("env key mismatch: %v", err)
	}

	// No source configured is an error (fail closed).
	if _, err := ResolveKey(KeySource{}); err == nil {
		t.Fatal("expected error when no key source configured")
	}

	// A configured source that yields the wrong length fails.
	if _, err := ResolveKey(KeySource{Command: "printf tooshort"}); err == nil {
		t.Fatal("expected error for short key")
	}
}
