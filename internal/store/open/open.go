// Package open is the store composition helper: it turns a DSN into a
// store.Store, so callers depend on the persistence interface rather than a
// concrete backend. `memory` (or `memory://`) yields the in-memory store for
// tests / offline dev; anything else is treated as a Postgres DSN (migrations
// are run first).
package open

import (
	"context"
	"strings"

	"github.com/Keyway-AI/keyway/internal/store"
	"github.com/Keyway-AI/keyway/internal/store/memory"
	"github.com/Keyway-AI/keyway/internal/store/postgres"
)

// Open returns a store for the DSN plus a cleanup func to call on shutdown.
func Open(ctx context.Context, dsn string) (store.Store, func(), error) {
	if dsn == "memory" || strings.HasPrefix(dsn, "memory:") {
		st := memory.New()
		return st, func() { _ = st.Close() }, nil
	}
	if err := postgres.MigrateUp(dsn); err != nil {
		return nil, nil, err
	}
	st, err := postgres.Open(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	return st, st.Close, nil
}
