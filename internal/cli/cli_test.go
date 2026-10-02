package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/headwalluk/foxtrainer/internal/config"
)

// runWith runs the CLI with a config loaded from values and returns the exit code and output.
func runWith(arguments []string, values map[string]string) (int, string, string) {
	var stdout, stderr bytes.Buffer

	loadConfig := func() (config.Config, error) {
		return config.Load(func(name string) (string, bool) {
			value, found := values[name]

			return value, found
		}, "linux")
	}

	exitCode := Run(context.Background(), arguments, loadConfig, strings.NewReader(""), &stdout, &stderr)

	return exitCode, stdout.String(), stderr.String()
}

func TestHelpListsCommands(test *testing.T) {
	exitCode, stdout, _ := runWith(nil, nil)
	if exitCode != ExitOK {
		test.Fatalf("exit code %d", exitCode)
	}

	for _, name := range []string{"configure", "apply", "list", "diff", "paths", "version"} {
		if !strings.Contains(stdout, name) {
			test.Errorf("help is missing %q", name)
		}
	}
}

func TestUnknownCommandIsUsageError(test *testing.T) {
	exitCode, _, stderr := runWith([]string{"tame"}, nil)
	if exitCode != ExitUsage || !strings.Contains(stderr, `unknown command "tame"`) {
		test.Errorf("got exit %d, stderr %q", exitCode, stderr)
	}
}

func TestVersionWorksWithBrokenConfig(test *testing.T) {
	exitCode, stdout, _ := runWith([]string{"--version"}, map[string]string{config.EnvLogLevel: "chatty"})
	if exitCode != ExitOK || !strings.HasPrefix(stdout, "foxtrainer ") {
		test.Errorf("got exit %d, stdout %q", exitCode, stdout)
	}
}

func TestBrokenConfigFailsLoud(test *testing.T) {
	exitCode, _, stderr := runWith([]string{"paths"}, map[string]string{config.EnvLogLevel: "chatty"})
	if exitCode != ExitError || !strings.Contains(stderr, "HOME is not set") || !strings.Contains(stderr, "chatty") {
		test.Errorf("got exit %d, stderr %q", exitCode, stderr)
	}
}

func TestPathsPrintsResolvedFolders(test *testing.T) {
	exitCode, stdout, _ := runWith([]string{"paths"}, map[string]string{"HOME": "/home/fox"})
	if exitCode != ExitOK || !strings.Contains(stdout, "answers: /home/fox/.config/foxtrainer/config.toml") {
		test.Errorf("got exit %d, stdout %q", exitCode, stdout)
	}
}

func TestConfigureWithoutTerminalExplainsTheOptions(test *testing.T) {
	exitCode, _, stderr := runWith([]string{"configure"}, map[string]string{"HOME": "/home/fox"})
	if exitCode != ExitError || !strings.Contains(stderr, "--accessible") || !strings.Contains(stderr, "--profile") {
		test.Errorf("got exit %d, stderr %q", exitCode, stderr)
	}
}
