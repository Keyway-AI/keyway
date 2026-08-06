package coordination

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Keyway-AI/keyway/internal/store/postgres"
)

// testDSN returns the Postgres DSN from KEYWAY_TEST_DB, skipping the test when
// unset so `go test ./...` stays green without a database. It runs migrations so
// the coordination tables exist. Set e.g.
//
//	KEYWAY_TEST_DB=postgres://keyway:keyway@localhost:5442/keyway?sslmode=disable
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("KEYWAY_TEST_DB")
	if dsn == "" {
		t.Skip("KEYWAY_TEST_DB not set; skipping Postgres integration test")
	}
	if err := postgres.MigrateUp(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return dsn
}

func TestPGIdempotency_RoundTripAndExpiry(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c, err := Open(ctx, dsn, time.Hour)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer c.Close()
	s := c.Idempotency()

	key := "it-" + time.Now().Format("150405.000000")
	if _, ok, err := s.Get(ctx, key); err != nil || ok {
		t.Fatalf("expected miss, ok=%v err=%v", ok, err)
	}
	rec := Record{Status: 201, Body: []byte(`{"v":1}`), ContentType: "application/json"}
	if err := s.Put(ctx, key, rec); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, ok, err := s.Get(ctx, key)
	if err != nil || !ok {
		t.Fatalf("expected hit, ok=%v err=%v", ok, err)
	}
	if got.Status != 201 || string(got.Body) != `{"v":1}` || got.ContentType != "application/json" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	// A store with a negative TTL writes an already-expired row → Get must miss.
	expired, err := Open(ctx, dsn, -time.Second)
	if err != nil {
		t.Fatalf("open expired: %v", err)
	}
	defer expired.Close()
	ekey := "exp-" + time.Now().Format("150405.000000")
	if err := expired.Idempotency().Put(ctx, ekey, rec); err != nil {
		t.Fatalf("put expired: %v", err)
	}
	if _, ok, _ := expired.Idempotency().Get(ctx, ekey); ok {
		t.Fatal("expected expired row to miss")
	}
}

// TestPGLeader_MutualExclusion verifies two coordinators contending for the same
// advisory lock: exactly one is leader, and releasing it lets the other acquire.
func TestPGLeader_MutualExclusion(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	a, err := Open(ctx, dsn, time.Hour)
	if err != nil {
		t.Fatalf("open a: %v", err)
	}
	b, err := Open(ctx, dsn, time.Hour)
	if err != nil {
		a.Close()
		t.Fatalf("open b: %v", err)
	}
	defer b.Close()

	if !a.Leader().IsLeader(ctx) {
		a.Close()
		t.Fatal("first coordinator should acquire leadership")
	}
	if b.Leader().IsLeader(ctx) {
		a.Close()
		t.Fatal("second coordinator must NOT be leader while the first holds the lock")
	}

	// Releasing A's lock (Close) must let B become leader.
	a.Close()
	// The lock releases when A's connection returns/closes; retry briefly.
	var bLeads bool
	for i := 0; i < 20; i++ {
		if b.Leader().IsLeader(ctx) {
			bLeads = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !bLeads {
		t.Fatal("second coordinator should acquire leadership after the first releases it")
	}
}
