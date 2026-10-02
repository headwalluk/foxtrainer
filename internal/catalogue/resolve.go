package catalogue

import (
	"errors"
	"fmt"
	"slices"

	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/prefs"
	"github.com/headwalluk/foxtrainer/internal/sources/betterfox"
)

// ownSource labels prefs whose values come from foxtrainer's own groups.
const ownSource = "foxtrainer"

// Origin says where a planned pref's value came from.
type Origin struct {
	Source     string // upstream source ID, or "foxtrainer"
	File       string
	Line       int
	Section    string
	Subsection string
}

// PlannedPref is one pref to write, with its origin and any version or platform limits.
type PlannedPref struct {
	Name      string
	Value     prefs.Value
	Origin    Origin
	Since     int
	Until     int
	Platforms []string
}

// PlannedGroup is an active group and the prefs it contributes.
type PlannedGroup struct {
	Group *Group
	Prefs []PlannedPref
}

// Target describes the Firefox the plan is for.
type Target struct {
	FirefoxMajor int    // 0 = unknown: no version filtering
	Platform     string // runtime.GOOS value
}

// Generator produces a generated group's prefs from the answers and target.
type Generator func(chosen answers.Answers, target Target) ([]PlannedPref, []string, error)

// Plan is the resolved result: active groups with their prefs, plus notes for the user.
type Plan struct {
	Groups  []PlannedGroup
	Skipped []PlannedPref // filtered out by version or platform
	Notes   []string
}

// Resolve selects the groups matching chosen and expands them into prefs for target.
//
// Three passes: expand every active group, drop prefs that an overriding group replaces,
// then keep the first group's copy of any pref two groups set to the same value.
func Resolve(loaded Catalogue, upstream Upstream, chosen answers.Answers, target Target, generators map[string]Generator) (Plan, error) {
	var plan Plan

	var problems []error

	for index := range loaded.Groups {
		group := &loaded.Groups[index]
		if !group.Matches(chosen) {
			continue
		}

		expanded, expandError := expandForPlan(group, upstream, chosen, target, generators, &plan.Notes)
		if expandError != nil {
			problems = append(problems, fmt.Errorf("group %s: %w", group.ID, expandError))

			continue
		}

		planned := PlannedGroup{Group: group}

		for _, pref := range expanded {
			if pref.AppliesTo(target) {
				planned.Prefs = append(planned.Prefs, pref)
			} else {
				plan.Skipped = append(plan.Skipped, pref)
			}
		}

		plan.Groups = append(plan.Groups, planned)
	}

	plan = applyOverrides(plan)
	plan = removeDuplicates(plan)

	return plan, errors.Join(problems...)
}

// removeDuplicates keeps the first group's copy of each pref; a differing later value becomes a note.
func removeDuplicates(plan Plan) Plan {
	firstSet := map[string]PlannedPref{}
	firstGroup := map[string]string{}

	for index := range plan.Groups {
		groupID := plan.Groups[index].Group.ID

		plan.Groups[index].Prefs = slices.DeleteFunc(plan.Groups[index].Prefs, func(pref PlannedPref) bool {
			earlier, seen := firstSet[pref.Name]
			if !seen {
				firstSet[pref.Name] = pref
				firstGroup[pref.Name] = groupID

				return false
			}

			if earlier.Value != pref.Value {
				plan.Notes = append(plan.Notes, fmt.Sprintf("%s: %s keeps %s from %s, not %s (catalogue conflict)",
					pref.Name, groupID, earlier.Value, firstGroup[pref.Name], pref.Value))
			}

			return true
		})
	}

	return plan
}

// expandForPlan expands a group, calling its generator when it has one.
func expandForPlan(group *Group, upstream Upstream, chosen answers.Answers, target Target, generators map[string]Generator, notes *[]string) ([]PlannedPref, error) {
	if group.Generator == "" {
		return group.Expand(upstream)
	}

	generator, known := generators[group.Generator]
	if !known {
		*notes = append(*notes, fmt.Sprintf("%s: generator %q is not available in this build", group.ID, group.Generator))

		return nil, nil
	}

	generated, generatedNotes, generateError := generator(chosen, target)
	*notes = append(*notes, generatedNotes...)

	return generated, generateError
}

// applyOverrides removes prefs from groups that an active group declares it overrides.
func applyOverrides(plan Plan) Plan {
	overridden := map[string]map[string]bool{} // overridden group ID -> pref names set by the overriding group

	for _, planned := range plan.Groups {
		for _, overriddenID := range planned.Group.Overrides {
			if overridden[overriddenID] == nil {
				overridden[overriddenID] = map[string]bool{}
			}

			for _, pref := range planned.Prefs {
				overridden[overriddenID][pref.Name] = true
			}
		}
	}

	for index := range plan.Groups {
		names := overridden[plan.Groups[index].Group.ID]
		if names == nil {
			continue
		}

		plan.Groups[index].Prefs = slices.DeleteFunc(plan.Groups[index].Prefs, func(pref PlannedPref) bool { return names[pref.Name] })
	}

	return plan
}

// Matches reports whether chosen satisfies every condition in the group's when table.
func (group *Group) Matches(chosen answers.Answers) bool {
	matches := true

	for question, allowed := range group.When {
		if !slices.Contains(allowed, answerValue(chosen, question)) {
			matches = false

			break
		}
	}

	return matches
}

// answerValue returns the answer to question in the form used by when tables.
func answerValue(chosen answers.Answers, question string) any {
	values := map[string]any{
		"feel":       chosen.Feel,
		"ai":         chosen.AI,
		"privacy":    chosen.Privacy,
		"https_only": chosen.HTTPSOnly,
	}

	return values[question]
}

// AppliesTo reports whether the pref's version and platform limits allow target.
func (pref PlannedPref) AppliesTo(target Target) bool {
	versionOK := target.FirefoxMajor == 0 ||
		((pref.Since == 0 || target.FirefoxMajor >= pref.Since) && (pref.Until == 0 || target.FirefoxMajor <= pref.Until))
	platformOK := len(pref.Platforms) == 0 || slices.Contains(pref.Platforms, target.Platform)

	return versionOK && platformOK
}

// Expand turns a non-generated group into concrete prefs: includes, then picks, then explicit prefs.
func (group *Group) Expand(upstream Upstream) ([]PlannedPref, error) {
	var expanded []PlannedPref

	var problems []error

	for _, include := range group.Includes {
		matched, includeError := include.records(upstream)
		problems = append(problems, includeError)

		for _, record := range matched {
			expanded = append(expanded, fromRecord(include.Source, record))
		}
	}

	for _, pick := range group.Picks {
		for _, name := range pick.Prefs {
			record, pickError := pick.record(upstream, name)
			if pickError != nil {
				problems = append(problems, pickError)

				continue
			}

			expanded = append(expanded, fromRecord(pick.Source, record))
		}
	}

	for _, spec := range group.Prefs {
		expanded = append(expanded, PlannedPref{
			Name: spec.Name, Value: spec.Value, Origin: Origin{Source: ownSource},
			Since: spec.Since, Until: spec.Until, Platforms: spec.Platforms,
		})
	}

	return expanded, errors.Join(problems...)
}

// fromRecord makes a PlannedPref from an upstream record.
func fromRecord(sourceID string, record betterfox.Record) PlannedPref {
	return PlannedPref{
		Name:  record.Name,
		Value: record.Value,
		Origin: Origin{
			Source: sourceID, File: record.File, Line: record.Line,
			Section: record.Section, Subsection: record.Subsection,
		},
	}
}

// records returns the active records the include selects, minus its exclusions.
func (include Include) records(upstream Upstream) ([]betterfox.Record, error) {
	fileRecords, found := upstream[include.Source][include.File]
	if !found {
		return nil, fmt.Errorf("include: %s %s is not loaded", include.Source, include.File)
	}

	var matched []betterfox.Record

	for _, record := range fileRecords {
		inScope := record.State == betterfox.StateActive &&
			record.Section == include.Section &&
			(include.Subsection == "" || record.Subsection == include.Subsection)

		if inScope && !slices.Contains(include.Exclude, record.Name) {
			matched = append(matched, record)
		}
	}

	var includeError error
	if len(matched) == 0 {
		includeError = fmt.Errorf("include: no active prefs in %s %s › %s", include.File, include.Section, include.Subsection)
	}

	return matched, includeError
}

// record returns the single usable record called name in the pick's file.
func (pick Pick) record(upstream Upstream, name string) (betterfox.Record, error) {
	fileRecords, found := upstream[pick.Source][pick.File]
	if !found {
		return betterfox.Record{}, fmt.Errorf("pick: %s %s is not loaded", pick.Source, pick.File)
	}

	var matches []betterfox.Record

	for _, record := range fileRecords {
		if record.Name == name && record.State != betterfox.StateMalformed {
			matches = append(matches, record)
		}
	}

	var chosen betterfox.Record

	var pickError error

	switch len(matches) {
	case 0:
		pickError = fmt.Errorf("pick: %s is not in %s", name, pick.File)
	case 1:
		chosen = matches[0]
	default:
		pickError = fmt.Errorf("pick: %s appears %d times in %s; give the value explicitly in [prefs]", name, len(matches), pick.File)
	}

	return chosen, pickError
}
