// Package language generates the language and spelling prefs from the chosen languages and the system's dictionaries.
//
// On Linux, Firefox's own builds ship only an en-US dictionary, and language packs add none.
// Pointing spellchecker.dictionary_path at the system hunspell folder makes the distro's
// dictionaries available; see docs/how-it-works.md ("Language and spelling").
package language

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/catalogue"
	"github.com/headwalluk/foxtrainer/internal/prefs"
)

// Generator builds the "language" generated group.
type Generator struct {
	HunspellDirs []string // system dictionary folders, most preferred first
}

// Generate returns the language prefs for chosen on target, plus notes about missing dictionaries.
func (generator Generator) Generate(chosen answers.Answers, target catalogue.Target) ([]catalogue.PlannedPref, []string, error) {
	if len(chosen.Languages) == 0 {
		return nil, []string{"language and spelling: no languages chosen; Firefox's defaults apply"}, nil
	}

	planned := []catalogue.PlannedPref{
		ownPref("intl.accept_languages", strings.Join(chosen.Languages, ", ")),
	}

	var notes []string

	if target.Platform != "linux" {
		notes = append(notes, "language and spelling: dictionary set-up on "+target.Platform+" arrives with macOS and Windows support")

		return planned, notes, nil
	}

	dictionaryDir, found, missing, scanError := generator.findDictionaries(spellcheckLanguages(chosen.Languages))
	if scanError != nil {
		return nil, nil, scanError
	}

	if len(found) > 0 {
		planned = append(planned,
			ownPref("spellchecker.dictionary_path", dictionaryDir),
			ownPref("spellchecker.dictionary", strings.Join(found, ",")),
		)
	}

	for _, tag := range missing {
		notes = append(notes, fmt.Sprintf("no system dictionary for %s; on Debian or Ubuntu install it with: sudo apt install hunspell-%s",
			tag, strings.ToLower(tag)))
	}

	return planned, notes, nil
}

// findDictionaries picks the first folder holding a dictionary for the preferred language, and lists which languages it covers.
func (generator Generator) findDictionaries(tags []string) (string, []string, []string, error) {
	var problems []error

	chosenDir := ""

	var found []string

	for _, dictionaryDir := range generator.HunspellDirs {
		covered, dirError := coveredLanguages(dictionaryDir, tags)
		problems = append(problems, dirError)

		if len(covered) > 0 && (chosenDir == "" || len(covered) > len(found)) {
			chosenDir = dictionaryDir
			found = covered
		}
	}

	var missing []string

	for _, tag := range tags {
		if !contains(found, tag) {
			missing = append(missing, tag)
		}
	}

	return chosenDir, found, missing, errors.Join(problems...)
}

// coveredLanguages lists the tags that have both a .dic and an .aff file in dictionaryDir.
func coveredLanguages(dictionaryDir string, tags []string) ([]string, error) {
	var covered []string

	var problems []error

	for _, tag := range tags {
		baseName := strings.ReplaceAll(tag, "-", "_")
		dicPresent, dicError := exists(filepath.Join(dictionaryDir, baseName+".dic"))
		affPresent, affError := exists(filepath.Join(dictionaryDir, baseName+".aff"))
		problems = append(problems, dicError, affError)

		if dicPresent && affPresent {
			covered = append(covered, tag)
		}
	}

	return covered, errors.Join(problems...)
}

// spellcheckLanguages drops a bare language ("en") when a regional variant of it ("en-GB") is also chosen.
func spellcheckLanguages(languages []string) []string {
	var tags []string

	for _, tag := range languages {
		primary, _, hasRegion := strings.Cut(tag, "-")

		hasVariant := false

		for _, other := range languages {
			if !hasRegion && strings.HasPrefix(other, primary+"-") {
				hasVariant = true

				break
			}
		}

		if !hasVariant {
			tags = append(tags, tag)
		}
	}

	return tags
}

// exists reports whether path exists; errors other than not-exist are returned.
func exists(path string) (bool, error) {
	_, statError := os.Stat(path)

	var present bool

	var existsError error

	switch {
	case statError == nil:
		present = true
	case errors.Is(statError, fs.ErrNotExist):
		// Absent.
	default:
		existsError = fmt.Errorf("check %s: %w", path, statError)
	}

	return present, existsError
}

// ownPref makes a foxtrainer-originated string pref.
func ownPref(name, value string) catalogue.PlannedPref {
	return catalogue.PlannedPref{Name: name, Value: prefs.String(value), Origin: catalogue.Origin{Source: "foxtrainer"}}
}

// contains reports whether items holds wanted.
func contains(items []string, wanted string) bool {
	found := false

	for _, item := range items {
		if item == wanted {
			found = true

			break
		}
	}

	return found
}
