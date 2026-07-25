package keystore

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/nometria/keyway/internal/issuer/localkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func key32(seed byte) []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = seed + byte(i)
	}
	return b
}

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFileStore(dir, key32(1))
	require.NoError(t, err)

	keys := []localkeys.PersistedKey{
		{KID: "kid-a", Alg: "RS256", Status: "announced", PrivatePEM: "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----"},
	}
	require.NoError(t, s.Save("my-issuer", keys))

	got, err := s.Load("my-issuer")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "kid-a", got[0].KID)
	assert.Equal(t, "announced", got[0].Status)

	// The file on disk must be encrypted — no plaintext key material leaks.
	raw, err := os.ReadFile(filepath.Join(dir, "my-issuer.enc"))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "PRIVATE KEY")
	assert.NotContains(t, string(raw), "announced")
}

func TestFileStoreWrongKeyFails(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFileStore(dir, key32(1))
	require.NoError(t, err)
	require.NoError(t, s.Save("iss", []localkeys.PersistedKey{{KID: "x", PrivatePEM: "p"}}))

	// A store with a different key cannot decrypt (GCM auth fails).
	other, err := NewFileStore(dir, key32(9))
	require.NoError(t, err)
	_, err = other.Load("iss")
	require.Error(t, err)
}

// TestFileStoreConcurrentSaves stresses the atomic-write path: many concurrent
// Saves to the same issuer must all succeed (no lost rename) and leave a valid,
// decryptable file. Guards the KI-09 concurrency fix.
func TestFileStoreConcurrentSaves(t *testing.T) {
	s, err := NewFileStore(t.TempDir(), key32(2))
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			keys := []localkeys.PersistedKey{{KID: "k", Alg: "RS256", Status: "active", PrivatePEM: strings.Repeat("x", n+1)}}
			if err := s.Save("iss", keys); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("concurrent Save failed: %v", e)
	}
	got, err := s.Load("iss")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "k", got[0].KID)
}

func TestFileStoreEmptyLoad(t *testing.T) {
	s, err := NewFileStore(t.TempDir(), key32(1))
	require.NoError(t, err)
	got, err := s.Load("never-saved")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestNewFileStoreRejectsShortKey(t *testing.T) {
	_, err := NewFileStore(t.TempDir(), []byte("too-short"))
	require.Error(t, err)
}

func TestKeyFromEnv(t *testing.T) {
	// 32-byte hex key decodes.
	t.Setenv("KW_TEST_KEY", hex.EncodeToString(key32(3)))
	k, err := KeyFromEnv("KW_TEST_KEY")
	require.NoError(t, err)
	assert.Len(t, k, 32)

	// Unset -> fail closed.
	_, err = KeyFromEnv("KW_TEST_UNSET")
	require.Error(t, err)

	// Wrong length -> error.
	t.Setenv("KW_TEST_KEY", "abcd")
	_, err = KeyFromEnv("KW_TEST_KEY")
	require.Error(t, err)
}

func TestSanitize(t *testing.T) {
	assert.Equal(t, "https___kc_realms_main", sanitize("https://kc/realms/main"))
	assert.Equal(t, "issuer", sanitize(""))
	assert.False(t, strings.ContainsAny(sanitize("a/b\\c"), `/\`))
}
