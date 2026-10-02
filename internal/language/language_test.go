package language

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/catalogue"
	"github.com/headwalluk/foxtrainer/internal/prefs"
)

// dictionaryDir creates a folder holding .dic and .aff files for each base name.
func dictionaryDir(test *testing.T, baseNames ...string) string {
	test.Helper()

	folder := test.TempDir()

	for _, baseName := range baseNames {
		for _, extension := range []string{".dic", ".aff"} {
			if writeError := os.WriteFile(filepath.Join(folder, baseName+extension), []byte("x"), 0o644); writeError != nil {
				test.Fatal(writeError)
			}
		}
	}

	return folder
}

// generated runs the generator and returns its prefs as a name -> value map, plus notes.
func generated(test *testing.T, generator Generator, languages []string, platform string) (map[string]prefs.Value, []string) {
	test.Helper()

	planned, notes, generateError := generator.Generate(answers.Answers{Languages: languages}, catalogue.Target{Platform: platform})
	if generateError != nil {
		test.Fatal(generateError)
	}

	values := map[string]prefs.Value{}
	for _, pref := range planned {
		values[pref.Name] = pref.Value
	}

	return values, notes
}

func TestBritishEnglishOnDebian(test *testing.T) {
	hunspell := dictionaryDir(test, "en_GB", "en_US", "fr")
	generator := Generator{HunspellDirs: []string{filepath.Join(test.TempDir(), "missing"), hunspell}}

	values, notes := generated(test, generator, []string{"en-GB", "en"}, "linux")

	want := map[string]prefs.Value{
		"intl.accept_languages":        prefs.String("en-GB, en"),
		"spellchecker.dictionary_path": prefs.String(hunspell),
		"spellchecker.dictionary":      prefs.String("en-GB"),
	}
	if !reflect.DeepEqual(values, want) {
		test.Errorf("got %v\nwant %v", values, want)
	}

	if len(notes) != 0 {
		test.Errorf("unexpected notes: %v", notes)
	}
}

func TestSeveralLanguagesAndMissingDictionaries(test *testing.T) {
	generator := Generator{HunspellDirs: []string{dictionaryDir(test, "en_GB", "fr")}}

	values, notes := generated(test, generator, []string{"en-GB", "fr", "de-DE"}, "linux")

	if values["spellchecker.dictionary"] != prefs.String("en-GB,fr") {
		test.Errorf("dictionary: %v", values["spellchecker.dictionary"])
	}

	if len(notes) != 1 || !strings.Contains(notes[0], "sudo apt install hunspell-de-de") {
		test.Errorf("want a note suggesting hunspell-de-de, got %v", notes)
	}
}

func TestNoDictionariesAtAll(test *testing.T) {
	values, notes := generated(test, Generator{HunspellDirs: []string{test.TempDir()}}, []string{"en-GB"}, "linux")

	if _, set := values["spellchecker.dictionary_path"]; set || values["intl.accept_languages"] != prefs.String("en-GB") {
		test.Errorf("got %v", values)
	}

	if len(notes) != 1 || !strings.Contains(notes[0], "hunspell-en-gb") {
		test.Errorf("notes: %v", notes)
	}
}

func TestOtherPlatformsOnlySetAcceptLanguages(test *testing.T) {
	values, notes := generated(test, Generator{HunspellDirs: []string{dictionaryDir(test, "en_GB")}}, []string{"en-GB"}, "darwin")

	if len(values) != 1 || len(notes) != 1 {
		test.Errorf("macOS: got %v, notes %v", values, notes)
	}
}

func TestSpellcheckLanguages(test *testing.T) {
	cases := map[string][]string{
		"en-GB,en":    {"en-GB"},
		"en":          {"en"},
		"fr,en-GB,en": {"fr", "en-GB"},
		"pt-BR,pt-PT": {"pt-BR", "pt-PT"},
	}

	for input, want := range cases {
		if got := spellcheckLanguages(strings.Split(input, ",")); !reflect.DeepEqual(got, want) {
			test.Errorf("%s: got %v, want %v", input, got, want)
		}
	}
}
