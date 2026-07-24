// Command l2 is the live-probe (L2) benchmark rig. One binary, three modes:
//
//	l2 issuer      # serves OIDC discovery + JWKS + a test-only /mint endpoint
//	l2 validator   # a real JWT-validating service; VULN env selects a weakness
//	l2 score       # runs Keyway's probe engine against the validators, scores L2
//
// docker-compose.yml stands up one issuer + one validator per weakness; `score`
// probes them over the network and checks Keyway returns the correct verdict for
// each. This exercises the one layer the offline corpus can't: real tokens vs.
// real services (PRD §13, L2). See bench/l2/README.md.
package main

import (
	"fmt"
	"os"
)

// Logical identifiers shared by all three modes (independent of network host).
const (
	l2Issuer   = "http://issuer.l2.local"
	l2Audience = "l2-api"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: l2 <issuer|validator|score> [flags]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "issuer":
		err = runIssuer(os.Args[2:])
	case "validator":
		err = runValidator(os.Args[2:])
	case "score":
		err = runScore(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "l2:", err)
		os.Exit(1)
	}
}
