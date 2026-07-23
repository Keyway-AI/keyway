// Command harness runs the Keyway accuracy benchmark corpus and emits a
// scorecard (PRD §13). It scores two ways:
//
//   - a generated corpus (mutations.Generate) that exercises the diff/classify
//     layer (L3) at the §13.2 composition (~50% true / ~50% false positives);
//   - any file-based before/after scenarios under the corpus dir, which also
//     exercise real discovery (L1).
//
// With --ci-gate it exits non-zero if any metric drops below the §13.4
// thresholds.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/architsharma/keyway/bench/mutations"
)

func main() {
	var (
		corpus = flag.String("corpus", "./bench/corpus", "path to file-based scenarios")
		out    = flag.String("out", "./bench/out", "output directory for the scorecard")
		rounds = flag.Int("rounds", 50, "generated-corpus rounds (each ~12 scenarios)")
		ciGate = flag.Bool("ci-gate", false, "exit non-zero if any §13.4 threshold fails")
	)
	flag.Parse()

	generated := mutations.Generate(*rounds)
	fileScenarios, err := loadFileScenarios(*corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harness: load file scenarios:", err)
		os.Exit(1)
	}

	all := append([]mutations.Scenario{}, generated...)
	all = append(all, fileScenarios...)

	genCard := score(generated)
	genCard.Layer = "L3"
	fullCard := score(all)
	fullCard.Layer = "L3-all"

	cards := map[string]Scorecard{"L3": genCard, "L3-all": fullCard}
	// L1 is measured by the file-based scenarios (real discovery). Approximate
	// recall as scenarios whose consumers were discovered non-empty.
	if len(fileScenarios) > 0 {
		cards["L1"] = l1Card(fileScenarios)
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

	fmt.Printf("harness: %d generated + %d file scenario(s)\n", len(generated), len(fileScenarios))
	fmt.Printf("  L3 diff: TPR=%.3f FPR=%.3f precision=%.3f Youden=%.3f (TP=%d FP=%d TN=%d FN=%d)\n",
		genCard.TPR, genCard.FPR, genCard.Precision, genCard.Youden, genCard.TP, genCard.FP, genCard.TN, genCard.FN)
	fmt.Printf("  scorecard -> %s\n", scorePath)

	if *ciGate {
		if checkGates(cards) {
			os.Exit(2)
		}
	}
}

// l1Card approximates derivation recall: a scenario "found" its consumers when
// discovery returned a non-empty after-graph.
func l1Card(scenarios []mutations.Scenario) Scorecard {
	card := Scorecard{Layer: "L1"}
	for _, s := range scenarios {
		if len(s.After) > 0 {
			card.TP++
		} else {
			card.FN++
		}
	}
	card.Compute()
	return card
}

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
		fmt.Println("harness: all applicable §13.4 gates passed.")
	}
	return failed
}
