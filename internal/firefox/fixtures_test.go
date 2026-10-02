package firefox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFile writes content to path, creating parent folders.
func writeTestFile(test *testing.T, path, content string) {
	test.Helper()

	if mkdirError := os.MkdirAll(filepath.Dir(path), 0o755); mkdirError != nil {
		test.Fatalf("mkdir %s: %v", filepath.Dir(path), mkdirError)
	}

	if writeError := os.WriteFile(path, []byte(content), 0o644); writeError != nil {
		test.Fatalf("write %s: %v", path, writeError)
	}
}

// writeFakeInstall creates a minimal Firefox install in installDir and returns its resolved path.
func writeFakeInstall(test *testing.T, installDir, codeName, version, channel string) string {
	test.Helper()

	writeTestFile(test, filepath.Join(installDir, "application.ini"), strings.Join([]string{
		"; This file is not used.",
		"[App]",
		"Vendor=Mozilla",
		"Name=Firefox",
		"CodeName=" + codeName,
		"Version=" + version,
		"BuildID=20261002090342",
		"SourceRepository=https://hg.mozilla.org/releases/mozilla-beta",
		"",
	}, "\n"))
	writeTestFile(test, filepath.Join(installDir, "defaults", "pref", "channel-prefs.js"),
		`pref("app.update.channel", "`+channel+`");`+"\n")
	writeTestFile(test, filepath.Join(installDir, "is-packaged-app"), "")

	resolved, resolveError := filepath.EvalSymlinks(installDir)
	if resolveError != nil {
		test.Fatalf("resolve %s: %v", installDir, resolveError)
	}

	return resolved
}

// writeCompatibility writes a compatibility.ini saying installDir last ran the profile at version.
func writeCompatibility(test *testing.T, profileDir, version, installDir string) {
	test.Helper()

	writeTestFile(test, filepath.Join(profileDir, "compatibility.ini"), strings.Join([]string{
		"[Compatibility]",
		"LastVersion=" + version + "_20261002090342/20261002090342",
		"LastOSABI=Linux_x86_64-gcc3",
		"LastPlatformDir=" + installDir,
		"LastAppDir=" + installDir + "/browser",
		"",
	}, "\n"))
}
