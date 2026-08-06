package attribution

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/Keyway-AI/keyway/internal/model"
)

// GitAttributor binds a change event to the last git commit that touched the
// file named in its provenance evidence (PRD §16 OPEN-5: v1 covers git).
type GitAttributor struct {
	repoDir string
}

// NewGit constructs a git attributor rooted at repoDir.
func NewGit(repoDir string) *GitAttributor { return &GitAttributor{repoDir: repoDir} }

var _ Attributor = (*GitAttributor)(nil)

// Attribute finds the last commit touching a file locator in the event's
// evidence. It returns Unattributed when no file path is present or git yields
// nothing.
func (g *GitAttributor) Attribute(ctx context.Context, ev model.ChangeEvent) (*model.Attribution, error) {
	path := filePathFrom(ev.Evidence)
	if path == "" {
		return Unattributed(), nil
	}
	// %H commit, %an author, %aI author date (ISO-8601 strict).
	out, err := exec.CommandContext(ctx, "git", "-C", g.repoDir,
		"log", "-1", "--format=%H%x1f%an%x1f%aI", "--", path).Output()
	if err != nil {
		return Unattributed(), nil
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return Unattributed(), nil
	}
	parts := strings.Split(line, "\x1f")
	if len(parts) < 3 {
		return Unattributed(), nil
	}
	ts, _ := time.Parse(time.RFC3339, parts[2])
	return &model.Attribution{
		Kind:       "commit",
		Ref:        parts[0],
		Actor:      parts[1],
		Timestamp:  ts,
		Confidence: 0.9,
	}, nil
}

// filePathFrom returns the first evidence entry that looks like a file path
// (i.e. not a "probe:"/"lib:"/"source:" marker).
func filePathFrom(evidence []string) string {
	for _, e := range evidence {
		if strings.ContainsAny(e, "/\\.") &&
			!strings.HasPrefix(e, "probe:") &&
			!strings.HasPrefix(e, "lib:") &&
			!strings.HasPrefix(e, "source:") &&
			!strings.HasPrefix(e, "expects.") &&
			!strings.HasPrefix(e, "jwks_behavior.") {
			return e
		}
	}
	return ""
}
