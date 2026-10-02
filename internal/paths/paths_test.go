package paths

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveLinuxDefaults(test *testing.T) {
	resolved, warnings, resolveError := Resolve(Environment{OperatingSystem: "linux", HomeDir: "/home/fox"})
	if resolveError != nil {
		test.Fatalf("unexpected error: %v", resolveError)
	}

	want := Paths{
		HomeDir:      "/home/fox",
		ConfigDir:    "/home/fox/.config/foxtrainer",
		AnswersFile:  "/home/fox/.config/foxtrainer/config.toml",
		CacheDir:     "/home/fox/.cache/foxtrainer",
		StateDir:     "/home/fox/.local/state/foxtrainer",
		FirefoxRoots: []string{"/home/fox/.mozilla/firefox", "/home/fox/.config/mozilla/firefox"},
		InstallSearchPatterns: []string{
			"/usr/lib/firefox*", "/usr/lib64/firefox*", "/opt/firefox*",
			"/home/fox/firefox*", "/home/fox/.local/opt/firefox*",
		},
		HunspellDirs: []string{"/usr/share/hunspell", "/usr/share/myspell/dicts"},
		ScopedExtensionDirs: []string{
			"/home/fox/.mozilla/extensions/{ec8030f7-c20a-464f-9b0e-13a3a9e97384}",
			"/usr/lib/mozilla/extensions/{ec8030f7-c20a-464f-9b0e-13a3a9e97384}",
			"/usr/lib64/mozilla/extensions/{ec8030f7-c20a-464f-9b0e-13a3a9e97384}",
			"/usr/share/mozilla/extensions/{ec8030f7-c20a-464f-9b0e-13a3a9e97384}",
		},
	}

	if !reflect.DeepEqual(resolved, want) {
		test.Errorf("got %+v\nwant %+v", resolved, want)
	}

	if len(warnings) != 0 {
		test.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestResolveLinuxHonoursAbsoluteXDG(test *testing.T) {
	resolved, _, resolveError := Resolve(Environment{
		OperatingSystem: "linux",
		HomeDir:         "/home/fox",
		XDGConfigHome:   "/srv/config",
		XDGCacheHome:    "/srv/cache",
		XDGStateHome:    "/srv/state",
	})
	if resolveError != nil {
		test.Fatalf("unexpected error: %v", resolveError)
	}

	checks := map[string][2]string{
		"config": {resolved.ConfigDir, "/srv/config/foxtrainer"},
		"cache":  {resolved.CacheDir, "/srv/cache/foxtrainer"},
		"state":  {resolved.StateDir, "/srv/state/foxtrainer"},
		"xdg ff": {resolved.FirefoxRoots[1], "/srv/config/mozilla/firefox"},
	}

	for label, pair := range checks {
		if pair[0] != pair[1] {
			test.Errorf("%s: got %s, want %s", label, pair[0], pair[1])
		}
	}
}

func TestResolveLinuxIgnoresRelativeXDGWithWarning(test *testing.T) {
	resolved, warnings, resolveError := Resolve(Environment{
		OperatingSystem: "linux",
		HomeDir:         "/home/fox",
		XDGConfigHome:   "relative/config",
	})
	if resolveError != nil {
		test.Fatalf("unexpected error: %v", resolveError)
	}

	if resolved.ConfigDir != "/home/fox/.config/foxtrainer" {
		test.Errorf("relative XDG_CONFIG_HOME was not ignored: %s", resolved.ConfigDir)
	}

	if len(warnings) != 1 || !strings.Contains(warnings[0], "XDG_CONFIG_HOME") {
		test.Errorf("want one XDG_CONFIG_HOME warning, got %v", warnings)
	}
}

func TestResolveDarwin(test *testing.T) {
	resolved, _, resolveError := Resolve(Environment{OperatingSystem: "darwin", HomeDir: "/Users/fox"})
	if resolveError != nil {
		test.Fatalf("unexpected error: %v", resolveError)
	}

	if resolved.ConfigDir != "/Users/fox/Library/Application Support/foxtrainer" {
		test.Errorf("config: %s", resolved.ConfigDir)
	}

	if resolved.CacheDir != "/Users/fox/Library/Caches/foxtrainer" {
		test.Errorf("cache: %s", resolved.CacheDir)
	}

	if !reflect.DeepEqual(resolved.FirefoxRoots, []string{"/Users/fox/Library/Application Support/Firefox"}) {
		test.Errorf("firefox roots: %v", resolved.FirefoxRoots)
	}
}

func TestResolveReportsEveryProblem(test *testing.T) {
	_, _, resolveError := Resolve(Environment{OperatingSystem: "windows"})
	if resolveError == nil {
		test.Fatal("want an error for missing APPDATA and LOCALAPPDATA")
	}

	message := resolveError.Error()
	for _, expected := range []string{"APPDATA is not set", "LOCALAPPDATA is not set"} {
		if !strings.Contains(message, expected) {
			test.Errorf("error %q is missing %q", message, expected)
		}
	}
}

func TestResolveRejectsMissingOrRelativeHome(test *testing.T) {
	cases := map[string]string{"missing": "", "relative": "home/fox"}

	for label, homeDir := range cases {
		_, _, resolveError := Resolve(Environment{OperatingSystem: "linux", HomeDir: homeDir})
		if resolveError == nil || !strings.Contains(resolveError.Error(), "HOME") {
			test.Errorf("%s HOME: want a HOME error, got %v", label, resolveError)
		}
	}
}

func TestResolveRejectsUnknownOperatingSystem(test *testing.T) {
	_, _, resolveError := Resolve(Environment{OperatingSystem: "plan9", HomeDir: "/home/fox"})
	if resolveError == nil || !strings.Contains(resolveError.Error(), "plan9") {
		test.Errorf("want an unsupported OS error, got %v", resolveError)
	}
}

func TestResolveSplitsSearchPathKeepingAbsoluteDirs(test *testing.T) {
	resolved, _, resolveError := Resolve(Environment{
		OperatingSystem: "linux",
		HomeDir:         "/home/fox",
		SearchPath:      "/usr/local/bin::relative/bin:/usr/bin",
	})
	if resolveError != nil {
		test.Fatalf("unexpected error: %v", resolveError)
	}

	want := []string{"/usr/local/bin", "/usr/bin"}
	if !reflect.DeepEqual(resolved.ExecutableDirs, want) {
		test.Errorf("got %v, want %v", resolved.ExecutableDirs, want)
	}
}
