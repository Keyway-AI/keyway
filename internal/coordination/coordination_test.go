package coordination

import (
	"context"
	"testing"
	"time"
)

func TestMemoryIdempotency_GetPut(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryIdempotency(time.Hour)

	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("expected miss on empty store")
	}
	rec := Record{Status: 200, Body: []byte(`{"ok":true}`), ContentType: "application/json"}
	if err := s.Put(ctx, "k", rec); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, ok, err := s.Get(ctx, "k")
	if err != nil || !ok {
		t.Fatalf("expected hit, ok=%v err=%v", ok, err)
	}
	if got.Status != 200 || string(got.Body) != `{"ok":true}` || got.ContentType != "application/json" {
		t.Fatalf("record round-trip mismatch: %+v", got)
	}
}

func TestMemoryIdempotency_Expiry(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryIdempotency(-time.Second) // already expired on write
	_ = s.Put(ctx, "k", Record{Status: 201})
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("expected expired entry to miss")
	}
}

func TestMemoryIdempotency_FIFOCap(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryIdempotency(time.Hour).(*memoryIdempotency)
	s.max = 3
	for i, k := range []string{"a", "b", "c", "d"} {
		_ = s.Put(ctx, k, Record{Status: 200 + i})
	}
	if _, ok, _ := s.Get(ctx, "a"); ok {
		t.Fatal("expected oldest key 'a' to be evicted")
	}
	if _, ok, _ := s.Get(ctx, "d"); !ok {
		t.Fatal("expected newest key 'd' to be present")
	}
}

func TestLocalLeaderAlwaysLeads(t *testing.T) {
	c := NewMemory(time.Hour)
	defer c.Close()
	if !c.Leader().IsLeader(context.Background()) {
		t.Fatal("local leader should always be leader")
	}
}

func TestOpenMemoryDSN(t *testing.T) {
	c, err := Open(context.Background(), "memory", time.Hour)
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	defer c.Close()
	if c.Idempotency() == nil || c.Leader() == nil {
		t.Fatal("memory coordinator must provide both seams")
	}
}
