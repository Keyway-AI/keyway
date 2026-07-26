package app

import (
	"context"
	"testing"
)

// TestBuildMemory verifies the composition root assembles a working App over the
// in-memory backends (no database), wires the coordination seams, and cleans up.
func TestBuildMemory(t *testing.T) {
	a, err := Build(context.Background(), BuildConfig{DBURL: "memory"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer a.Close()

	if a.Deps.Store == nil {
		t.Fatal("expected a store")
	}
	if a.Idempotency() == nil {
		t.Fatal("expected an idempotency store")
	}
	if a.Leader() == nil || !a.Leader().IsLeader(context.Background()) {
		t.Fatal("single-node app must hold leadership")
	}
}

// TestBuildKeyStorePostgresRequiresDB verifies the postgres key-store mode fails
// fast without a Postgres DB rather than silently degrading.
func TestBuildKeyStorePostgresRequiresDB(t *testing.T) {
	_, err := Build(context.Background(), BuildConfig{
		DBURL:          "memory",
		KeyPersistence: KeyPersistence{Mode: "postgres"},
	})
	if err == nil {
		t.Fatal("expected error: postgres key store without a Postgres DB")
	}
}
