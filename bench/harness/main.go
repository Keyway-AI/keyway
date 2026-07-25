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

	"github.com/nometria/keyway/bench/mutations"
	rw "github.com/nometria/keyway/bench/realworld"
)

func main() {
	var (
		corpus    = flag.String("corpus", "./bench/corpus", "path to file-based scenarios")
		adversary = flag.String("adversarial", "./bench/corpus/adversarial", "path to the held-out adversarial corpus (scored separately, not gated)")
		out       = flag.String("out", "./bench/out", "output directory for the scorecard")
		rounds    = flag.Int("rounds", 50, "generated-corpus rounds (each ~12 scenarios)")
		realistic = flag.Int("realistic", 400, "count of generated realistic manifest scenarios (rendered YAML through real discovery)")
		ciGate    = flag.Bool("ci-gate", false, "exit non-zero if any §13.4 threshold fails")
		report    = flag.Bool("report", false, "also emit report.html + roc.svg (human-readable study)")
		realworld = flag.Bool("realworld", false, "validate against documented CVEs/incidents -> docs/realworld-validation.md")
	)
	flag.Parse()

	if *realworld {
		md, detected, total := rw.MarkdownReport()
		if err := os.WriteFile("docs/realworld-validation.md", []byte(md), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "harness: write realworld report:", err)
			os.Exit(1)
		}
		fmt.Printf("harness: real-world validation %d/%d documented risks detected -> docs/realworld-validation.md\n", detected, total)
		if *ciGate && detected < total {
			os.Exit(3)
		}
		return
	}

	generated := mutations.Generate(*rounds)
	fileScenarios, err := loadFileScenarios(*corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harness: load file scenarios:", err)
		os.Exit(1)
	}

	// Realistic scenarios: rendered YAML run through the actual discovery
	// pipeline (L1) into diff (L3), so a perfect score is end-to-end evidence,
	// not just struct-diff self-consistency (KI-18).
	realisticScenarios, cleanup, rerr := loadGeneratedRealistic(*realistic)
	if rerr != nil {
		fmt.Fprintln(os.Stderr, "harness: generate realistic scenarios:", rerr)
		os.Exit(1)
	}
	defer cleanup()

	all := append([]mutations.Scenario{}, generated...)
	all = append(all, fileScenarios...)

	genCard := score(generated)
	genCard.Layer = "L3"
	fullCard := score(all)
	fullCard.Layer = "L3-all"

	cards := map[string]Scorecard{"L3": genCard, "L3-all": fullCard}
	// The rendered realistic corpus is scored on its own so its end-to-end
	// signal is not diluted by the struct-level generated corpus.
	if len(realisticScenarios) > 0 {
		rc := score(realisticScenarios)
		rc.Layer = "L3-realistic"
		cards["L3-realistic"] = rc
	}
	// The held-out adversarial corpus is deliberately hard and includes cases we
	// expect to fail (documented normalization gaps). It is scored and reported
	// honestly but NOT gated — its job is to expose limitations, not to pass.
	if adv, e := loadFileScenarios(*adversary); e == nil && len(adv) > 0 {
		ac := score(adv)
		ac.Layer = "L3-adversarial"
		cards["L3-adversarial"] = ac
	}
	// L1 is measured by the file-based scenarios (real discovery). Approximate
	// recall as scenarios whose consumers were discovered non-empty.
	if len(fileScenarios) > 0 {
		cards["L1"] = l1Card(fileScenarios)
	}
	// L4 attribution (git + deploy chain) over a small realistic fixture.
	if l4, l4err := l4Score(); l4err == nil {
		cards["L4"] = l4
	} else {
		fmt.Fprintln(os.Stderr, "harness: L4 score skipped:", l4err)
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

	fmt.Printf("harness: %d generated + %d file + %d realistic scenario(s)\n", len(generated), len(fileScenarios), len(realisticScenarios))
	fmt.Printf("  L3 diff: TPR=%.3f FPR=%.3f precision=%.3f Youden=%.3f (TP=%d FP=%d TN=%d FN=%d)\n",
		genCard.TPR, genCard.FPR, genCard.Precision, genCard.Youden, genCard.TP, genCard.FP, genCard.TN, genCard.FN)
	if rc, ok := cards["L3-realistic"]; ok {
		fmt.Printf("  L3-realistic (rendered YAML → real discovery → diff): TPR=%.3f FPR=%.3f Youden=%.3f (TP=%d FP=%d TN=%d FN=%d)\n",
			rc.TPR, rc.FPR, rc.Youden, rc.TP, rc.FP, rc.TN, rc.FN)
	}
	if ac, ok := cards["L3-adversarial"]; ok {
		fmt.Printf("  L3-adversarial (held-out hard cases, NOT gated): TPR=%.3f FPR=%.3f Youden=%.3f (TP=%d FP=%d TN=%d FN=%d)\n",
			ac.TPR, ac.FPR, ac.Youden, ac.TP, ac.FP, ac.TN, ac.FN)
	}
	fmt.Printf("  scorecard -> %s\n", scorePath)

	if *report {
		rd := reportData{
			Card: fullCard, Generated: len(generated), FileBased: len(fileScenarios),
			TruePos: fullCard.TP + fullCard.FN, TrueNeg: fullCard.TN + fullCard.FP,
		}
		if err := writeReport(*out, rd); err != nil {
			fmt.Fprintln(os.Stderr, "harness: write report:", err)
			os.Exit(1)
		}
		fmt.Printf("  report -> %s\n", filepath.Join(*out, "report.html"))
	}

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
