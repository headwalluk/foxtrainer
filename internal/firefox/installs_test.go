package firefox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadInstallReadsVersionChannelAndHash(test *testing.T) {
	installDir := writeFakeInstall(test, filepath.Join(test.TempDir(), "firefox-devedition"), "Firefox Developer Edition", "158.0", "aurora")

	install, found, readError := ReadInstall(installDir)
	if readError != nil || !found {
		test.Fatalf("found=%v error=%v", found, readError)
	}

	want := Install{
		Dir:              installDir,
		Hash:             InstallHash(installDir),
		Name:             "Firefox Developer Edition",
		Version:          "158.0",
		MajorVersion:     158,
		BuildID:          "20261002090342",
		Channel:          "aurora",
		SourceRepository: "https://hg.mozilla.org/releases/mozilla-beta",
		Packaged:         true,
	}
	if install != want {
		test.Errorf("got  %+v\nwant %+v", install, want)
	}
}

func TestReadInstallIgnoresFoldersWithoutFirefox(test *testing.T) {
	for _, candidate := range []string{test.TempDir(), filepath.Join(test.TempDir(), "missing")} {
		_, found, readError := ReadInstall(candidate)
		if found || readError != nil {
			test.Errorf("%s: found=%v error=%v; want false, nil", candidate, found, readError)
		}
	}
}

func TestChannelFallbacks(test *testing.T) {
	installDir := test.TempDir()
	writeTestFile(test, filepath.Join(installDir, "update-settings.ini"),
		"[Settings]\nACCEPTED_MAR_CHANNEL_IDS=firefox-mozilla-beta,firefox-mozilla-release\n")

	channel, channelError := readChannel(installDir, "")
	if channelError != nil || channel != "beta" {
		test.Errorf("update-settings fallback: got %q, %v", channel, channelError)
	}

	repositoryCases := map[string]string{
		"https://hg.mozilla.org/releases/mozilla-release": "release",
		"https://hg.mozilla.org/releases/mozilla-esr140":  "esr",
		"https://hg.mozilla.org/mozilla-central":          "nightly",
		"":                                                "unknown",
	}

	for repository, want := range repositoryCases {
		if got := channelFromRepository(repository); got != want {
			test.Errorf("%q: got %q, want %q", repository, got, want)
		}
	}
}

func TestDisplayName(test *testing.T) {
	cases := []struct{ appName, codeName, channel, want string }{
		{"Firefox", "Firefox Developer Edition", "aurora", "Firefox Developer Edition"},
		{"Firefox", "", "esr", "Firefox ESR"},
		{"Firefox", "", "nightly", "Firefox Nightly"},
		{"Firefox", "", "release", "Firefox"},
		{"", "", "unknown", "Firefox"},
	}

	for _, testCase := range cases {
		if got := displayName(testCase.appName, testCase.codeName, testCase.channel); got != testCase.want {
			test.Errorf("%+v: got %q", testCase, got)
		}
	}
}

func TestDiscoverInstallsFollowsLaunchersAndReportsBrokenOnes(test *testing.T) {
	baseDir := test.TempDir()
	installDir := writeFakeInstall(test, filepath.Join(baseDir, "lib", "firefox-nightly"), "", "160.0a1", "nightly")
	binDir := filepath.Join(baseDir, "bin")

	if mkdirError := os.MkdirAll(binDir, 0o755); mkdirError != nil {
		test.Fatal(mkdirError)
	}

	symlinks := map[string]string{
		"firefox-nightly": filepath.Join(installDir, "firefox"),
		"firefox-gone":    filepath.Join(baseDir, "removed", "firefox"),
	}
	for name, target := range symlinks {
		if linkError := os.Symlink(target, filepath.Join(binDir, name)); linkError != nil {
			test.Fatal(linkError)
		}
	}

	writeTestFile(test, filepath.Join(installDir, "firefox"), "#!/bin/sh\n")

	installs, warnings := DiscoverInstalls(nil, []string{binDir}, nil)
	if len(installs) != 1 || installs[0].Dir != installDir || installs[0].Name != "Firefox Nightly" {
		test.Errorf("installs: %+v", installs)
	}

	if len(warnings) != 1 {
		test.Errorf("want one broken-launcher warning, got %v", warnings)
	}
}
