// Command keyway-runner is the Keyway daemon: the same functionality as the CLI
// but defaulting to `serve` (API + scheduler) for in-VPC deployment.
package main

import "github.com/Keyway-AI/keyway/internal/cli"

func main() {
	cli.Execute(cli.Options{Runner: true})
}
