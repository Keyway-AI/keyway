package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Scenario is one benchmark case (PRD §13.1).
type Scenario struct {
	ID       string         `yaml:"id"`
	Name     string         `yaml:"name"`
	Compose  string         `yaml:"compose"`
	Mutation map[string]any `yaml:"mutation"`
	Expected Expected       `yaml:"expected"`
	Path     string         `yaml:"-"`
}

// Expected is the ground truth for a scenario.
type Expected struct {
	Detected bool   `yaml:"detected"`
	Class    string `yaml:"class"`
	Severity string `yaml:"severity"`
	Consumer string `yaml:"consumer"`
}

// loadScenarios reads every scenario.yaml under corpus/scenarios.
func loadScenarios(corpus string) ([]Scenario, error) {
	root := filepath.Join(corpus, "scenarios")
	var scenarios []Scenario
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if info.IsDir() || filepath.Base(path) != "scenario.yaml" {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var sc Scenario
		if e := yaml.Unmarshal(b, &sc); e != nil {
			return e
		}
		sc.Path = filepath.Dir(path)
		scenarios = append(scenarios, sc)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return scenarios, nil
}

// Run executes each scenario and aggregates results into per-layer scorecards.
//
// TODO(M8): stand up each scenario's docker-compose, run Keyway's
// discover/snapshot/diff pipeline against it, compare to Expected, and tally
// the confusion matrix. For now this returns empty, computed scorecards so the
// harness wiring and gate logic are exercisable.
func Run(scenarios []Scenario) (map[string]Scorecard, error) {
	_ = scenarios
	cards := map[string]Scorecard{}
	for _, layer := range []string{"L1", "L2", "L3", "L4"} {
		c := Scorecard{Layer: layer}
		c.Compute()
		cards[layer] = c
	}
	return cards, nil
}
