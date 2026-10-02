package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/apply"
	"github.com/headwalluk/foxtrainer/internal/firefox"
	"github.com/headwalluk/foxtrainer/internal/wizard"
)

// startDefaults is the start choice meaning "recommended defaults"; other values index answers.File.Instances.
const startDefaults = -1

// runWizard walks through instance, starting point, questions and review, then saves and optionally applies.
func runWizard(environment Environment, accessible bool) error {
	if !accessible && !isTerminal(environment.Stdin) {
		return errNoTerminal
	}

	inventory := discover(environment)

	instances := inventory.Instances()
	if len(instances) == 0 {
		return errors.New("no Firefox instances found; start Firefox once so it creates a profile, then try again")
	}

	answersFile, loadError := answers.Load(environment.Config.Paths.AnswersFile)
	if loadError != nil {
		return loadError
	}

	asker := wizard.New(environment.Context, environment.Stdin, environment.Stdout, accessible)

	instanceIndex, chooseError := asker.ChooseInstance(instanceChoices(instances, answersFile))
	if chooseError != nil {
		return cancelledIsFine(environment, chooseError)
	}

	instance := instances[instanceIndex]

	startIndex, startError := asker.ChooseStart(startChoices(instance, answersFile))
	if startError != nil {
		return cancelledIsFine(environment, startError)
	}

	start := defaultAnswers(environment)
	if startIndex != startDefaults {
		start = answersFile.Instances[startIndex].Answers
	}

	chosen, questionsError := asker.AskQuestions(start, dictionaryHint(environment, environment.Config.Paths.HunspellDirs))
	if questionsError != nil {
		return cancelledIsFine(environment, questionsError)
	}

	if validationError := validateAnswers(chosen); validationError != nil {
		return validationError
	}

	return reviewAndSave(environment, asker, instance, chosen, answersFile)
}

// reviewAndSave previews the result, asks what to do, then saves and optionally applies.
func reviewAndSave(environment Environment, asker wizard.Wizard, instance firefox.Instance, chosen answers.Answers, answersFile answers.File) error {
	loaded, catalogueError := loadCatalogue(environment, false, true)
	if catalogueError != nil {
		return catalogueError
	}

	options := applyOptions(environment, false)

	prepared, prepareError := apply.Prepare(loaded.catalogue, loaded.upstream, targetFor(instance), chosen, options)
	if prepareError != nil {
		return prepareError
	}

	cannotApply := ""
	if instance.Profile.Lock.InUse {
		cannotApply = "Firefox is running on this profile, so it can't be applied now. Save, quit Firefox, then run `foxtrainer apply`."
	}

	action, reviewError := asker.Review(reviewSummary(prepared), cannotApply == "", cannotApply)
	if reviewError != nil {
		return cancelledIsFine(environment, reviewError)
	}

	if action == wizard.ActionCancel {
		return cancelledIsFine(environment, wizard.ErrCancelled)
	}

	answersFile.Upsert(answers.Instance{InstallPath: instance.Install.Dir, ProfilePath: instance.Profile.Profile.Dir, Answers: chosen})

	if saveError := answers.Save(environment.Config.Paths.AnswersFile, answersFile); saveError != nil {
		return saveError
	}

	var report strings.Builder

	fmt.Fprintf(&report, "Saved answers for %s.\n", instanceLabel(instance))

	if action == wizard.ActionApply {
		committed, commitError := apply.Commit(prepared, options)
		if commitError != nil {
			return commitError
		}

		writeApplied(&report, prepared, committed, environment.Config.Paths.HomeDir)
	} else {
		report.WriteString("Run `foxtrainer apply` to write them to the profile.\n")
	}

	_, writeError := fmt.Fprint(environment.Stdout, report.String())

	return writeError
}

// cancelledIsFine turns a user cancellation into a message and success; other errors pass through.
func cancelledIsFine(environment Environment, flowError error) error {
	// Input running out is a failure for scripts; only a deliberate cancel succeeds.
	if !errors.Is(flowError, wizard.ErrCancelled) || errors.Is(flowError, wizard.ErrInputEnded) {
		return flowError
	}

	_, writeError := fmt.Fprintln(environment.Stdout, "Cancelled; nothing was saved.")

	return writeError
}

// instanceChoices labels each instance, marking configured and running ones.
func instanceChoices(instances []firefox.Instance, answersFile answers.File) []wizard.Choice {
	choices := make([]wizard.Choice, 0, len(instances))

	for index, instance := range instances {
		var marks []string

		if _, configured := answersFile.Find(instance.Install.Dir, instance.Profile.Profile.Dir); configured {
			marks = append(marks, "configured")
		}

		if instance.IsDefault {
			marks = append(marks, "default profile")
		}

		if instance.Profile.Lock.InUse {
			marks = append(marks, "running")
		}

		label := instanceLabel(instance)
		if len(marks) > 0 {
			label += " (" + strings.Join(marks, ", ") + ")"
		}

		choices = append(choices, wizard.Choice{Label: label, Value: index})
	}

	return choices
}

// startChoices offers this instance's saved answers, every other configured instance's, and the defaults.
func startChoices(instance firefox.Instance, answersFile answers.File) []wizard.Choice {
	var choices []wizard.Choice

	var others []wizard.Choice

	for index, configured := range answersFile.Instances {
		isThis := configured.InstallPath == instance.Install.Dir && configured.ProfilePath == instance.Profile.Profile.Dir

		if isThis {
			choices = append(choices, wizard.Choice{Label: "This instance's saved answers (" + describeAnswers(configured.Answers) + ")", Value: index})
		} else {
			others = append(others, wizard.Choice{
				Label: fmt.Sprintf("Copy from %s in %s (%s)", filepath.Base(configured.ProfilePath), configured.InstallPath, describeAnswers(configured.Answers)),
				Value: index,
			})
		}
	}

	choices = append(choices, others...)

	return append(choices, wizard.Choice{Label: "Recommended defaults", Value: startDefaults})
}

// dictionaryHint lists the spellcheck dictionaries installed on this system, for the languages question.
func dictionaryHint(environment Environment, hunspellDirs []string) string {
	found := map[string]bool{}

	for _, hunspellDir := range hunspellDirs {
		dictionaries, globError := filepath.Glob(filepath.Join(hunspellDir, "*.dic"))
		if globError != nil {
			environment.Logger.Warnf("cannot list dictionaries in %s: %v", hunspellDir, globError)

			continue
		}

		for _, dictionary := range dictionaries {
			found[strings.ReplaceAll(strings.TrimSuffix(filepath.Base(dictionary), ".dic"), "_", "-")] = true
		}
	}

	tags := make([]string, 0, len(found))
	for tag := range found {
		tags = append(tags, tag)
	}

	sort.Strings(tags)

	hint := "Used for spellchecking and sent to websites. No system dictionaries found; install e.g. hunspell-en-gb."
	if len(tags) > 0 {
		hint = "Used for spellchecking and sent to websites. Dictionaries on this system: " + strings.Join(tags, ", ")
	}

	return hint
}

// reviewSummary describes what applying would change.
func reviewSummary(prepared apply.Prepared) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "%s\n", prepared.Target.Label)
	fmt.Fprintf(&builder, "%s\n", describeAnswers(prepared.Answers))
	fmt.Fprintf(&builder, "%d prefs from %d groups: %d added, %d changed, %d removed, %d reset in prefs.js",
		prepared.PrefCount(), len(prepared.Plan.Groups), len(prepared.Added), len(prepared.Changed), len(prepared.Removed), len(prepared.ResetNames))

	for _, note := range prepared.Notes {
		fmt.Fprintf(&builder, "\nNote: %s", note)
	}

	return builder.String()
}

// isTerminal reports whether input is an interactive terminal.
func isTerminal(input any) bool {
	file, isFile := input.(*os.File)
	if !isFile {
		return false
	}

	info, statError := file.Stat()

	return statError == nil && info.Mode()&os.ModeCharDevice != 0
}
