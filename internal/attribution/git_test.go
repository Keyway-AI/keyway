package attribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nometria/keyway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitInit(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Alice", "GIT_AUTHOR_EMAIL=alice@example.com",
			"GIT_COMMITTER_NAME=Alice", "GIT_COMMITTER_EMAIL=alice@example.com")
		require.NoError(t, cmd.Run(), "git %v", args)
	}
	run("init")
	run("config", "user.name", "Alice")
	run("config", "user.email", "alice@example.com")

	path := filepath.Join(dir, "istio.yaml")
	require.NoError(t, os.WriteFile(path, []byte("kind: RequestAuthentication\n"), 0o644))
	run("add", ".")
	run("commit", "-m", "add auth policy")

	// Return the repo-relative path used in evidence.
	return dir, "istio.yaml"
}

func TestGitAttribute(t *testing.T) {
	dir, rel := gitInit(t)
	g := NewGit(dir)

	ev := model.ChangeEvent{Evidence: []string{rel}}
	attr, err := g.Attribute(context.Background(), ev)
	require.NoError(t, err)
	require.NotNil(t, attr)
	assert.Equal(t, "commit", attr.Kind)
	assert.Equal(t, "Alice", attr.Actor)
	assert.NotEmpty(t, attr.Ref)
	assert.InDelta(t, 0.9, attr.Confidence, 0.001)
}

func TestGitAttributeUnattributed(t *testing.T) {
	dir, _ := gitInit(t)
	g := NewGit(dir)

	// Evidence with only markers -> no file path -> unattributed.
	ev := model.ChangeEvent{Evidence: []string{"probe:canary_key", "lib:keyfunc v1.9.0"}}
	attr, err := g.Attribute(context.Background(), ev)
	require.NoError(t, err)
	assert.Equal(t, "unattributed", attr.Kind)
}
