// Package catalogue loads, validates and resolves the catalogue that maps answers to groups to prefs.
//
// The design is documented in docs/how-it-works.md ("The catalogue").
package catalogue

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/headwalluk/foxtrainer/internal/prefs"
)

// SchemaVersion is the catalogue layout this build understands.
const SchemaVersion = 1

// Catalogue is the parsed catalogue: sources, exclusions and groups.
type Catalogue struct {
	SchemaVersion    int               `toml:"schema_version"`
	CatalogueVersion string            `toml:"catalogue_version"`
	Sources          map[string]Source `toml:"sources"`
	Excluded         []Exclusion       `toml:"excluded"`
	Groups           []Group           `toml:"-"`
}

// Source is an upstream pref collection pinned to one commit.
type Source struct {
	Format     string            `toml:"format"`
	Title      string            `toml:"title"`
	Repository string            `toml:"repository"`
	Licence    string            `toml:"licence"`
	Copyright  string            `toml:"copyright"`
	Tag        string            `toml:"tag"`
	Commit     string            `toml:"commit"`
	RawURL     string            `toml:"raw_url"`
	Coverage   []string          `toml:"coverage"`
	Files      map[string]string `toml:"files"`
}

// Exclusion records an upstream pref deliberately not used, and why.
type Exclusion struct {
	Source string `toml:"source"`
	Pref   string `toml:"pref"`
	Reason string `toml:"reason"`
}

// Group is a named, explained set of prefs switched on by answers.
type Group struct {
	ID          string           `toml:"id"`
	Title       string           `toml:"title"`
	Description string           `toml:"description"`
	When        map[string][]any `toml:"when"`
	Generator   string           `toml:"generator"`
	Overrides   []string         `toml:"overrides"`
	Includes    []Include        `toml:"include"`
	Picks       []Pick           `toml:"pick"`
	RawPrefs    map[string]any   `toml:"prefs"`

	FileName string     `toml:"-"`
	Prefs    []PrefSpec `toml:"-"` // RawPrefs, converted and sorted by name
}

// Include takes every active pref in an upstream section or subsection.
type Include struct {
	Source     string   `toml:"source"`
	File       string   `toml:"file"`
	Section    string   `toml:"section"`
	Subsection string   `toml:"subsection"`
	Exclude    []string `toml:"exclude"`
}

// Pick takes named prefs, active or commented out, from an upstream file.
type Pick struct {
	Source string   `toml:"source"`
	File   string   `toml:"file"`
	Prefs  []string `toml:"prefs"`
}

// PrefSpec is a pref value given directly in a group, with optional version and platform limits.
type PrefSpec struct {
	Name      string
	Value     prefs.Value
	Since     int      // first Firefox major it applies to; 0 = any
	Until     int      // last Firefox major it applies to; 0 = any
	Platforms []string // runtime.GOOS values; empty = all
}

// Load reads catalogue.toml and groups/*.toml from catalogueFiles, rejecting unknown keys.
func Load(catalogueFiles fs.FS) (Catalogue, error) {
	var loaded Catalogue

	parseError, unknownKeysError := decodeStrict(catalogueFiles, "catalogue.toml", &loaded)
	if parseError != nil {
		return Catalogue{}, parseError
	}

	problems := []error{unknownKeysError}

	groupFiles, globError := fs.Glob(catalogueFiles, "groups/*.toml")
	if globError != nil {
		return Catalogue{}, fmt.Errorf("list groups: %w", globError)
	}

	sort.Strings(groupFiles)

	for _, groupFile := range groupFiles {
		var group Group

		groupParseError, groupKeysError := decodeStrict(catalogueFiles, groupFile, &group)
		if groupParseError != nil {
			problems = append(problems, groupParseError)

			continue
		}

		problems = append(problems, groupKeysError)

		group.FileName = path.Base(groupFile)

		specs, specProblems := convertPrefSpecs(group.RawPrefs)
		group.Prefs = specs

		for _, specProblem := range specProblems {
			problems = append(problems, fmt.Errorf("%s: %w", group.FileName, specProblem))
		}

		loaded.Groups = append(loaded.Groups, group)
	}

	sort.SliceStable(loaded.Groups, func(left, right int) bool {
		return areaRank(loaded.Groups[left].ID) < areaRank(loaded.Groups[right].ID)
	})

	return loaded, errors.Join(problems...)
}

// areaOrder sets the order groups appear in plans and user.js, by the area prefix of their ID.
var areaOrder = []string{"baseline", "feel", "privacy", "https", "ai", "language"}

// areaRank returns a group's position in areaOrder; unknown areas sort last.
func areaRank(groupID string) int {
	area, _, _ := strings.Cut(groupID, ".")

	rank := len(areaOrder)

	for index, candidate := range areaOrder {
		if candidate == area {
			rank = index

			break
		}
	}

	return rank
}

// decodeStrict decodes one TOML file, returning a parse error and, separately, any keys the schema does not know.
func decodeStrict(catalogueFiles fs.FS, name string, target any) (error, error) {
	content, readError := fs.ReadFile(catalogueFiles, name)
	if readError != nil {
		return fmt.Errorf("read %s: %w", name, readError), nil
	}

	metadata, decodeError := toml.Decode(string(content), target)
	if decodeError != nil {
		return fmt.Errorf("parse %s: %w", name, decodeError), nil
	}

	var unknownKeys []string

	for _, key := range metadata.Undecoded() {
		keyText := key.String()
		// Pref names are free-form keys under [prefs]; everything else must be known.
		if !strings.HasPrefix(keyText, "prefs.") {
			unknownKeys = append(unknownKeys, keyText)
		}
	}

	var unknownKeysError error
	if len(unknownKeys) > 0 {
		unknownKeysError = fmt.Errorf("%s: unknown keys: %s", name, strings.Join(unknownKeys, ", "))
	}

	return nil, unknownKeysError
}

// convertPrefSpecs turns a group's [prefs] table into sorted PrefSpecs; values are scalars or {value, since, until, platforms}.
func convertPrefSpecs(rawPrefs map[string]any) ([]PrefSpec, []error) {
	var specs []PrefSpec

	var problems []error

	for name, raw := range rawPrefs {
		spec, specError := convertPrefSpec(name, raw)
		if specError != nil {
			problems = append(problems, fmt.Errorf("pref %s: %w", name, specError))

			continue
		}

		specs = append(specs, spec)
	}

	sort.Slice(specs, func(left, right int) bool { return specs[left].Name < specs[right].Name })

	return specs, problems
}

// convertPrefSpec converts one [prefs] entry.
func convertPrefSpec(name string, raw any) (PrefSpec, error) {
	table, isTable := raw.(map[string]any)
	if !isTable {
		value, valueError := prefs.FromTOML(raw)

		return PrefSpec{Name: name, Value: value}, valueError
	}

	spec := PrefSpec{Name: name}

	var problems []error

	for key, field := range table {
		switch key {
		case "value":
			value, valueError := prefs.FromTOML(field)
			spec.Value = value
			problems = append(problems, valueError)
		case "since", "until":
			version, isInt := field.(int64)
			if !isInt || version < 0 {
				problems = append(problems, fmt.Errorf("%s must be a Firefox major version", key))
			}

			if key == "since" {
				spec.Since = int(version)
			} else {
				spec.Until = int(version)
			}
		case "platforms":
			platforms, platformError := stringList(field)
			spec.Platforms = platforms
			problems = append(problems, platformError)
		default:
			problems = append(problems, fmt.Errorf("unknown field %q", key))
		}
	}

	if spec.Value.Kind == 0 {
		problems = append(problems, errors.New("missing value"))
	}

	return spec, errors.Join(problems...)
}

// stringList converts a TOML array of strings.
func stringList(field any) ([]string, error) {
	items, isList := field.([]any)
	if !isList {
		return nil, errors.New("must be a list of strings")
	}

	converted := make([]string, 0, len(items))

	var listError error

	for _, item := range items {
		text, isString := item.(string)
		if !isString {
			listError = errors.New("must be a list of strings")

			break
		}

		converted = append(converted, text)
	}

	return converted, listError
}

// GroupByID returns the group with id, or nil.
func (loaded Catalogue) GroupByID(id string) *Group {
	var found *Group

	for index := range loaded.Groups {
		if loaded.Groups[index].ID == id {
			found = &loaded.Groups[index]

			break
		}
	}

	return found
}
