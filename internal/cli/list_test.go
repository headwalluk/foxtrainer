package cli

import (
	"strings"
	"testing"

	"github.com/headwalluk/foxtrainer/internal/firefox"
)

func TestRenderInventory(test *testing.T) {
	devEdition := firefox.Install{
		Dir: "/usr/lib/firefox-devedition", Hash: "BCAEFFD141225C21", Name: "Firefox Developer Edition",
		Version: "158.0", Channel: "aurora", Packaged: true,
	}

	inventory := firefox.Inventory{
		Installs: []firefox.Install{devEdition},
		Roots: []firefox.Root{{
			Dir:  "/home/fox/.mozilla/firefox",
			File: firefox.ProfilesFile{StraySections: []string{"6AFDA46A1A8AD48"}},
			Profiles: []firefox.ProfileStatus{
				{
					Profile:     firefox.Profile{Name: "dev-edition-default-1", Dir: "/home/fox/.mozilla/firefox/new.dev-edition-default-1", StoreID: "7f1ff9e3"},
					HasRun:      true,
					LastInstall: &devEdition,
					DefaultFor:  []firefox.Install{devEdition},
					Lock:        firefox.LockState{InUse: true, HolderPID: 4242},
				},
				{
					Profile:       firefox.Profile{Name: "dev-edition-default", Dir: "/home/fox/.mozilla/firefox/old.dev-edition-default"},
					HasRun:        true,
					Compatibility: firefox.Compatibility{AppVersion: "157.0", LastPlatformDir: "/opt/firefox"},
					Orphan:        true,
				},
				{Profile: firefox.Profile{Name: "default", Dir: "/home/fox/.mozilla/firefox/stub.default"}},
			},
			DeadInstallSections: []firefox.DeadInstallSection{
				{Section: firefox.InstallSection{Hash: "6AFDA46A1A8AD48"}, LastKnownDir: "/opt/firefox"},
			},
		}},
	}

	rendered := renderInventory(inventory, "/home/fox")

	for _, expected := range []string{
		"● Firefox Developer Edition 158.0 › dev-edition-default-1",
		"profile: ~/.mozilla/firefox/new.dev-edition-default-1",
		"default profile · in a Profile Group: some telemetry prefs are group-wide and may be overridden · RUNNING (pid 4242)",
		"○ dev-edition-default  ~/.mozilla/firefox/old.dev-edition-default",
		"last used by /opt/firefox (Firefox 157.0), which no longer exists",
		"○ default  ~/.mozilla/firefox/stub.default  never started",
		"Firefox Developer Edition 158.0 (aurora, Mozilla package)  /usr/lib/firefox-devedition  [BCAEFFD141225C21]",
		"[Install6AFDA46A1A8AD48] for /opt/firefox, which no longer exists",
		"1 section(s) Firefox ignores: [6AFDA46A1A8AD48]",
	} {
		if !strings.Contains(rendered, expected) {
			test.Errorf("output is missing %q\n---\n%s", expected, rendered)
		}
	}
}

func TestShortenHome(test *testing.T) {
	cases := map[string]string{
		"/home/fox/.mozilla/firefox": "~/.mozilla/firefox",
		"/home/foxglove/profile":     "/home/foxglove/profile",
		"/usr/lib/firefox":           "/usr/lib/firefox",
	}

	for path, want := range cases {
		if got := shortenHome(path, "/home/fox"); got != want {
			test.Errorf("%s: got %s, want %s", path, got, want)
		}
	}
}
