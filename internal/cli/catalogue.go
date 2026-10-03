package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
	"strings"

	cataloguedata "github.com/headwalluk/foxtrainer/catalogue"
	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/catalogue"
	"github.com/headwalluk/foxtrainer/internal/language"
	"github.com/headwalluk/foxtrainer/internal/sources"
)

// errCatalogueInvalid is returned when the catalogue fails validation.
var errCatalogueInvalid = errors.New("catalogue is invalid")

// runCatalogue dispatches "catalogue check" and "catalogue show".
func runCatalogue(environment Environment, arguments []string) error {
	subcommand := ""
	if len(arguments) > 0 {
		subcommand = arguments[0]
	}

	var runError error

	switch subcommand {
	case "check":
		runError = runCatalogueCheck(environment, arguments[1:])
	case "show":
		runError = runCatalogueShow(environment, arguments[1:])
	default:
		runError = fmt.Errorf("usage: foxtrainer catalogue check|show [options] (got %q)", subcommand)
	}

	return runError
}

// loadedCatalogue is the catalogue plus its verified upstream files.
type loadedCatalogue struct {
	catalogue catalogue.Catalogue
	upstream  catalogue.Upstream
}

// loadCatalogue loads the embedded catalogue and fetches its pinned upstream files, reporting progress unless quiet.
func loadCatalogue(environment Environment, offline, quiet bool) (loadedCatalogue, error) {
	loaded, loadError := catalogue.Load(cataloguedata.Files)
	if loadError != nil {
		return loadedCatalogue{}, fmt.Errorf("load catalogue: %w", loadError)
	}

	store := sources.Store{CacheDir: environment.Config.Paths.CacheDir, Offline: offline}

	upstream, fetched, fetchError := catalogue.LoadUpstream(environment.Context, loaded, store)

	for _, file := range fetched {
		source := loaded.Sources[file.SourceID]
		origin := "downloaded"

		if file.FromCache {
			origin = "cached"
		}

		announce := environment.Logger.Infof
		if quiet {
			announce = environment.Logger.Debugf
		}

		announce("Definitions from %s %s: %s (%s)", source.Title, source.Tag, file.File, origin)
	}

	return loadedCatalogue{catalogue: loaded, upstream: upstream}, fetchError
}

// runCatalogueCheck validates the catalogue against its pinned upstream files.
func runCatalogueCheck(environment Environment, arguments []string) error {
	flags := flag.NewFlagSet("catalogue check", flag.ContinueOnError)
	flags.SetOutput(environment.Stderr)
	offline := flags.Bool("offline", false, "use cached upstream files only")

	if parseError := flags.Parse(arguments); parseError != nil {
		return parseError
	}

	loaded, loadError := loadCatalogue(environment, *offline, false)
	if loadError != nil {
		return loadError
	}

	problems := catalogue.Validate(loaded.catalogue, loaded.upstream)
	for _, problem := range problems {
		environment.Logger.Errorf("%v", problem)
	}

	var checkError error

	if len(problems) > 0 {
		checkError = fmt.Errorf("%w: %d problem(s)", errCatalogueInvalid, len(problems))
	} else {
		_, checkError = fmt.Fprintf(environment.Stdout, "Catalogue %s is valid: %d groups, every upstream pref accounted for.\n",
			loaded.catalogue.CatalogueVersion, len(loaded.catalogue.Groups))
	}

	return checkError
}

// runCatalogueShow prints the prefs a set of answers resolves to.
func runCatalogueShow(environment Environment, arguments []string) error {
	flags := flag.NewFlagSet("catalogue show", flag.ContinueOnError)
	flags.SetOutput(environment.Stderr)

	chosen := answers.Answers{}
	flags.StringVar(&chosen.Feel, "feel", "balanced", "lean | balanced | full")
	flags.StringVar(&chosen.AI, "ai", "off", "off | local | all")
	flags.StringVar(&chosen.Privacy, "privacy", "strict", "standard | strict")
	flags.BoolVar(&chosen.HTTPSOnly, "https-only", true, "HTTPS-Only mode")
	languages := flags.String("languages", strings.Join(defaultAnswers(environment).Languages, ","), "comma-separated language tags")
	firefoxMajor := flags.Int("firefox", 0, "Firefox major version to target (0 = no version filtering)")
	offline := flags.Bool("offline", false, "use cached upstream files only")

	if parseError := flags.Parse(arguments); parseError != nil {
		return parseError
	}

	chosen.Languages = splitList(*languages)

	if validateError := validateAnswers(chosen); validateError != nil {
		return validateError
	}

	loaded, loadError := loadCatalogue(environment, *offline, false)
	if loadError != nil {
		return loadError
	}

	target := catalogue.Target{FirefoxMajor: *firefoxMajor, Platform: runtime.GOOS}

	plan, resolveError := catalogue.Resolve(loaded.catalogue, loaded.upstream, chosen, target, generators(environment))
	if resolveError != nil {
		return resolveError
	}

	_, writeError := io.WriteString(environment.Stdout, renderPlan(loaded.catalogue, plan, chosen, target))

	return writeError
}

// renderPlan formats a resolved plan with each pref's origin.
func renderPlan(loaded catalogue.Catalogue, plan catalogue.Plan, chosen answers.Answers, target catalogue.Target) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "Catalogue %s", loaded.CatalogueVersion)

	for _, sourceID := range []string{"betterfox"} {
		if source, known := loaded.Sources[sourceID]; known {
			fmt.Fprintf(&builder, " · %s %s (%s)", source.Title, source.Tag, source.Commit[:7])
		}
	}

	firefoxLabel := "any Firefox"
	if target.FirefoxMajor > 0 {
		firefoxLabel = fmt.Sprintf("Firefox %d", target.FirefoxMajor)
	}

	fmt.Fprintf(&builder, "\nAnswers: feel=%s ai=%s privacy=%s https_only=%t · %s on %s\n",
		chosen.Feel, chosen.AI, chosen.Privacy, chosen.HTTPSOnly, firefoxLabel, target.Platform)

	total := 0

	for _, planned := range plan.Groups {
		fmt.Fprintf(&builder, "\n[%s] %s (%d prefs)\n", planned.Group.ID, planned.Group.Title, len(planned.Prefs))

		for _, pref := range planned.Prefs {
			fmt.Fprintf(&builder, "  %-62s = %-12s %s\n", pref.Name, pref.Value, originLabel(pref.Origin))
		}

		total += len(planned.Prefs)
	}

	if len(plan.Skipped) > 0 {
		builder.WriteString("\nSkipped (not in this Firefox version or platform):\n")

		for _, skipped := range plan.Skipped {
			fmt.Fprintf(&builder, "  %s\n", skipped.Name)
		}
	}

	if len(plan.Notes) > 0 {
		builder.WriteString("\nNotes:\n")

		for _, note := range plan.Notes {
			fmt.Fprintf(&builder, "  %s\n", note)
		}
	}

	fmt.Fprintf(&builder, "\nTotal: %d prefs from %d groups\n", total, len(plan.Groups))

	return builder.String()
}

// originLabel describes where a pref's value came from.
func originLabel(origin catalogue.Origin) string {
	label := "foxtrainer"

	if origin.Source != "foxtrainer" {
		label = fmt.Sprintf("%s %s:%d", origin.Source, origin.File, origin.Line)
	}

	return label
}

// generators returns the code-backed group generators, configured from the resolved paths.
func generators(environment Environment) map[string]catalogue.Generator {
	languageGenerator := language.Generator{HunspellDirs: environment.Config.Paths.HunspellDirs}

	return map[string]catalogue.Generator{"language": languageGenerator.Generate}
}
