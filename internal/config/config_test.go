package config

import (
	"strings"
	"testing"

	"github.com/headwalluk/foxtrainer/internal/logger"
)

// lookupFrom returns a LookupFunc backed by values.
func lookupFrom(values map[string]string) LookupFunc {
	return func(name string) (string, bool) {
		value, found := values[name]

		return value, found
	}
}

func TestLoadDefaults(test *testing.T) {
	loaded, loadError := Load(lookupFrom(map[string]string{"HOME": "/home/fox"}), "linux")
	if loadError != nil {
		test.Fatalf("unexpected error: %v", loadError)
	}

	if loaded.LogLevel != logger.LevelInfo {
		test.Errorf("log level: got %v, want info", loaded.LogLevel)
	}

	if loaded.Paths.AnswersFile != "/home/fox/.config/foxtrainer/config.toml" {
		test.Errorf("answers file: %s", loaded.Paths.AnswersFile)
	}
}

func TestLoadReadsLogLevel(test *testing.T) {
	loaded, loadError := Load(lookupFrom(map[string]string{"HOME": "/home/fox", EnvLogLevel: "DEBUG"}), "linux")
	if loadError != nil {
		test.Fatalf("unexpected error: %v", loadError)
	}

	if loaded.LogLevel != logger.LevelDebug {
		test.Errorf("log level: got %v, want debug", loaded.LogLevel)
	}
}

func TestLoadReportsEveryProblemAtOnce(test *testing.T) {
	_, loadError := Load(lookupFrom(map[string]string{EnvLogLevel: "chatty"}), "linux")
	if loadError == nil {
		test.Fatal("want an error")
	}

	message := loadError.Error()
	for _, expected := range []string{EnvLogLevel, "chatty", "HOME is not set"} {
		if !strings.Contains(message, expected) {
			test.Errorf("error %q is missing %q", message, expected)
		}
	}
}

func TestLoadUsesUserProfileOnWindows(test *testing.T) {
	values := map[string]string{
		"USERPROFILE":  `C:\Users\fox`,
		"APPDATA":      `C:\Users\fox\AppData\Roaming`,
		"LOCALAPPDATA": `C:\Users\fox\AppData\Local`,
	}

	_, loadError := Load(lookupFrom(values), "windows")
	if loadError != nil {
		test.Errorf("unexpected error: %v", loadError)
	}
}

func TestLocaleLanguages(test *testing.T) {
	cases := []struct {
		allValue, messagesValue, langValue string
		want                               []string
	}{
		{"", "", "en_GB.UTF-8", []string{"en-GB", "en"}},
		{"fr_CA.UTF-8", "", "en_GB.UTF-8", []string{"fr-CA", "fr"}},
		{"C", "de_DE@euro", "", []string{"de-DE", "de"}},
		{"", "", "", nil},
	}

	for _, testCase := range cases {
		got := localeLanguages(testCase.allValue, testCase.messagesValue, testCase.langValue)
		if strings.Join(got, ",") != strings.Join(testCase.want, ",") {
			test.Errorf("%+v: got %v", testCase, got)
		}
	}
}
