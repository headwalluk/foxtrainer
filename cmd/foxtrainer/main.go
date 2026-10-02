// Command foxtrainer configures Firefox profiles from a few high-level answers.
package main

import (
	"os"

	"github.com/headwalluk/foxtrainer/internal/cli"
	"github.com/headwalluk/foxtrainer/internal/config"
)

// main runs the CLI and exits with its status code.
func main() {
	os.Exit(cli.Run(os.Args[1:], config.FromEnvironment, os.Stdout, os.Stderr))
}
