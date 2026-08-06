package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/Keyway-AI/keyway/internal/model"
	"github.com/Keyway-AI/keyway/internal/probe"
)

// validatorSpec ties a running validator container to its ground-truth weakness
// and the probe that should flag it. Mirrors docker-compose.yml.
type validatorSpec struct {
	name     string
	port     string
	vuln     string
	affected string // probe ID expected to fire; "" for the secure service
}

func topology() []validatorSpec {
	return []validatorSpec{
		{"secure", "8081", "secure", ""},
		{"vuln-alg-none", "8082", "none", probe.ProbeAlgNone},
		{"vuln-confusion", "8083", "confusion", probe.ProbeAlgConfusion},
		{"vuln-nosig", "8084", "nosig", probe.ProbeTamperedSignature},
		{"vuln-no-aud", "8085", "noaud", probe.ProbeWrongAudience},
		{"vuln-no-iss", "8086", "noiss", probe.ProbeWrongIssuer},
		{"vuln-no-exp", "8087", "noexp", probe.ProbeExpired},
		{"vuln-header", "8088", "header", probe.ProbeHeaderBypass},
	}
}

func runScore(args []string) error {
	fs := flag.NewFlagSet("score", flag.ExitOnError)
	issuerURL := fs.String("issuer", "http://localhost:9000", "issuer base URL")
	host := fs.String("host", "http://localhost", "validator host base")
	out := fs.String("out", "./bench/out", "output directory")
	gate := fs.Bool("ci-gate", false, "exit non-zero if L2 accuracy < 0.95")
	_ = fs.Parse(args)

	ctx := context.Background()

	iss, err := fetchIssuer(*issuerURL)
	if err != nil {
		return fmt.Errorf("fetch issuer /describe (is the rig up?): %w", err)
	}
	mint := httpMint(*issuerURL)

	// Build one consumer per validator and probe them all.
	var consumers []model.Consumer
	specByID := map[string]validatorSpec{}
	for _, s := range topology() {
		url := fmt.Sprintf("%s:%s", *host, s.port)
		c := model.Consumer{
			ID: s.name, StableID: s.name, Name: s.name, Kind: model.ConsumerService,
			Endpoints: []model.Endpoint{{URL: url, Method: "GET", SafeProbePath: "/"}},
			Expects: model.Expectations{
				Issuers: []string{l2Issuer}, Audiences: []string{l2Audience}, Algorithms: []string{"RS256"},
			},
			Probeable: true,
		}
		consumers = append(consumers, c)
		specByID[s.name] = s
	}

	eng := probe.NewEngine(probe.EngineConfig{
		Allowlist:      []string{"localhost", "127.0.0.1"},
		RequestTimeout: 8 * time.Second,
	})
	results, outcomes, err := eng.Run(ctx, iss, mint, consumers)
	if err != nil {
		return err
	}
	for _, o := range outcomes {
		if o.Skipped {
			return fmt.Errorf("consumer %s skipped: %s", o.ConsumerID, o.Reason)
		}
	}

	// Group results per consumer.
	byConsumer := map[string][]model.ProbeResult{}
	for _, r := range results {
		byConsumer[r.ConsumerID] = append(byConsumer[r.ConsumerID], r)
	}

	// Score: for each probe verdict, "correct" means Keyway's pass/fail matches
	// reality (the affected probe should flag exactly the vulnerable service).
	var correct, totalV int
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "\nVALIDATOR\tWEAKNESS\tFLAGGED BY\tVERDICT")
	for _, s := range topology() {
		res := byConsumer[s.name]
		flagged := ""
		detectedTarget := s.affected == "" // secure service needs nothing flagged
		for _, r := range res {
			isCorrect := (r.ProbeID == s.affected) == !r.Passed
			if isCorrect {
				correct++
			}
			totalV++
			if !r.Passed {
				flagged += r.ProbeID + " "
				if r.ProbeID == s.affected {
					detectedTarget = true
				}
			}
		}
		verdict := "✓ correct"
		if !detectedTarget {
			verdict = "✗ MISSED"
		}
		flagCol := flagged
		if flagCol == "" {
			flagCol = "(none — clean)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.name, s.vuln, flagCol, verdict)
	}
	_ = tw.Flush()

	acc := 1.0
	if totalV > 0 {
		acc = float64(correct) / float64(totalV)
	}
	fmt.Printf("\nL2 accuracy: %.4f (%d/%d correct probe verdicts across %d real services)\n",
		acc, correct, totalV, len(topology()))

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	card := map[string]any{"layer": "L2", "accuracy": acc, "correct": correct, "total": totalV, "services": len(topology())}
	b, _ := json.MarshalIndent(card, "", "  ")
	scorePath := filepath.Join(*out, "l2-scorecard.json")
	if err := os.WriteFile(scorePath, b, 0o644); err != nil {
		return err
	}
	fmt.Println("scorecard ->", scorePath)

	if *gate && acc < 0.95 { // PRD §13.4: L2 fails below 95%
		return fmt.Errorf("L2 gate failed: accuracy %.4f < 0.95", acc)
	}
	return nil
}

// fetchIssuer reads the issuer's Keyway model (keys/kids/PEM) from /describe.
func fetchIssuer(base string) (model.Issuer, error) {
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(base + "/describe")
	if err != nil {
		return model.Issuer{}, err
	}
	defer resp.Body.Close()
	var iss model.Issuer
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err := json.Unmarshal(body, &iss); err != nil {
		return model.Issuer{}, err
	}
	return iss, nil
}

// httpMint is the probe engine's MintFunc, backed by the issuer's /mint endpoint.
func httpMint(base string) probe.MintFunc {
	client := &http.Client{Timeout: 5 * time.Second}
	return func(kid string, claims map[string]any) (string, error) {
		body, _ := json.Marshal(map[string]any{"kid": kid, "claims": claims})
		resp, err := client.Post(base+"/mint", "application/json", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		var out struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return "", err
		}
		return out.Token, nil
	}
}
