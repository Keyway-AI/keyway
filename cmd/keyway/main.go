// Command keyway is the Keyway CLI.
package main

import "github.com/Keyway-AI/keyway/internal/cli"

func main() {
	cli.Execute(cli.Options{Runner: false})
}
