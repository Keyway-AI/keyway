// Command harness runs the Keyway accuracy benchmark corpus and emits a
// scorecard (PRD §13). With --ci-gate it exits non-zero if any metric drops
// below the PRD §13.4 thresholds.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var (
		corpus = flag.String("corpus", "./bench/corpus", "path to the scenario corpus")
		out    = flag.String("out", "./bench/out", "output directory for the scorecard")
		ciGate = flag.Bool("ci-gate", false, "exit non-zero if any §13.4 threshold fails")
	)
	flag.Parse()

	scenarios, err := loadScenarios(*corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harness: load scenarios:", err)
		os.Exit(1)
	}

	if len(scenarios) == 0 {
		fmt.Printf("harness: no scenarios found under %s — nothing to score.\n", *corpus)
		fmt.Println("harness: add scenarios per PRD §13.1 (target: 400 for v1). See PROGRESS.md M8.")
		// An empty corpus is not a gate failure; there is simply nothing to check yet.
		return
	}

	cards, err := Run(scenarios)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harness: run:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "harness: mkdir out:", err)
		os.Exit(1)
	}
	scorePath := filepath.Join(*out, "scorecard.json")
	b, _ := json.MarshalIndent(cards, "", "  ")
	if err := os.WriteFile(scorePath, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "harness: write scorecard:", err)
		os.Exit(1)
	}
	fmt.Printf("harness: scored %d scenarios -> %s\n", len(scenarios), scorePath)

	if *ciGate {
		if failed := checkGates(cards); failed {
			os.Exit(2)
		}
	}
}

// checkGates prints any failing §13.4 thresholds and reports whether the build
// should fail.
func checkGates(cards map[string]Scorecard) bool {
	failed := false
	for _, g := range DefaultGates() {
		card, ok := cards[g.Layer]
		if !ok {
			continue
		}
		v := g.Get(card)
		if g.FailIf(v) {
			failed = true
			fmt.Printf("GATE FAIL [%s %s] = %.4f (%s)\n", g.Layer, g.Metric, v, g.Explain)
		}
	}
	if !failed {
		fmt.Println("harness: all §13.4 gates passed.")
	}
	return failed
}
