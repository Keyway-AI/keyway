package main

import (
	"context"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Keyway-AI/keyway/bench/mutations"
	"github.com/Keyway-AI/keyway/internal/contract"
	"github.com/Keyway-AI/keyway/internal/diff"
	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/discovery/envoy"
	"github.com/Keyway-AI/keyway/internal/discovery/istio"
	"github.com/Keyway-AI/keyway/internal/discovery/k8s"
	"github.com/Keyway-AI/keyway/internal/model"
)

// fileScenario is a benchmark case backed by before/after manifest directories
// (exercises real discovery — the L1 layer — as well as diff — L3).
type fileScenario struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Before   string `yaml:"before"`
	After    string `yaml:"after"`
	Expected struct {
		Detected bool   `yaml:"detected"`
		Class    string `yaml:"class"`
		Severity string `yaml:"severity"`
		Consumer string `yaml:"consumer"`
	} `yaml:"expected"`
	dir string
}

// loadFileScenarios discovers before/after manifests for each scenario.yaml
// under corpus/scenarios and returns them as scored-ready Scenarios.
func loadFileScenarios(corpus string) ([]mutations.Scenario, error) {
	root := filepath.Join(corpus, "scenarios")
	var out []mutations.Scenario
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() || filepath.Base(path) != "scenario.yaml" {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var fs fileScenario
		if e := yaml.Unmarshal(b, &fs); e != nil {
			return e
		}
		fs.dir = filepath.Dir(path)
		if fs.Before == "" || fs.After == "" {
			return nil // not a before/after scenario; skipped
		}
		before, e1 := discoverDir(filepath.Join(fs.dir, fs.Before))
		after, e2 := discoverDir(filepath.Join(fs.dir, fs.After))
		if e1 != nil || e2 != nil {
			return nil
		}
		out = append(out, mutations.Scenario{
			ID: fs.ID, Name: fs.Name, Before: before, After: after,
			Expected: mutations.Expected{
				Detected: fs.Expected.Detected,
				Class:    model.ChangeClass(fs.Expected.Class),
				Severity: model.Severity(fs.Expected.Severity),
				Consumer: fs.Expected.Consumer,
			},
		})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return out, nil
}

func discoverDir(dir string) ([]model.Consumer, error) {
	scope := discovery.Scope{ConfigPaths: []string{dir}}
	return discovery.Run(context.Background(), scope, istio.New(), k8s.New(), envoy.New())
}

// score runs every scenario through the real diff pipeline and tallies an L3
// confusion matrix (detected vs. expected, with class agreement for positives).
func score(scenarios []mutations.Scenario) Scorecard {
	card := Scorecard{Layer: "L3"}
	for _, s := range scenarios {
		v1 := contract.Build(contract.BuildInput{Consumers: s.Before})
		v2 := contract.Build(contract.BuildInput{Consumers: s.After})
		events := diff.Compute(v1, v2)

		matched := matchExpected(events, s.Expected)
		switch {
		case s.Expected.Detected && matched:
			card.TP++
		case s.Expected.Detected && !matched:
			card.FN++
		case !s.Expected.Detected && len(events) == 0:
			card.TN++
		default: // expected no detection, but events emitted
			card.FP++
		}
	}
	card.Compute()
	return card
}

// matchExpected reports whether the events contain the expected class (and
// consumer, when specified).
func matchExpected(events []model.ChangeEvent, exp mutations.Expected) bool {
	if !exp.Detected {
		return len(events) == 0
	}
	for _, e := range events {
		if e.Class != exp.Class {
			continue
		}
		if exp.Consumer != "" && e.ConsumerID != exp.Consumer {
			continue
		}
		return true
	}
	return false
}
