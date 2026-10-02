package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/catalogue"
	"github.com/headwalluk/foxtrainer/internal/firefox"
)

// errNoTerminal is returned when the wizard is asked for but stdin is not a terminal.
var errNoTerminal = errors.New("the configure wizard needs a terminal; pass --accessible for plain prompts, or --profile with answer flags")

// runConfigure saves answers for one instance from flags; unset answers keep their saved or default value.
func runConfigure(environment Environment, arguments []string) error {
	flags := flag.NewFlagSet("configure", flag.ContinueOnError)
	flags.SetOutput(environment.Stderr)

	profile := flags.String("profile", "", "profile name or folder (see `foxtrainer list`)")
	install := flags.String("install", "", "install folder, when the profile is used by more than one install")
	feel := flags.String("feel", "", "lean | balanced | full")
	artificialIntelligence := flags.String("ai", "", "off | local | all")
	privacy := flags.String("privacy", "", "standard | strict | hardened")
	httpsOnly := flags.Bool("https-only", true, "HTTPS-Only mode")
	languages := flags.String("languages", "", "comma-separated language tags, preferred first, e.g. en-GB,fr")
	accessible := flags.Bool("accessible", false, "wizard: plain line-by-line prompts (screen readers, scripts)")

	if parseError := flags.Parse(arguments); parseError != nil {
		return parseError
	}

	if *profile == "" {
		return runWizard(environment, *accessible)
	}

	inventory := discover(environment)

	instance, findError := findInstance(inventory, *profile, *install)
	if findError != nil {
		return findError
	}

	answersFile, loadError := answers.Load(environment.Config.Paths.AnswersFile)
	if loadError != nil {
		return loadError
	}

	saved, hasSaved := answersFile.Find(instance.Install.Dir, instance.Profile.Profile.Dir)

	chosen := defaultAnswers(environment)
	if hasSaved {
		chosen = saved.Answers
	}

	flags.Visit(func(setFlag *flag.Flag) {
		switch setFlag.Name {
		case "feel":
			chosen.Feel = *feel
		case "ai":
			chosen.AI = *artificialIntelligence
		case "privacy":
			chosen.Privacy = *privacy
		case "https-only":
			chosen.HTTPSOnly = *httpsOnly
		case "languages":
			chosen.Languages = splitList(*languages)
		}
	})

	if validationError := validateAnswers(chosen); validationError != nil {
		return validationError
	}

	answersFile.Upsert(answers.Instance{InstallPath: instance.Install.Dir, ProfilePath: instance.Profile.Profile.Dir, Answers: chosen})

	if saveError := answers.Save(environment.Config.Paths.AnswersFile, answersFile); saveError != nil {
		return saveError
	}

	_, writeError := fmt.Fprintf(environment.Stdout, "Saved answers for %s: %s\nRun `foxtrainer apply` to write them to the profile.\n",
		instanceLabel(instance), describeAnswers(chosen))

	return writeError
}

// defaultAnswers are the starting answers for a new instance (decisions E1–E7).
func defaultAnswers(environment Environment) answers.Answers {
	languages := environment.Config.Languages
	if len(languages) == 0 {
		languages = []string{"en-US", "en"}
	}

	return answers.Answers{Feel: "balanced", AI: "off", Privacy: "strict", HTTPSOnly: true, Languages: languages}
}

// validateAnswers checks each answer against the catalogue's questions, reporting every problem.
func validateAnswers(chosen answers.Answers) error {
	var problems []error

	checks := map[string]any{"feel": chosen.Feel, "ai": chosen.AI, "privacy": chosen.Privacy}
	for _, question := range []string{"feel", "ai", "privacy"} {
		if !slices.Contains(catalogue.Questions[question], checks[question]) {
			problems = append(problems, fmt.Errorf("--%s: %q is not one of %v", question, checks[question], catalogue.Questions[question]))
		}
	}

	if chosen.Privacy == "hardened" {
		problems = append(problems, errors.New("--privacy hardened is coming soon (it needs the arkenfox reader); use strict for now"))
	}

	return errors.Join(problems...)
}

// findInstance picks the instance whose profile matches name or folder, narrowed by install if given.
func findInstance(inventory firefox.Inventory, profile, install string) (firefox.Instance, error) {
	var matches []firefox.Instance

	for _, instance := range inventory.Instances() {
		profileMatches := instance.Profile.Profile.Name == profile ||
			instance.Profile.Profile.Dir == filepath.Clean(profile) ||
			filepath.Base(instance.Profile.Profile.Dir) == profile
		installMatches := install == "" || instance.Install.Dir == filepath.Clean(install)

		if profileMatches && installMatches {
			matches = append(matches, instance)
		}
	}

	var found firefox.Instance

	var findError error

	switch len(matches) {
	case 0:
		findError = fmt.Errorf("no instance uses profile %q; run `foxtrainer list` to see instances", profile)
	case 1:
		found = matches[0]
	default:
		var installs []string
		for _, match := range matches {
			installs = append(installs, match.Install.Dir)
		}

		findError = fmt.Errorf("profile %q is used by several installs; add --install with one of: %s", profile, strings.Join(installs, ", "))
	}

	return found, findError
}

// discover runs instance discovery with the configured paths, logging warnings.
func discover(environment Environment) firefox.Inventory {
	resolved := environment.Config.Paths

	inventory := firefox.Discover(firefox.DiscoverOptions{
		ProfileRoots:          resolved.FirefoxRoots,
		InstallSearchPatterns: resolved.InstallSearchPatterns,
		ExecutableDirs:        resolved.ExecutableDirs,
	})

	for _, warning := range inventory.Warnings {
		environment.Logger.Warnf("%s", warning)
	}

	return inventory
}

// instanceLabel names an instance for people, e.g. "Firefox Developer Edition 158.0 › dev-edition-default-1".
func instanceLabel(instance firefox.Instance) string {
	return fmt.Sprintf("%s %s › %s", instance.Install.Name, instance.Install.Version, instance.Profile.Profile.Name)
}

// describeAnswers summarises answers on one line.
func describeAnswers(chosen answers.Answers) string {
	return fmt.Sprintf("feel=%s ai=%s privacy=%s https_only=%t languages=%s",
		chosen.Feel, chosen.AI, chosen.Privacy, chosen.HTTPSOnly, strings.Join(chosen.Languages, ","))
}

// splitList splits a comma-separated flag value, dropping blanks.
func splitList(text string) []string {
	var items []string

	for item := range strings.SplitSeq(text, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return items
}
