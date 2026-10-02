// Package wizard asks the configure questions in the terminal: choose an instance, choose a starting point, answer, review.
package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"charm.land/huh/v2"

	"github.com/headwalluk/foxtrainer/internal/answers"
)

// ErrCancelled means the user left the wizard without saving.
var ErrCancelled = errors.New("cancelled; nothing was saved")

// languageTagPattern accepts BCP 47 tags such as en, en-GB or zh-Hant-TW.
var languageTagPattern = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})*$`)

// Wizard runs forms against a terminal, or plain line prompts in accessible mode.
type Wizard struct {
	Context    context.Context
	Output     io.Writer
	Accessible bool
	terminal   io.Reader   // the real input; interactive forms need the terminal itself to read keys
	lines      *lineReader // accessible mode only
}

// New returns a Wizard reading answers from input.
func New(ctx context.Context, input io.Reader, output io.Writer, accessible bool) Wizard {
	return Wizard{Context: ctx, Output: output, Accessible: accessible, terminal: input, lines: newLineReader(input)}
}

// Choice is one option shown in a list.
type Choice struct {
	Label string
	Value int
}

// Action is what to do once the answers are reviewed.
type Action int

// Review actions.
const (
	ActionApply Action = iota + 1
	ActionSaveOnly
	ActionCancel
)

// run shows one form, translating an aborted form into ErrCancelled.
func (wizard Wizard) run(groups ...*huh.Group) error {
	form := huh.NewForm(groups...).WithAccessible(wizard.Accessible).WithOutput(wizard.Output)

	if wizard.Accessible {
		// Plain text for screen readers (no escape codes), and one line per form so answers aren't swallowed.
		form = form.WithTheme(huh.ThemeFunc(huh.ThemeBase)).WithInput(wizard.lines)
	} else {
		// The terminal itself, so huh can switch it to raw mode and read arrow keys.
		form = form.WithInput(wizard.terminal)
	}

	runError := form.RunWithContext(wizard.Context)

	switch {
	case errors.Is(runError, huh.ErrUserAborted):
		runError = ErrCancelled
	case runError == nil && wizard.Accessible && wizard.lines.ended:
		runError = fmt.Errorf("%w: %w", ErrCancelled, ErrInputEnded)
	}

	return runError
}

// ChooseInstance asks which instance to configure and returns the chosen Choice value.
func (wizard Wizard) ChooseInstance(choices []Choice) (int, error) {
	selected := choices[0].Value

	if len(choices) == 1 {
		return selected, nil
	}

	selectField := huh.NewSelect[int]().
		Title("Which Firefox do you want to configure?").
		Description("An instance is a Firefox install plus one of its profiles.").
		Options(options(choices)...).
		Value(&selected)

	return selected, wizard.run(huh.NewGroup(selectField))
}

// ChooseStart asks where the answers should start from and returns the chosen Choice value.
func (wizard Wizard) ChooseStart(choices []Choice) (int, error) {
	selected := choices[0].Value

	if len(choices) == 1 {
		return selected, nil
	}

	selectField := huh.NewSelect[int]().
		Title("Start from").
		Description("Every answer can still be changed on the next screens.").
		Options(options(choices)...).
		Value(&selected)

	return selected, wizard.run(huh.NewGroup(selectField))
}

// AskQuestions asks the configuration questions, pre-filled from start.
func (wizard Wizard) AskQuestions(start answers.Answers, dictionaryHint string) (answers.Answers, error) {
	chosen := start
	languages := strings.Join(start.Languages, ", ")

	feel := huh.NewSelect[string]().
		Title("Overall feel").
		Options(
			huh.NewOption("Lean: strip Firefox back (no What's New page, fewer URL bar extras)", "lean"),
			huh.NewOption("Balanced: quiet and tidy, everything useful kept (recommended)", "balanced"),
			huh.NewOption("Full-fat: keep all the features; just no telemetry, ads or nags", "full"),
		).
		Value(&chosen.Feel)

	artificialIntelligence := huh.NewSelect[string]().
		Title("AI features").
		Options(
			huh.NewOption("Off: block every AI feature, including translations", "off"),
			huh.NewOption("Local only: keep on-device features like translations; block cloud AI", "local"),
			huh.NewOption("Everything: leave Firefox's AI features available", "all"),
		).
		Value(&chosen.AI)

	privacy := huh.NewSelect[string]().
		Title("Privacy").
		Description("Hardened (arkenfox-based, breaks some sites) is coming soon.").
		Options(
			huh.NewOption("Standard: Firefox's protections, plus the essentials", "standard"),
			huh.NewOption("Strict: stronger tracking protection, no disk cache; occasionally a site needs an exception (recommended)", "strict"),
		).
		Value(&chosen.Privacy)

	httpsOnly := huh.NewConfirm().
		Title("HTTPS-Only mode?").
		Description("Refuse insecure http:// connections unless you allow a site.").
		Affirmative("Yes").
		Negative("No").
		Value(&chosen.HTTPSOnly)

	languageInput := huh.NewInput().
		Title("Languages, preferred first").
		Description(dictionaryHint).
		Validate(ValidateLanguages).
		Value(&languages)

	formError := wizard.run(
		huh.NewGroup(feel),
		huh.NewGroup(artificialIntelligence),
		huh.NewGroup(privacy, httpsOnly),
		huh.NewGroup(languageInput),
	)

	chosen.Languages = SplitLanguages(languages)

	return chosen, formError
}

// Review shows the summary and asks what to do; apply is offered only when canApply.
func (wizard Wizard) Review(summary string, canApply bool, cannotApplyReason string) (Action, error) {
	choices := []huh.Option[Action]{
		huh.NewOption("Save and apply now", ActionApply),
		huh.NewOption("Save only (run `foxtrainer apply` later)", ActionSaveOnly),
		huh.NewOption("Cancel without saving", ActionCancel),
	}

	description := summary
	selected := ActionApply

	if !canApply {
		choices = choices[1:]
		selected = ActionSaveOnly
		description = summary + "\n\n" + cannotApplyReason
	}

	selectField := huh.NewSelect[Action]().
		Title("Review").
		Description(description).
		Options(choices...).
		Value(&selected)

	return selected, wizard.run(huh.NewGroup(selectField))
}

// ValidateLanguages accepts a comma-separated list of language tags.
func ValidateLanguages(text string) error {
	tags := SplitLanguages(text)

	var validateError error

	if len(tags) == 0 {
		validateError = errors.New("enter at least one language, e.g. en-GB")
	}

	for _, tag := range tags {
		if !languageTagPattern.MatchString(tag) {
			validateError = fmt.Errorf("%q is not a language tag like en-GB or fr", tag)

			break
		}
	}

	return validateError
}

// SplitLanguages splits "en-GB, en" into tags, dropping blanks.
func SplitLanguages(text string) []string {
	var tags []string

	for part := range strings.SplitSeq(text, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}

	return tags
}

// options converts Choices into huh options.
func options(choices []Choice) []huh.Option[int] {
	converted := make([]huh.Option[int], 0, len(choices))
	for _, choice := range choices {
		converted = append(converted, huh.NewOption(choice.Label, choice.Value))
	}

	return converted
}
