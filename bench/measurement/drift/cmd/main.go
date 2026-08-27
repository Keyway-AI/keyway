// Command drift reports whether authorization contracts widen more than they narrow
// over a timeline of contract-version JSON snapshots — the longitudinal question in
// Paper A. Point it at a directory whose files are named in chronological order
// (e.g. 0001.json, 0002.json). Producing that timeline from real commit history is
// the crawl step (G2); this reports the direction once you have it.
//
// Usage: go run ./bench/measurement/drift/cmd --dir <versions> [--json]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Keyway-AI/keyway/bench/measurement/drift"
)

func main() {
	dir := flag.String("dir", "", "directory of contract-version JSON files, named in chronological order")
	asJSON := flag.Bool("json", false, "emit the full report as JSON")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "error: --dir is required")
		os.Exit(2)
	}
	timeline, err := drift.LoadTimeline(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	r := drift.Analyze(timeline)

	if *asJSON {
		if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("drift over %d versions (%d transitions):\n", r.Versions, len(r.Transitions))
	fmt.Printf("  widened %d, narrowed %d, neutral %d\n", r.TotalWidened, r.TotalNarrowed, r.TotalNeutral)
	fmt.Printf("  net %+d, of which %d are breaking widenings (high/critical)\n", r.NetWidened, r.BreakingWidenings)
	fmt.Printf("  %d widening vs %d narrowing transitions", r.WideningTransitions, r.NarrowingTransitions)
	if r.WidenNarrowRatio > 0 {
		fmt.Printf("; widen/narrow ratio %.2f", r.WidenNarrowRatio)
	}
	fmt.Println()
	switch {
	case r.NetWidened > 0:
		fmt.Println("  => contracts widened more than they narrowed over this window.")
	case r.NetWidened < 0:
		fmt.Println("  => contracts narrowed more than they widened over this window.")
	}
}
