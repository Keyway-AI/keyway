// Package version exposes build metadata injected at link time via -ldflags.
package version

import "fmt"

// These are overridden at build time (see Makefile / Dockerfile).
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable build identifier.
func String() string {
	return fmt.Sprintf("keyway %s (commit %s, built %s)", Version, Commit, Date)
}
