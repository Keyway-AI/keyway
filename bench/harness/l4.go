package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Keyway-AI/keyway/internal/attribution"
	"github.com/Keyway-AI/keyway/internal/model"
)

// l4Score exercises the attribution layer (L4): it builds a small realistic
// fixture (a git repo with a committed policy and a manifest carrying a deploy
// change-cause) and checks the chain attributor binds each change to the right
// cause. Returns a Scorecard keyed "L4" (PRD §13.4 gate: fail below 60%).
func l4Score() (Scorecard, error) {
	dir, err := os.MkdirTemp("", "keyway-l4-")
	if err != nil {
		return Scorecard{}, err
	}
	defer os.RemoveAll(dir)

	// A committed policy file (git-attributable).
	policy := "kind: RequestAuthentication\nmetadata:\n  name: api-a\n"
	if err := os.WriteFile(filepath.Join(dir, "policy.yaml"), []byte(policy), 0o644); err != nil {
		return Scorecard{}, err
	}
	if err := gitInit(dir); err != nil {
		return Scorecard{}, err
	}

	// An uncommitted manifest with a deploy change-cause (deploy-attributable).
	deploy := "kind: Deployment\nmetadata:\n  name: api-a\n  annotations:\n    kubernetes.io/change-cause: \"rollout by CI #918\"\n"
	if err := os.WriteFile(filepath.Join(dir, "deploy.yaml"), []byte(deploy), 0o644); err != nil {
		return Scorecard{}, err
	}

	chain := attribution.NewChain(attribution.NewGit(dir), attribution.NewK8sDeploy(dir))

	type tc struct {
		evidence []string
		wantKind string
	}
	cases := []tc{
		{[]string{"policy.yaml"}, "commit"},            // committed → git
		{[]string{"deploy.yaml"}, "deploy"},            // annotated manifest → deploy
		{[]string{"probe:canary_key"}, "unattributed"}, // marker only → unattributed
	}

	card := Scorecard{Layer: "L4"}
	for _, c := range cases {
		attr, aerr := chain.Attribute(context.Background(), model.ChangeEvent{Evidence: c.evidence})
		if aerr != nil {
			card.FN++
			continue
		}
		if attr.Kind == c.wantKind {
			card.TP++
		} else {
			card.FN++
		}
	}
	card.Compute()
	return card, nil
}

func gitInit(dir string) error {
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=Alice", "GIT_AUTHOR_EMAIL=alice@example.com",
		"GIT_COMMITTER_NAME=Alice", "GIT_COMMITTER_EMAIL=alice@example.com")
	for _, args := range [][]string{
		{"init"}, {"config", "user.name", "Alice"}, {"config", "user.email", "alice@example.com"},
		{"add", "."}, {"commit", "-m", "add auth policy"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
