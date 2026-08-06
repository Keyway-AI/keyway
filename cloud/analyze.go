package cloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Keyway-AI/keyway/internal/contract"
	"github.com/Keyway-AI/keyway/internal/diff"
	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/discovery/envoy"
	"github.com/Keyway-AI/keyway/internal/discovery/istio"
	"github.com/Keyway-AI/keyway/internal/discovery/k8s"
	"github.com/Keyway-AI/keyway/internal/model"
)

// Analyze runs the engine over a set of manifest files (path → YAML/JSON content)
// and returns the derived contract version plus the drift versus prev (nil for a
// first run). It reuses the exact discovery + contract + diff logic the CLI and
// self-hosted server use — the manifests are written to a throwaway temp dir and
// removed immediately, so nothing is persisted to disk.
func Analyze(ctx context.Context, manifests map[string]string, prev *model.ContractVersion) (model.ContractVersion, []model.ChangeEvent, error) {
	if len(manifests) == 0 {
		return model.ContractVersion{}, nil, fmt.Errorf("no manifests provided")
	}
	dir, err := os.MkdirTemp("", "keyway-cloud-*")
	if err != nil {
		return model.ContractVersion{}, nil, err
	}
	defer os.RemoveAll(dir)

	for name, content := range manifests {
		if err := os.WriteFile(filepath.Join(dir, sanitizeName(name)), []byte(content), 0o600); err != nil {
			return model.ContractVersion{}, nil, err
		}
	}

	consumers, err := discovery.Run(ctx,
		discovery.Scope{ConfigPaths: []string{dir}},
		istio.New(), k8s.New(), envoy.New(),
	)
	if err != nil {
		return model.ContractVersion{}, nil, fmt.Errorf("discovery: %w", err)
	}
	version := contract.Build(contract.BuildInput{Consumers: consumers})

	var changes []model.ChangeEvent
	if prev != nil {
		changes = diff.Compute(*prev, version)
	}
	return version, changes, nil
}

// sanitizeName flattens an uploaded path to a safe filename in the temp dir and
// ensures a config extension so the discoverers pick it up. No path traversal is
// possible: separators are stripped, so the write always lands inside the dir.
func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "/", "__")
	name = strings.ReplaceAll(name, "\\", "__")
	name = strings.TrimSpace(name)
	if name == "" {
		name = "manifest"
	}
	lower := strings.ToLower(name)
	if !strings.HasSuffix(lower, ".yaml") && !strings.HasSuffix(lower, ".yml") && !strings.HasSuffix(lower, ".json") {
		name += ".yaml"
	}
	return name
}

// isManifestFile reports whether a repo path looks like a config manifest worth
// feeding to discovery (used when syncing from a connected repo).
func isManifestFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "/node_modules/") || strings.Contains(lower, "/vendor/") {
		return false
	}
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}
