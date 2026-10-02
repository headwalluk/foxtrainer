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
	"slices"
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

	return current.foxtrainerWithInput(test, "", arguments...)
}

// foxtrainerWithInput runs the CLI with stdin set to input.
func (current *harness) foxtrainerWithInput(test *testing.T, input string, arguments ...string) string {
	test.Helper()

	environment := map[string]string{"HOME": current.homeDir, "PATH": hermeticSearchPath}
	loadConfig := func() (config.Config, error) {
		return config.Load(func(name string) (string, bool) {
			value, found := environment[name]

			return value, found
		}, "linux")
	}

	var stdout, stderr bytes.Buffer

	if exitCode := cli.Run(test.Context(), arguments, loadConfig, strings.NewReader(input), &stdout, &stderr); exitCode != cli.ExitOK {
		test.Fatalf("foxtrainer %v: exit %d\n%s%s", arguments, exitCode, stdout.String(), stderr.String())
	}

	return stdout.String()
}

// profileDir returns the e2e profile's folder.
func (current *harness) profileDir(test *testing.T) string {
	test.Helper()

	matches, globError := filepath.Glob(filepath.Join(current.homeDir, ".config", "mozilla", "firefox", "*.e2e"))
	if globError != nil || len(matches) != 1 {
		test.Fatalf("find profile: %v %v", matches, globError)
	}

	return matches[0]
}

// prefsJS returns the user values Firefox saved in the profile's prefs.js.
func (current *harness) prefsJS(test *testing.T) map[string]prefs.Value {
	test.Helper()

	content, readError := os.ReadFile(filepath.Join(current.profileDir(test), "prefs.js"))
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

// The problem this project started from: Mozilla's builds spellcheck only in en-US until pointed at system hunspell.
func TestBritishEnglishSpellchecking(test *testing.T) {
	if _, statError := os.Stat("/usr/share/hunspell/en_GB.dic"); statError != nil {
		test.Skip("needs hunspell-en-gb installed")
	}

	current := newHarness(test)

	current.foxtrainer(test, "configure", "--profile", "e2e", "--languages", "en-GB,en")
	current.foxtrainer(test, "apply", "--offline")

	var spelling struct {
		Dictionaries []string `json:"dictionaries"`
		Selected     string   `json:"selected"`
		Accept       string   `json:"accept"`
	}

	current.withMarionette(test, func(client *marionette) {
		client.script(test, `
		  const engine = Cc["@mozilla.org/spellchecker/engine;1"].getService(Ci.mozISpellCheckingEngine);
		  return { dictionaries: engine.getDictionaryList(),
		           selected: Services.prefs.getCharPref("spellchecker.dictionary", ""),
		           accept: Services.prefs.getCharPref("intl.accept_languages", "") };`, &spelling)
	})

	if !slices.Contains(spelling.Dictionaries, "en-GB") {
		test.Errorf("Firefox's dictionaries %v do not include en-GB", spelling.Dictionaries)
	}

	if spelling.Selected != "en-GB" || spelling.Accept != "en-GB, en" {
		test.Errorf("selected dictionary %q, accept-languages %q", spelling.Selected, spelling.Accept)
	}
}

// The interactive path: answers piped into the accessible wizard, which saves and applies.
func TestWizardSavesAndApplies(test *testing.T) {
	current := newHarness(test)

	// One unconfigured instance, so neither the instance nor the starting point is asked.
	// Feel: lean (1). AI: local only (2). Privacy: strict (2). HTTPS-Only: no. Languages. Review: save and apply (1).
	output := current.foxtrainerWithInput(test, "1\n2\n2\nn\nen-GB, en\n1\n", "configure", "--accessible")

	if !strings.Contains(output, "Saved answers for") || !strings.Contains(output, "written (+") {
		test.Fatalf("wizard output:\n%s", output)
	}

	current.runFirefox(test, 8*time.Second, "-P", "e2e")

	saved := current.prefsJS(test)
	expected := map[string]prefs.Value{
		"browser.ai.control.sidebarChatbot": prefs.String("blocked"),
		"startup.homepage_override_url":     prefs.String(""),
		"browser.contentblocking.category":  prefs.String("strict"),
		"intl.accept_languages":             prefs.String("en-GB, en"),
	}

	for name, want := range expected {
		if got, found := saved[name]; !found || got != want {
			test.Errorf("%s = %v (found %v), want %v", name, got, found, want)
		}
	}

	for _, absent := range []string{"browser.ai.control.default", "dom.security.https_only_mode"} {
		if value, found := saved[absent]; found {
			test.Errorf("%s should not be set with AI local-only and HTTPS-Only off, got %v", absent, value)
		}
	}
}
