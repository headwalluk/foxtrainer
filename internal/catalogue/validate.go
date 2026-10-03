package catalogue

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/sources/betterfox"
)

// Questions lists every answer a when table may test, with its allowed values.
var Questions = map[string][]any{
	"feel":       {"lean", "balanced", "full"},
	"ai":         {"off", "local", "all"},
	"privacy":    {"standard", "strict", "hardened"},
	"https_only": {true, false},
}

// KnownGenerators lists the generator names a group may declare.
var KnownGenerators = []string{"language"}

// forbiddenPrefixes are pref names foxtrainer never writes: profile-group state (docs/catalogue.md).
var forbiddenPrefixes = []string{"browser.profiles.", "toolkit.profiles."}

var (
	commitPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Pattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	groupIDPattern = regexp.MustCompile(`^[a-z]+(\.[a-z0-9-]+)+$`)
)

// IsForbidden reports whether foxtrainer must never write name.
func IsForbidden(name string) bool {
	forbidden := false

	for _, prefix := range forbiddenPrefixes {
		if strings.HasPrefix(name, prefix) {
			forbidden = true

			break
		}
	}

	return forbidden
}

// Validate checks the catalogue against the upstream files and returns every problem found.
func Validate(loaded Catalogue, upstream Upstream) []error {
	var problems []error

	if loaded.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Errorf("schema_version %d is not supported (want %d)", loaded.SchemaVersion, SchemaVersion))
	}

	if loaded.CatalogueVersion == "" {
		problems = append(problems, errors.New("catalogue_version is missing"))
	}

	problems = append(problems, validateSources(loaded)...)
	problems = append(problems, validateGroups(loaded, upstream)...)
	problems = append(problems, validateCoverage(loaded, upstream)...)
	problems = append(problems, validateConflicts(loaded, upstream)...)

	return problems
}

// validateSources checks each source's pin.
func validateSources(loaded Catalogue) []error {
	var problems []error

	for _, sourceID := range sortedKeys(loaded.Sources) {
		source := loaded.Sources[sourceID]
		label := "source " + sourceID

		if !commitPattern.MatchString(source.Commit) {
			problems = append(problems, fmt.Errorf("%s: commit must be a full 40-character lower-case SHA", label))
		}

		if !strings.Contains(source.RawURL, "{commit}") || !strings.Contains(source.RawURL, "{file}") {
			problems = append(problems, fmt.Errorf("%s: raw_url must contain {commit} and {file}", label))
		}

		if source.Licence == "" || source.Copyright == "" {
			problems = append(problems, fmt.Errorf("%s: licence and copyright are required for attribution", label))
		}

		for fileName, digest := range source.Files {
			if !sha256Pattern.MatchString(digest) {
				problems = append(problems, fmt.Errorf("%s: %s: sha256 must be 64 lower-case hex characters", label, fileName))
			}
		}

		for _, coverageFile := range source.Coverage {
			if _, pinned := source.Files[coverageFile]; !pinned {
				problems = append(problems, fmt.Errorf("%s: coverage file %s is not pinned", label, coverageFile))
			}
		}
	}

	return problems
}

// validateGroups checks each group's identity, conditions, references and expanded prefs.
func validateGroups(loaded Catalogue, upstream Upstream) []error {
	var problems []error

	seen := map[string]string{}

	for index := range loaded.Groups {
		group := &loaded.Groups[index]
		label := group.FileName

		if !groupIDPattern.MatchString(group.ID) {
			problems = append(problems, fmt.Errorf("%s: id %q must look like area.name", label, group.ID))
		}

		if otherFile, duplicate := seen[group.ID]; duplicate {
			problems = append(problems, fmt.Errorf("%s: id %s is also used by %s", label, group.ID, otherFile))
		}

		seen[group.ID] = label

		if group.Title == "" || group.Description == "" {
			problems = append(problems, fmt.Errorf("%s: title and description are required", label))
		}

		problems = append(problems, validateWhen(label, group.When)...)
		problems = append(problems, validateGroupContent(loaded, group, upstream)...)
	}

	for index := range loaded.Groups {
		for _, overriddenID := range loaded.Groups[index].Overrides {
			if loaded.GroupByID(overriddenID) == nil {
				problems = append(problems, fmt.Errorf("%s: overrides unknown group %s", loaded.Groups[index].FileName, overriddenID))
			}
		}
	}

	return problems
}

// validateWhen checks a when table against Questions.
func validateWhen(label string, when map[string][]any) []error {
	var problems []error

	for question, values := range when {
		allowed, known := Questions[question]
		if !known {
			problems = append(problems, fmt.Errorf("%s: when: unknown question %q", label, question))

			continue
		}

		if len(values) == 0 {
			problems = append(problems, fmt.Errorf("%s: when: %s lists no answers", label, question))
		}

		for _, value := range values {
			if !slices.Contains(allowed, value) {
				problems = append(problems, fmt.Errorf("%s: when: %s has no answer %v", label, question, value))
			}
		}
	}

	return problems
}

// validateGroupContent checks generators, upstream references, exclusions and forbidden prefs.
func validateGroupContent(loaded Catalogue, group *Group, upstream Upstream) []error {
	var problems []error

	label := group.FileName

	if group.Generator != "" {
		if !slices.Contains(KnownGenerators, group.Generator) {
			problems = append(problems, fmt.Errorf("%s: unknown generator %q", label, group.Generator))
		}

		if len(group.Includes)+len(group.Picks)+len(group.Prefs) > 0 {
			problems = append(problems, fmt.Errorf("%s: a generated group cannot also list prefs", label))
		}

		return problems
	}

	for _, include := range group.Includes {
		problems = append(problems, checkPinned(loaded, label, include.Source, include.File)...)

		for _, excludedName := range include.Exclude {
			if !sectionHasPref(upstream, include, excludedName) {
				problems = append(problems, fmt.Errorf("%s: include excludes %s, which is not in %s › %s", label, excludedName, include.Section, include.Subsection))
			}
		}
	}

	for _, pick := range group.Picks {
		problems = append(problems, checkPinned(loaded, label, pick.Source, pick.File)...)
	}

	expanded, expandError := group.Expand(upstream)
	if expandError != nil {
		problems = append(problems, fmt.Errorf("%s: %w", label, expandError))
	}

	for _, pref := range expanded {
		if IsForbidden(pref.Name) {
			problems = append(problems, fmt.Errorf("%s: %s is forbidden (profile-group state)", label, pref.Name))
		}
	}

	return problems
}

// checkPinned verifies that sourceID exists and pins fileName.
func checkPinned(loaded Catalogue, label, sourceID, fileName string) []error {
	var problems []error

	source, known := loaded.Sources[sourceID]

	switch {
	case !known:
		problems = append(problems, fmt.Errorf("%s: unknown source %q", label, sourceID))
	case source.Files[fileName] == "":
		problems = append(problems, fmt.Errorf("%s: %s %s is not pinned", label, sourceID, fileName))
	}

	return problems
}

// sectionHasPref reports whether name is an active pref inside the include's section.
func sectionHasPref(upstream Upstream, include Include, name string) bool {
	found := false

	for _, record := range upstream[include.Source][include.File] {
		inScope := record.State == betterfox.StateActive && record.Section == include.Section &&
			(include.Subsection == "" || record.Subsection == include.Subsection)

		if inScope && record.Name == name {
			found = true

			break
		}
	}

	return found
}

// validateCoverage checks that every active pref in each coverage file is used exactly once or excluded.
func validateCoverage(loaded Catalogue, upstream Upstream) []error {
	var problems []error

	for _, sourceID := range sortedKeys(loaded.Sources) {
		for _, coverageFile := range loaded.Sources[sourceID].Coverage {
			claims := coverageClaims(loaded, upstream, sourceID, coverageFile)

			for _, record := range upstream[sourceID][coverageFile] {
				if record.State != betterfox.StateActive {
					continue
				}

				claimants := claims[record.Name]

				switch len(claimants) {
				case 0:
					problems = append(problems, fmt.Errorf("coverage: %s %s:%d %s (%s › %s) is not used by any group or excluded",
						sourceID, coverageFile, record.Line, record.Name, record.Section, record.Subsection))
				case 1:
					// Exactly one owner: correct.
				default:
					problems = append(problems, fmt.Errorf("coverage: %s %s is claimed more than once: %s",
						sourceID, record.Name, strings.Join(claimants, ", ")))
				}
			}

			for _, exclusion := range loaded.Excluded {
				if exclusion.Source == sourceID && !fileHasActive(upstream[sourceID][coverageFile], exclusion.Pref) {
					problems = append(problems, fmt.Errorf("excluded: %s %s is not an active pref in %s", sourceID, exclusion.Pref, coverageFile))
				}
			}
		}
	}

	return problems
}

// coverageClaims maps each pref name in a coverage file to the groups (or "excluded") that use it.
func coverageClaims(loaded Catalogue, upstream Upstream, sourceID, fileName string) map[string][]string {
	claims := map[string][]string{}

	for index := range loaded.Groups {
		group := &loaded.Groups[index]

		for _, include := range group.Includes {
			if include.Source != sourceID || include.File != fileName {
				continue
			}

			matched, _ := include.records(upstream) // errors are reported by validateGroups
			for _, record := range matched {
				claims[record.Name] = append(claims[record.Name], group.ID)
			}
		}

		for _, pick := range group.Picks {
			if pick.Source != sourceID || pick.File != fileName {
				continue
			}

			for _, name := range pick.Prefs {
				claims[name] = append(claims[name], group.ID)
			}
		}
	}

	for _, exclusion := range loaded.Excluded {
		if exclusion.Source == sourceID {
			claims[exclusion.Pref] = append(claims[exclusion.Pref], "excluded")
		}
	}

	return claims
}

// fileHasActive reports whether records contain an active pref called name.
func fileHasActive(records []betterfox.Record, name string) bool {
	return slices.ContainsFunc(records, func(record betterfox.Record) bool {
		return record.State == betterfox.StateActive && record.Name == name
	})
}

// validateConflicts reports prefs given different values by groups that can be active together.
func validateConflicts(loaded Catalogue, upstream Upstream) []error {
	var problems []error

	expanded := make([][]PlannedPref, len(loaded.Groups))
	for index := range loaded.Groups {
		expanded[index], _ = loaded.Groups[index].Expand(upstream) // errors are reported by validateGroups
	}

	for first := range loaded.Groups {
		for second := first + 1; second < len(loaded.Groups); second++ {
			firstGroup, secondGroup := &loaded.Groups[first], &loaded.Groups[second]
			if !canCoexist(firstGroup, secondGroup) || overrides(firstGroup, secondGroup) || overrides(secondGroup, firstGroup) {
				continue
			}

			for _, firstPref := range expanded[first] {
				for _, secondPref := range expanded[second] {
					if firstPref.Name == secondPref.Name && firstPref.Value != secondPref.Value {
						problems = append(problems, fmt.Errorf("conflict: %s sets %s = %s but %s sets %s; add overrides to one group",
							firstGroup.ID, firstPref.Name, firstPref.Value, secondGroup.ID, secondPref.Value))
					}
				}
			}
		}
	}

	return problems
}

// canCoexist reports whether some set of answers switches on both groups.
func canCoexist(first, second *Group) bool {
	coexist := true

	for question, firstAllowed := range first.When {
		secondAllowed, constrained := second.When[question]
		if !constrained {
			continue
		}

		overlap := slices.ContainsFunc(firstAllowed, func(value any) bool { return slices.Contains(secondAllowed, value) })
		if !overlap {
			coexist = false

			break
		}
	}

	return coexist
}

// overrides reports whether group declares that it overrides other.
func overrides(group, other *Group) bool {
	return slices.Contains(group.Overrides, other.ID)
}
