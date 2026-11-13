package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Keyway-AI/keyway/cloud"
)

// testStore opens a store against KEYWAY_TEST_DB, skipping when unset so
// `go test ./...` stays green without a database (same convention as
// internal/store/postgres). Set e.g.
//
//	KEYWAY_TEST_DB=postgres://keyway:keyway@localhost:5439/keyway?sslmode=disable
func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("KEYWAY_TEST_DB")
	if dsn == "" {
		t.Skip("KEYWAY_TEST_DB not set; skipping Postgres integration test")
	}
	require.NoError(t, MigrateUp(dsn))
	st, err := Open(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestUserUpsertPreservesCreatedAt(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	id := "gh:" + uuid.NewString()

	require.NoError(t, st.UpsertUser(ctx, cloud.User{ID: id, Login: "alice", Name: "Alice"}))
	first, err := st.GetUser(ctx, id)
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)
	require.NoError(t, st.UpsertUser(ctx, cloud.User{ID: id, Login: "alice2", Name: "Alice Two"}))
	second, err := st.GetUser(ctx, id)
	require.NoError(t, err)

	assert.Equal(t, "alice2", second.Login, "a repeat login updates the profile")
	assert.Equal(t, first.CreatedAt, second.CreatedAt, "created_at is preserved across logins")

	_, err = st.GetUser(ctx, "gh:does-not-exist")
	assert.ErrorIs(t, err, cloud.ErrNotFound)
}

func TestProjectsAreTenantScoped(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	alice, bob := "gh:"+uuid.NewString(), "gh:"+uuid.NewString()

	pA := cloud.Project{ID: uuid.NewString(), OwnerID: alice, Name: "alice-proj",
		Source: cloud.Source{Kind: cloud.SourceGitHub, Repo: "alice/repo", Ref: "main"}}
	pB := cloud.Project{ID: uuid.NewString(), OwnerID: bob, Name: "bob-proj",
		Source: cloud.Source{Kind: cloud.SourceUpload}}
	require.NoError(t, st.CreateProject(ctx, pA))
	require.NoError(t, st.CreateProject(ctx, pB))

	aList, err := st.ListProjects(ctx, alice)
	require.NoError(t, err)
	require.Len(t, aList, 1, "a tenant sees only its own projects")
	assert.Equal(t, "alice-proj", aList[0].Name)
	assert.Equal(t, cloud.SourceGitHub, aList[0].Source.Kind, "the JSONB source round-trips")
	assert.Equal(t, "alice/repo", aList[0].Source.Repo)

	got, err := st.GetProject(ctx, pA.ID)
	require.NoError(t, err)
	assert.Equal(t, alice, got.OwnerID)
}

func TestAnalysisLifecycleAndCascade(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	owner := "gh:" + uuid.NewString()
	proj := cloud.Project{ID: uuid.NewString(), OwnerID: owner, Name: "p"}
	require.NoError(t, st.CreateProject(ctx, proj))

	base := cloud.Analysis{ID: uuid.NewString(), ProjectID: proj.ID, Hash: "h1", IsBaseline: true,
		TriggerKind: "upload", ConsumerCount: 3, CreatedAt: time.Now().Add(-time.Minute)}
	next := cloud.Analysis{ID: uuid.NewString(), ProjectID: proj.ID, Hash: "h2",
		TriggerKind: "sync", ChangeCount: 2, CreatedAt: time.Now()}
	require.NoError(t, st.SaveAnalysis(ctx, base))
	require.NoError(t, st.SaveAnalysis(ctx, next))

	// SaveAnalysis points the project at the latest analysis.
	p, err := st.GetProject(ctx, proj.ID)
	require.NoError(t, err)
	assert.Equal(t, next.ID, p.LatestAnalysisID)

	list, err := st.ListAnalyses(ctx, proj.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, next.ID, list[0].ID, "newest first")

	latest, err := st.LatestAnalysis(ctx, proj.ID)
	require.NoError(t, err)
	assert.Equal(t, "h2", latest.Hash)

	one, err := st.GetAnalysis(ctx, base.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, one.ConsumerCount, "denormalized counts and the JSONB body round-trip")

	// Deleting the project cascades to its analyses.
	require.NoError(t, st.DeleteProject(ctx, proj.ID))
	_, err = st.GetProject(ctx, proj.ID)
	assert.ErrorIs(t, err, cloud.ErrNotFound)
	_, err = st.GetAnalysis(ctx, base.ID)
	assert.ErrorIs(t, err, cloud.ErrNotFound, "analyses are removed with their project (FK cascade)")

	assert.ErrorIs(t, st.DeleteProject(ctx, "missing"), cloud.ErrNotFound)
}
