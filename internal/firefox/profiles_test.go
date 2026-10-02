package firefox

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadProfilesFileFollowsFirefoxRules(test *testing.T) {
	rootDir := test.TempDir()
	writeTestFile(test, filepath.Join(rootDir, "profiles.ini"), strings.Join([]string{
		"[General]",
		"StartWithLastProfile=1",
		"Version=2",
		"",
		"[Profile1]",
		"Name=absolute",
		"IsRelative=0",
		"Path=/srv/profiles/absolute",
		"",
		"[Profile0]",
		"Name=default-release",
		"IsRelative=1",
		"Path=abcd1234.default-release",
		"Default=1",
		"StoreID=7f1ff9e3",
		"",
		"[Profile3]",
		"Name=after-gap",
		"IsRelative=1",
		"Path=gap.after-gap",
		"",
		"[Install4F96D1932A9F858E]",
		"Default=abcd1234.default-release",
		"Locked=1",
		"",
		"[6AFDA46A1A8AD48]",
		"Default=abcd1234.default-release",
		"this line is junk",
		"",
	}, "\n"))

	parsed, found, readError := ReadProfilesFile(rootDir)
	if readError != nil || !found {
		test.Fatalf("found=%v error=%v", found, readError)
	}

	if parsed.Version != "2" {
		test.Errorf("version: %q", parsed.Version)
	}

	wantProfiles := []Profile{
		{
			Section: "Profile0", Name: "default-release", Descriptor: "abcd1234.default-release",
			Dir: filepath.Join(rootDir, "abcd1234.default-release"), IsLegacyDefault: true, StoreID: "7f1ff9e3",
		},
		{Section: "Profile1", Name: "absolute", Descriptor: "/srv/profiles/absolute", Dir: "/srv/profiles/absolute"},
	}
	if !reflect.DeepEqual(parsed.Profiles, wantProfiles) {
		test.Errorf("profiles:\n got %+v\nwant %+v", parsed.Profiles, wantProfiles)
	}

	wantInstalls := []InstallSection{{Hash: "4F96D1932A9F858E", DefaultDescriptor: "abcd1234.default-release", Locked: true}}
	if !reflect.DeepEqual(parsed.InstallSections, wantInstalls) {
		test.Errorf("install sections: %+v", parsed.InstallSections)
	}

	if !reflect.DeepEqual(parsed.StraySections, []string{"6AFDA46A1A8AD48"}) || !IsBareHashSection(parsed.StraySections[0]) {
		test.Errorf("stray sections: %v", parsed.StraySections)
	}

	allWarnings := strings.Join(parsed.Warnings, "\n")
	for _, expected := range []string{"[Profile3] is ignored by Firefox", "ignored malformed line"} {
		if !strings.Contains(allWarnings, expected) {
			test.Errorf("warnings %q missing %q", allWarnings, expected)
		}
	}
}

func TestReadProfilesFileMissingIsNotAnError(test *testing.T) {
	_, found, readError := ReadProfilesFile(test.TempDir())
	if found || readError != nil {
		test.Errorf("found=%v error=%v; want false, nil", found, readError)
	}
}

func TestReadCompatibility(test *testing.T) {
	profileDir := test.TempDir()
	writeCompatibility(test, profileDir, "157.0", "/opt/firefox")

	compatibility, found, readError := ReadCompatibility(profileDir)
	if readError != nil || !found {
		test.Fatalf("found=%v error=%v", found, readError)
	}

	if compatibility.AppVersion != "157.0" || compatibility.MajorVersion != 157 || compatibility.LastPlatformDir != "/opt/firefox" {
		test.Errorf("got %+v", compatibility)
	}

	_, found, readError = ReadCompatibility(test.TempDir())
	if found || readError != nil {
		test.Errorf("never-run profile: found=%v error=%v", found, readError)
	}
}
