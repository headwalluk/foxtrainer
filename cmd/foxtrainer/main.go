// Command foxtrainer configures Firefox profiles from a few high-level answers.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/headwalluk/foxtrainer/internal/cli"
	"github.com/headwalluk/foxtrainer/internal/config"
)

// main runs the CLI, cancelling on Ctrl-C, and exits with its status code.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	exitCode := cli.Run(ctx, os.Args[1:], config.FromEnvironment, os.Stdin, os.Stdout, os.Stderr)

	stop()
	os.Exit(exitCode)
}
