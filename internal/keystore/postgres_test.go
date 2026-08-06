package keystore

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Keyway-AI/keyway/internal/issuer/localkeys"
	"github.com/Keyway-AI/keyway/internal/store/postgres"
)

// testPool returns a pgx pool against KEYWAY_TEST_DB (skipping when unset) with
// the schema migrated so keyway_operated_keys exists.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("KEYWAY_TEST_DB")
	if dsn == "" {
		t.Skip("KEYWAY_TEST_DB not set; skipping Postgres integration test")
	}
	if err := postgres.MigrateUp(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func sampleKeys() []localkeys.PersistedKey {
	return []localkeys.PersistedKey{{
		KID:             "rsa-test-1",
		Alg:             "RS256",
		Status:          "active",
		PrivatePEM:      "-----BEGIN PRIVATE KEY-----\nSUPERSECRETMATERIAL\n-----END PRIVATE KEY-----",
		FirstSeenInJWKS: time.Now().UTC().Truncate(time.Second),
	}}
}

func TestPostgresStore_RoundTripAndSharing(t *testing.T) {
	pool := testPool(t)
	enc := key32(0x11)
	issuer := "it-" + time.Now().Format("150405.000000")

	s1, err := NewPostgresStore(pool, enc)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// Absent issuer → nil, no error.
	if got, err := s1.Load(issuer); err != nil || got != nil {
		t.Fatalf("expected empty load, got=%v err=%v", got, err)
	}

	keys := sampleKeys()
	if err := s1.Save(issuer, keys); err != nil {
		t.Fatalf("save: %v", err)
	}

	// A SECOND store instance (simulating another replica) sees the same keys.
	s2, err := NewPostgresStore(pool, enc)
	if err != nil {
		t.Fatalf("new store 2: %v", err)
	}
	got, err := s2.Load(issuer)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].KID != "rsa-test-1" || got[0].PrivatePEM != keys[0].PrivatePEM {
		t.Fatalf("cross-instance load mismatch: %+v", got)
	}
}

// TestPostgresStore_EncryptedAtRest verifies the stored bytes are ciphertext (no
// plaintext private material) and that a wrong key cannot decrypt them.
func TestPostgresStore_EncryptedAtRest(t *testing.T) {
	pool := testPool(t)
	issuer := "enc-" + time.Now().Format("150405.000000")

	s, err := NewPostgresStore(pool, key32(0x22))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := s.Save(issuer, sampleKeys()); err != nil {
		t.Fatalf("save: %v", err)
	}

	// The raw column must not contain the plaintext private material.
	var raw []byte
	if err := pool.QueryRow(context.Background(),
		`SELECT data FROM keyway_operated_keys WHERE issuer = $1`, issuer).Scan(&raw); err != nil {
		t.Fatalf("read raw: %v", err)
	}
	if bytes.Contains(raw, []byte("SUPERSECRETMATERIAL")) {
		t.Fatal("private material found in plaintext in the stored row")
	}

	// A store with a different key must fail to decrypt.
	wrong, err := NewPostgresStore(pool, key32(0x33))
	if err != nil {
		t.Fatalf("new wrong-key store: %v", err)
	}
	if _, err := wrong.Load(issuer); err == nil {
		t.Fatal("expected decrypt failure under the wrong key")
	}
}
