package attribution

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nometria/keyway/internal/model"
)

// K8sDeployAttributor attributes a change to a Kubernetes deploy by reading the
// `kubernetes.io/change-cause` annotation from the manifest named in the event's
// evidence (set by `kubectl apply --record` or a CI pipeline). PRD §16 OPEN-5.
type K8sDeployAttributor struct {
	root string // base dir for resolving relative manifest paths
}

// NewK8sDeploy constructs a deploy attributor rooted at dir.
func NewK8sDeploy(dir string) *K8sDeployAttributor { return &K8sDeployAttributor{root: dir} }

var _ Attributor = (*K8sDeployAttributor)(nil)

const changeCauseKey = "kubernetes.io/change-cause"

// Attribute returns a deploy attribution when the changed manifest carries a
// change-cause annotation; otherwise Unattributed.
func (a *K8sDeployAttributor) Attribute(_ context.Context, ev model.ChangeEvent) (*model.Attribution, error) {
	path := filePathFrom(ev.Evidence)
	if path == "" {
		return Unattributed(), nil
	}
	// Evidence can carry attacker-influenced manifest names (e.g. a manifest's
	// metadata.name), so confine the read to the root: reject absolute paths and
	// `..` escapes, and verify containment after joining.
	rel := filepath.Clean(path)
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return Unattributed(), nil
	}
	full := rel
	if a.root != "" {
		root := filepath.Clean(a.root)
		full = filepath.Join(root, rel)
		if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
			return Unattributed(), nil
		}
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return Unattributed(), nil
	}
	cause := scanChangeCause(data)
	if cause == "" {
		return Unattributed(), nil
	}
	return &model.Attribution{Kind: "deploy", Ref: cause, Confidence: 0.8}, nil
}

// scanChangeCause returns the first change-cause annotation across a manifest's
// YAML documents.
func scanChangeCause(data []byte) string {
	dec := yaml.NewDecoder(newReader(data))
	for {
		var doc struct {
			Metadata struct {
				Annotations map[string]string `yaml:"annotations"`
			} `yaml:"metadata"`
		}
		if err := dec.Decode(&doc); err != nil {
			return ""
		}
		if c := doc.Metadata.Annotations[changeCauseKey]; c != "" {
			return c
		}
	}
}
