//go:build e2e

// Package e2e runs foxtrainer against a real Firefox on a throwaway profile in a fake HOME.
//
// Run with `make e2e`. Values are read from prefs.js after a plain headless run, not over
// Marionette, whose automation prefs override some of the prefs under test.
package e2e

import (
	"bytes"
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/headwalluk/foxtrainer/internal/cli"
	"github.com/headwalluk/foxtrainer/internal/config"
	"github.com/headwalluk/foxtrainer/internal/prefs"
)

var firefoxBinary = flag.String("firefox", "firefox-devedition", "Firefox binary to test against")

const pinnedCommit = "067172a4b0dc90e78e5b8b94d9abfe6430c6a7be"

// hermeticSearchPath is the PATH given to Firefox and foxtrainer; the developer's own environment is never used.
const hermeticSearchPath = "/usr/local/bin:/usr/bin:/bin"

// hermeticEnvironment is the complete environment for a Firefox child process.
func hermeticEnvironment(homeDir string) []string {
	return []string{"HOME=" + homeDir, "PATH=" + hermeticSearchPath, "LANG=en_GB.UTF-8"}
}

// harness is a fake HOME holding one Firefox profile called "e2e".
type harness struct {
	homeDir string
	firefox string
}

// newHarness creates the fake HOME, the profile, a first Firefox run and a pre-filled source cache.
func newHarness(test *testing.T) *harness {
	test.Helper()

	firefox, lookError := exec.LookPath(*firefoxBinary)
	if lookError != nil {
		test.Skipf("Firefox not available (%v); pass -args -firefox=PATH", lookError)
	}

	created := &harness{homeDir: test.TempDir(), firefox: firefox}

	created.runFirefox(test, 0, "-CreateProfile", "e2e")
	created.runFirefox(test, 6*time.Second, "-P", "e2e")

	cacheDir := filepath.Join(created.homeDir, ".cache", "foxtrainer", "sources", "betterfox", pinnedCommit)
	if mkdirError := os.MkdirAll(cacheDir, 0o700); mkdirError != nil {
		test.Fatal(mkdirError)
	}

	for _, fileName := range []string{"user.js", "Peskyfox.js"} {
		content, readError := os.ReadFile(filepath.Join("../../testdata/upstream/betterfox", pinnedCommit, fileName))
		if readError != nil {
			test.Fatal(readError)
		}

		if writeError := os.WriteFile(filepath.Join(cacheDir, fileName), content, 0o600); writeError != nil {
			test.Fatal(writeError)
		}
	}

	return created
}

// runFirefox runs headless Firefox in the fake HOME; a non-zero duration stops it with SIGTERM after that long.
func (current *harness) runFirefox(test *testing.T, duration time.Duration, arguments ...string) {
	test.Helper()

	ctx := test.Context()

	if duration > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	command := exec.CommandContext(ctx, current.firefox, append([]string{"--headless", "-no-remote"}, arguments...)...)
	command.Env = hermeticEnvironment(current.homeDir)
	command.Cancel = func() error { return command.Process.Signal(syscall.SIGTERM) }
	command.WaitDelay = 20 * time.Second

	runError := command.Run()
	if runError != nil && ctx.Err() == nil {
		test.Fatalf("firefox %v: %v", arguments, runError)
	}
}

// foxtrainer runs the CLI in-process against the fake HOME and returns its output.
func (current *harness) foxtrainer(test *testing.T, arguments ...string) string {
	test.Helper()

	environment := map[string]string{"HOME": current.homeDir, "PATH": hermeticSearchPath}
	loadConfig := func() (config.Config, error) {
		return config.Load(func(name string) (string, bool) {
			value, found := environment[name]

			return value, found
		}, "linux")
	}

	var stdout, stderr bytes.Buffer

	if exitCode := cli.Run(test.Context(), arguments, loadConfig, &stdout, &stderr); exitCode != cli.ExitOK {
		test.Fatalf("foxtrainer %v: exit %d\n%s%s", arguments, exitCode, stdout.String(), stderr.String())
	}

	return stdout.String()
}

// prefsJS returns the user values Firefox saved in the profile's prefs.js.
func (current *harness) prefsJS(test *testing.T) map[string]prefs.Value {
	test.Helper()

	matches, globError := filepath.Glob(filepath.Join(current.homeDir, ".config", "mozilla", "firefox", "*.e2e", "prefs.js"))
	if globError != nil || len(matches) != 1 {
		test.Fatalf("find prefs.js: %v %v", matches, globError)
	}

	content, readError := os.ReadFile(matches[0])
	if readError != nil {
		test.Fatal(readError)
	}

	values := map[string]prefs.Value{}

	for line := range strings.SplitSeq(string(content), "\n") {
		if statement, parseError := prefs.ParseStatement(strings.TrimSpace(line)); parseError == nil {
			values[statement.Name] = statement.Value
		}
	}

	return values
}

func TestApplyThenFirefoxHonoursIt(test *testing.T) {
	current := newHarness(test)

	current.foxtrainer(test, "configure", "--profile", "e2e", "--languages", "en-GB,en")
	applied := current.foxtrainer(test, "apply", "--offline")

	if !strings.Contains(applied, "written (+") {
		test.Fatalf("apply output: %s", applied)
	}

	current.runFirefox(test, 8*time.Second, "-P", "e2e")

	saved := current.prefsJS(test)
	expected := map[string]prefs.Value{
		"browser.ai.control.default":                                 prefs.String("blocked"),
		"browser.ml.enable":                                          prefs.Bool(false),
		"toolkit.telemetry.enabled":                                  prefs.Bool(false),
		"dom.security.https_only_mode":                               prefs.Bool(true),
		"browser.contentblocking.category":                           prefs.String("strict"),
		"browser.newtabpage.activity-stream.feeds.weatherfeed":       prefs.Bool(false),
		"browser.newtabpage.activity-stream.widgets.weather.enabled": prefs.Bool(false),
		"browser.newtabpage.activity-stream.feeds.topsites":          prefs.Bool(false),
	}

	for name, want := range expected {
		if got, found := saved[name]; !found || got != want {
			test.Errorf("after apply, Firefox saved %s = %v (found %v), want %v", name, got, found, want)
		}
	}

	current.foxtrainer(test, "configure", "--profile", "e2e", "--ai", "all")

	switched := current.foxtrainer(test, "apply", "--offline")
	if !strings.Contains(switched, "reset to Firefox's default") {
		test.Fatalf("apply output: %s", switched)
	}

	current.runFirefox(test, 8*time.Second, "-P", "e2e")

	afterSwitch := current.prefsJS(test)
	for _, name := range []string{"browser.ai.control.default", "browser.ml.enable", "browser.translations.enable"} {
		if value, found := afterSwitch[name]; found {
			test.Errorf("after switching AI to all, %s should be back at Firefox's default, but prefs.js has %v", name, value)
		}
	}

	if afterSwitch["toolkit.telemetry.enabled"] != prefs.Bool(false) {
		test.Error("telemetry must stay off after the AI switch")
	}
}
