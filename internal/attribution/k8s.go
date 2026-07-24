package attribution

import (
	"context"
	"os"
	"path/filepath"

	"github.com/nometria/keyway/internal/model"
	"gopkg.in/yaml.v3"
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
	full := path
	if !filepath.IsAbs(path) && a.root != "" {
		full = filepath.Join(a.root, path)
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
