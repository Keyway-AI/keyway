// Command measurement is the Paper A instrument: it runs Keyway's discovery +
// contract engine read-only over a corpus of real deployment config and reports
// the PREVALENCE of concrete, cited auth-verification weaknesses across the
// population, with Wilson 95% confidence intervals.
//
// It emits NO judgement about any single project's security and fabricates
// nothing: every flag is derived mechanically from the discovered contract, and
// every check names the normative source it is grounded in. See
// bench/measurement/README.md for methodology, caveats, and how to run at scale.
//
// Usage:
//
//	go run ./bench/measurement --path <manifests-dir> --out bench/measurement/out
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/Keyway-AI/keyway/internal/contract"
	"github.com/Keyway-AI/keyway/internal/discovery"
	"github.com/Keyway-AI/keyway/internal/discovery/envoy"
	"github.com/Keyway-AI/keyway/internal/discovery/istio"
	"github.com/Keyway-AI/keyway/internal/discovery/k8s"
	"github.com/Keyway-AI/keyway/internal/model"
)

// check is one prevalence measurement: a cited weakness and a predicate over a
// discovered JWT-validating consumer.
type check struct {
	ID     string
	Title  string
	Source string // normative citation
	// Flag reports whether this consumer exhibits the weakness. It is only
	// evaluated over the denominator (consumers that validate at least one issuer).
	Flag func(model.Consumer) bool
}

// checks are deliberately conservative and config-derivable. Where "absent in
// config" can mean "supplied by a library default" we say so in the README; that
// gap is itself a study result (RQ4: static vs runtime), not a bug.
var checks = []check{
	{
		ID:     "P1-unbound-audience",
		Title:  "validates a token but declares no audience (unbound to a resource)",
		Source: "RFC 8707 / RFC 9728",
		Flag:   func(c model.Consumer) bool { return len(c.Expects.Audiences) == 0 },
	},
	{
		ID:     "P2-no-required-claims",
		Title:  "requires no claims beyond issuer/audience (no authorization constraint)",
		Source: "RFC 8725 §3.9 (validate all claims)",
		Flag:   func(c model.Consumer) bool { return len(c.Expects.RequiredClaims) == 0 },
	},
	{
		ID:     "P3-no-algorithm-pinning",
		Title:  "pins no signing algorithm in config (relies on the library default)",
		Source: "RFC 8725 §3.1 (restrict algorithms)",
		Flag:   func(c model.Consumer) bool { return len(c.Expects.Algorithms) == 0 },
	},
	{
		ID:     "P4-multi-issuer-trust",
		Title:  "trusts more than one issuer (wider trust surface; descriptive)",
		Source: "descriptive — not a weakness per se",
		Flag:   func(c model.Consumer) bool { return len(c.Expects.Issuers) > 1 },
	},
	{
		ID:     "P5-wide-clock-skew",
		Title:  "accepts a clock skew greater than 5 minutes",
		Source: "RFC 8725 §3.11 (bound token lifetime); operational norm",
		Flag:   func(c model.Consumer) bool { return c.Expects.ClockSkewSec > 300 },
	},
}

type record struct {
	Source         string          `json:"source"`
	StableID       string          `json:"stable_id"`
	Service        string          `json:"service"`
	Issuers        []string        `json:"issuers"`
	Audiences      []string        `json:"audiences"`
	Algorithms     []string        `json:"algorithms"`
	RequiredClaims []string        `json:"required_claims"`
	ClockSkewSec   int             `json:"clock_skew_sec"`
	Flags          map[string]bool `json:"flags"`
}

type finding struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Source     string  `json:"source"`
	K          int     `json:"k"`
	N          int     `json:"n"`
	Prevalence float64 `json:"prevalence"`
	CILow      float64 `json:"ci_low"`
	CIHigh     float64 `json:"ci_high"`
}

type summary struct {
	Consumers    int       `json:"consumers_total"`
	JWTConsumers int       `json:"jwt_consumers"` // denominator: validate >=1 issuer
	Findings     []finding `json:"findings"`
	Note         string    `json:"note"`
}

func main() {
	path := flag.String("path", "", "directory of real manifests to scan (read-only)")
	out := flag.String("out", "bench/measurement/out", "output directory")
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "error: --path is required")
		os.Exit(2)
	}

	consumers, err := discovery.Run(context.Background(),
		discovery.Scope{ConfigPaths: []string{*path}},
		istio.New(), k8s.New(), envoy.New(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "discovery:", err)
		os.Exit(1)
	}
	// Build the contract too, so the run exercises the same path a real analysis
	// would (and to fail loudly if the engine can't assemble what it discovered).
	_ = contract.Build(contract.BuildInput{Consumers: consumers})

	// Denominator = consumers that actually validate JWTs (>=1 issuer expected).
	var jwt []model.Consumer
	for _, c := range consumers {
		if len(c.Expects.Issuers) > 0 {
			jwt = append(jwt, c)
		}
	}

	records := make([]record, 0, len(jwt))
	counts := map[string]int{}
	for _, c := range jwt {
		flags := map[string]bool{}
		for _, ck := range checks {
			f := ck.Flag(c)
			flags[ck.ID] = f
			if f {
				counts[ck.ID]++
			}
		}
		records = append(records, record{
			Source: provenanceSource(c), StableID: c.StableID, Service: c.Name,
			Issuers: c.Expects.Issuers, Audiences: c.Expects.Audiences,
			Algorithms: c.Expects.Algorithms, RequiredClaims: c.Expects.RequiredClaims,
			ClockSkewSec: c.Expects.ClockSkewSec, Flags: flags,
		})
	}

	n := len(jwt)
	sum := summary{
		Consumers:    len(consumers),
		JWTConsumers: n,
		Note: "Prevalence is over JWT-validating consumers (denominator). 'Absent " +
			"in config' can mean 'set by a library default' — see README (RQ4). " +
			"This is measurement, not a judgement of any single project.",
	}
	for _, ck := range checks {
		k := counts[ck.ID]
		lo, hi := wilson(k, n)
		p := 0.0
		if n > 0 {
			p = float64(k) / float64(n)
		}
		sum.Findings = append(sum.Findings, finding{
			ID: ck.ID, Title: ck.Title, Source: ck.Source,
			K: k, N: n, Prevalence: p, CILow: lo, CIHigh: hi,
		})
	}

	if err := writeOutputs(*out, records, sum); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	printTable(sum)
}

// wilson returns the Wilson score 95% confidence interval for k successes in n
// trials — the right interval for proportions, especially at small n or extreme
// prevalence (unlike the naive normal approximation).
func wilson(k, n int) (lo, hi float64) {
	if n == 0 {
		return 0, 0
	}
	const z = 1.959963985 // 95%
	phat := float64(k) / float64(n)
	z2 := z * z
	denom := 1 + z2/float64(n)
	center := (phat + z2/(2*float64(n))) / denom
	margin := (z * math.Sqrt(phat*(1-phat)/float64(n)+z2/(4*float64(n)*float64(n)))) / denom
	lo = center - margin
	hi = center + margin
	if lo < 0 {
		lo = 0
	}
	if hi > 1 {
		hi = 1
	}
	return lo, hi
}

func provenanceSource(c model.Consumer) string {
	// Use the first provenance locator if present; fall back to the stable id.
	for _, entries := range c.Provenance {
		for _, e := range entries {
			if e.Locator != "" {
				return filepath.Base(e.Locator)
			}
			if e.Source != "" {
				return e.Source
			}
		}
	}
	return c.StableID
}

func writeOutputs(dir string, records []record, sum summary) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	// dataset.jsonl — one labelled observation per line.
	f, err := os.Create(filepath.Join(dir, "dataset.jsonl")) // #nosec G304 -- fixed out dir
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	// summary.json — prevalence with CIs.
	b, err := json.MarshalIndent(sum, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "summary.json"), b, 0o600)
}

func printTable(sum summary) {
	fmt.Printf("\nKeyway measurement — %d consumers discovered, %d validate JWTs (denominator)\n\n",
		sum.Consumers, sum.JWTConsumers)
	if sum.JWTConsumers == 0 {
		fmt.Println("No JWT-validating consumers found in this corpus.")
		return
	}
	rows := append([]finding(nil), sum.Findings...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Prevalence > rows[j].Prevalence })
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "CHECK\tK/N\tPREVALENCE\t95% CI (Wilson)\tSOURCE")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%d/%d\t%.1f%%\t[%.1f%%, %.1f%%]\t%s\n",
			r.ID, r.K, r.N, r.Prevalence*100, r.CILow*100, r.CIHigh*100, r.Source)
	}
	_ = tw.Flush()
	fmt.Printf("\n%s\n", sum.Note)
}
