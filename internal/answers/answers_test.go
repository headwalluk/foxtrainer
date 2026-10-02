package answers

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsEmpty(test *testing.T) {
	loaded, loadError := Load(filepath.Join(test.TempDir(), "config.toml"))
	if loadError != nil {
		test.Fatalf("unexpected error: %v", loadError)
	}

	if loaded.SchemaVersion != SchemaVersion || len(loaded.Instances) != 0 {
		test.Errorf("want an empty file at the current schema, got %+v", loaded)
	}
}

func TestSaveLoadRoundTrip(test *testing.T) {
	path := filepath.Join(test.TempDir(), "nested", "config.toml")
	original := File{
		Instances: []Instance{
			{
				InstallPath: "/usr/lib/firefox-devedition",
				ProfilePath: "/home/fox/.mozilla/firefox/0ujjyyoj.dev-edition-default-1",
				Answers: Answers{
					Feel:      "balanced",
					AI:        "off",
					Privacy:   "strict",
					HTTPSOnly: true,
					Languages: []string{"en-GB", "fr"},
				},
			},
		},
	}

	if saveError := Save(path, original); saveError != nil {
		test.Fatalf("save: %v", saveError)
	}

	info, statError := os.Stat(path)
	if statError != nil {
		test.Fatalf("stat: %v", statError)
	}

	if info.Mode().Perm() != 0o600 {
		test.Errorf("permissions: got %o, want 600", info.Mode().Perm())
	}

	loaded, loadError := Load(path)
	if loadError != nil {
		test.Fatalf("load: %v", loadError)
	}

	original.SchemaVersion = SchemaVersion
	if !reflect.DeepEqual(loaded, original) {
		test.Errorf("round trip mismatch:\n got %+v\nwant %+v", loaded, original)
	}
}

func TestLoadRejectsUnknownKeysAndSchema(test *testing.T) {
	path := filepath.Join(test.TempDir(), "config.toml")
	content := "schema_version = 99\n[[instance]]\ninstall_path = \"/x\"\ncolour = \"red\"\n"

	if writeError := os.WriteFile(path, []byte(content), 0o600); writeError != nil {
		test.Fatalf("write: %v", writeError)
	}

	_, loadError := Load(path)
	if loadError == nil {
		test.Fatal("want an error")
	}

	for _, expected := range []string{"colour", "schema_version 99"} {
		if !strings.Contains(loadError.Error(), expected) {
			test.Errorf("error %q is missing %q", loadError, expected)
		}
	}
}
