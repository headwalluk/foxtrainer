// Package cli parses the command line and dispatches to foxtrainer's subcommands.
package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/buildinfo"
	"github.com/headwalluk/foxtrainer/internal/config"
	"github.com/headwalluk/foxtrainer/internal/logger"
)

// Exit codes returned by Run.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

// Environment is what a command needs to run.
type Environment struct {
	Context context.Context
	Config  config.Config
	Stdin   io.Reader
	Logger  *logger.Logger
	Stdout  io.Writer
	Stderr  io.Writer
}

// command is one subcommand.
type command struct {
	summary   string
	run       func(environment Environment, arguments []string) error
	setupFree bool // runs without loading config, so it works even when config is broken
}

// allCommands lists every subcommand by name; a function rather than a var to avoid an init cycle via runHelp.
func allCommands() map[string]command {
	return map[string]command{
		"configure": {summary: "Answer a few questions for an instance (interactive), or pass --profile and answer flags", run: runConfigure},
		"apply":     {summary: "Rebuild and write user.js for every configured instance (--dry-run, --offline)", run: runApply},
		"list":      {summary: "List Firefox instances (install + profile) found on this machine", run: runList},
		"diff":      {summary: "Show what apply would change, without writing anything", run: runDiff},
		"paths":     {summary: "Show the folders foxtrainer uses", run: runPaths},
		"catalogue": {summary: "Check the catalogue, or show the prefs a set of answers produces (check | show)", run: runCatalogue},
		"version":   {summary: "Print the foxtrainer version", run: runVersion, setupFree: true},
		"help":      {summary: "Show this help", run: runHelp, setupFree: true},
	}
}

// Run executes the command line in arguments (without the program name) and returns an exit code.
func Run(ctx context.Context, arguments []string, loadConfig func() (config.Config, error), stdin io.Reader, stdout, stderr io.Writer) int {
	commandName := "help"
	if len(arguments) > 0 {
		commandName = arguments[0]
	}

	switch commandName {
	case "-h", "--help":
		commandName = "help"
	case "-v", "--version":
		commandName = "version"
	}

	selected, found := allCommands()[commandName]
	if !found {
		report(stderr, "foxtrainer: unknown command %q\n\n", commandName)
		writeUsage(stderr)

		return ExitUsage
	}

	environment := Environment{Context: ctx, Stdin: stdin, Stdout: stdout, Stderr: stderr, Logger: logger.New(logger.LevelInfo, stderr)}

	exitCode := ExitOK

	if !selected.setupFree {
		loaded, loadError := loadConfig()
		if loadError != nil {
			report(stderr, "foxtrainer: %v\n", loadError)

			return ExitError
		}

		environment.Config = loaded
		environment.Logger = logger.New(loaded.LogLevel, stderr)

		for _, warning := range loaded.Warnings {
			environment.Logger.Warnf("%s", warning)
		}
	}

	commandArguments := []string{}
	if len(arguments) > 1 {
		commandArguments = arguments[1:]
	}

	if runError := selected.run(environment, commandArguments); runError != nil {
		environment.Logger.Errorf("%s: %v", commandName, runError)

		exitCode = ExitError
	}

	return exitCode
}

// report writes a message to stderr before a logger exists.
func report(stderr io.Writer, format string, arguments ...any) {
	// stderr is the last-resort channel; a failed write there has nowhere else to go.
	_, _ = fmt.Fprintf(stderr, format, arguments...)
}

// runVersion prints the build version.
func runVersion(environment Environment, _ []string) error {
	_, writeError := fmt.Fprintf(environment.Stdout, "foxtrainer %s\n", buildinfo.Version())

	return writeError
}

// runHelp prints usage to stdout.
func runHelp(environment Environment, _ []string) error {
	writeUsage(environment.Stdout)

	return nil
}

// runPaths prints the resolved foxtrainer folders and Firefox roots.
func runPaths(environment Environment, _ []string) error {
	resolved := environment.Config.Paths

	var builder strings.Builder
	fmt.Fprintf(&builder, "config:  %s\n", resolved.ConfigDir)
	fmt.Fprintf(&builder, "answers: %s\n", resolved.AnswersFile)
	fmt.Fprintf(&builder, "cache:   %s\n", resolved.CacheDir)
	fmt.Fprintf(&builder, "state:   %s\n", resolved.StateDir)

	for _, root := range resolved.FirefoxRoots {
		fmt.Fprintf(&builder, "firefox: %s\n", root)
	}

	_, writeError := io.WriteString(environment.Stdout, builder.String())

	return writeError
}

// writeUsage prints the command summary.
func writeUsage(output io.Writer) {
	commands := allCommands()

	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}

	sort.Strings(names)

	var builder strings.Builder
	builder.WriteString("foxtrainer tames Firefox: answer a few questions, then apply them to every instance.\n\n")
	builder.WriteString("Usage: foxtrainer <command> [options]\n\nCommands:\n")

	for _, name := range names {
		fmt.Fprintf(&builder, "  %-10s %s\n", name, commands[name].summary)
	}

	fmt.Fprintf(&builder, "\nEnvironment:\n  %s  error | warn | info | debug (default info)\n", config.EnvLogLevel)

	// Usage goes to a terminal; a failed write has nowhere better to be reported.
	_, _ = io.WriteString(output, builder.String())
}
