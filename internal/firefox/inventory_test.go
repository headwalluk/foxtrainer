package firefox

import (
	"path/filepath"
	"strings"
	"testing"
)

// Mirrors the real migration this project was born from: a tarball Developer Edition replaced by the apt package.
func TestDiscoverFindsInstancesOrphansAndDeadSections(test *testing.T) {
	baseDir := test.TempDir()
	rootDir := filepath.Join(baseDir, "home", ".mozilla", "firefox")
	liveInstall := writeFakeInstall(test, filepath.Join(baseDir, "usr", "lib", "firefox-devedition"), "Firefox Developer Edition", "158.0", "aurora")
	goneInstall := filepath.Join(baseDir, "opt", "firefox") // never created: removed tarball

	writeTestFile(test, filepath.Join(rootDir, "profiles.ini"), strings.Join([]string{
		"[General]", "StartWithLastProfile=1", "Version=2", "",
		"[Profile0]", "Name=dev-edition-default-1", "IsRelative=1", "Path=new.dev-edition-default-1", "",
		"[Profile1]", "Name=dev-edition-default", "IsRelative=1", "Path=old.dev-edition-default", "",
		"[Profile2]", "Name=default", "IsRelative=1", "Path=stub.default", "Default=1", "",
		"[Install" + InstallHash(liveInstall) + "]", "Default=new.dev-edition-default-1", "Locked=1", "",
		"[Install" + InstallHash(goneInstall) + "]", "Default=old.dev-edition-default", "Locked=1", "",
	}, "\n"))

	writeCompatibility(test, filepath.Join(rootDir, "new.dev-edition-default-1"), "158.0", liveInstall)
	writeCompatibility(test, filepath.Join(rootDir, "old.dev-edition-default"), "157.0", goneInstall)
	writeTestFile(test, filepath.Join(rootDir, "stub.default", "times.json"), "{}")

	inventory := Discover(DiscoverOptions{
		ProfileRoots:          []string{rootDir, filepath.Join(baseDir, "missing-root")},
		InstallSearchPatterns: []string{filepath.Join(baseDir, "usr", "lib", "firefox*")},
	})

	if len(inventory.Warnings) != 0 {
		test.Errorf("unexpected warnings: %v", inventory.Warnings)
	}

	if len(inventory.Roots) != 1 || len(inventory.Installs) != 1 {
		test.Fatalf("roots=%d installs=%d; want 1 and 1", len(inventory.Roots), len(inventory.Installs))
	}

	instances := inventory.Instances()
	if len(instances) != 1 {
		test.Fatalf("want 1 instance, got %d: %+v", len(instances), instances)
	}

	if instances[0].Profile.Profile.Name != "dev-edition-default-1" || !instances[0].IsDefault {
		test.Errorf("instance: %+v", instances[0])
	}

	profilesByName := map[string]ProfileStatus{}
	for _, status := range inventory.Roots[0].Profiles {
		profilesByName[status.Profile.Name] = status
	}

	if !profilesByName["dev-edition-default"].Orphan {
		test.Error("old tarball profile should be an orphan")
	}

	if stub := profilesByName["default"]; stub.Orphan || stub.HasRun {
		test.Errorf("never-run stub profile: %+v", stub)
	}

	dead := inventory.Roots[0].DeadInstallSections
	if len(dead) != 1 || dead[0].LastKnownDir != goneInstall {
		test.Errorf("dead install sections: %+v", dead)
	}
}

func TestSandboxPathsAreNeverOrphans(test *testing.T) {
	for _, dir := range []string{"/snap/firefox/4793/usr/lib/firefox", "/app/lib/firefox"} {
		if !isSandboxPath(dir) {
			test.Errorf("%s should count as a sandbox path", dir)
		}
	}

	if isSandboxPath("/opt/firefox") {
		test.Error("/opt/firefox is not a sandbox path")
	}
}
