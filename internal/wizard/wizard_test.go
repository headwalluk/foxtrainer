package wizard

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/headwalluk/foxtrainer/internal/answers"
)

// accessibleWizard answers prompts from the given lines.
func accessibleWizard(test *testing.T, lines ...string) Wizard {
	test.Helper()

	return New(test.Context(), strings.NewReader(strings.Join(lines, "\n")+"\n"), io.Discard, true)
}

func TestAskQuestionsAcrossSeveralForms(test *testing.T) {
	start := answers.Answers{Feel: "balanced", AI: "off", Privacy: "strict", HTTPSOnly: true, Languages: []string{"en-GB", "en"}}

	chosen, askError := accessibleWizard(test, "1", "2", "1", "n", "en-GB, fr").AskQuestions(start, "hint")
	if askError != nil {
		test.Fatal(askError)
	}

	want := answers.Answers{Feel: "lean", AI: "local", Privacy: "standard", HTTPSOnly: false, Languages: []string{"en-GB", "fr"}}
	if !reflect.DeepEqual(chosen, want) {
		test.Errorf("got %+v\nwant %+v", chosen, want)
	}
}

func TestRunningOutOfInputCancelsInsteadOfAcceptingDefaults(test *testing.T) {
	asker := New(test.Context(), strings.NewReader(""), io.Discard, true)

	_, reviewError := asker.Review("summary", true, "")
	if !errors.Is(reviewError, ErrCancelled) || !errors.Is(reviewError, ErrInputEnded) {
		test.Errorf("empty input must cancel, got %v", reviewError)
	}
}

func TestReviewWithoutApplyOffersOnlySaveOrCancel(test *testing.T) {
	action, reviewError := accessibleWizard(test, "1").Review("summary", false, "Firefox is running")
	if reviewError != nil || action != ActionSaveOnly {
		test.Errorf("first option must be save only when apply is impossible: %v %v", action, reviewError)
	}
}

func TestSingleChoicesAreNotAsked(test *testing.T) {
	asker := New(test.Context(), strings.NewReader(""), io.Discard, true)

	selected, chooseError := asker.ChooseInstance([]Choice{{Label: "only", Value: 7}})
	if chooseError != nil || selected != 7 {
		test.Errorf("got %d, %v", selected, chooseError)
	}
}

func TestValidateLanguages(test *testing.T) {
	for _, good := range []string{"en-GB", "en-GB, en", "fr", "zh-Hant-TW"} {
		if validateError := ValidateLanguages(good); validateError != nil {
			test.Errorf("%q: %v", good, validateError)
		}
	}

	for _, bad := range []string{"", " , ", "english", "en_GB", "en-GB; fr"} {
		if ValidateLanguages(bad) == nil {
			test.Errorf("%q should be rejected", bad)
		}
	}
}

func TestLineReaderHandsOutOneLineAtATime(test *testing.T) {
	reader := newLineReader(strings.NewReader("first\nsecond"))
	buffer := make([]byte, 64)

	count, readError := reader.Read(buffer)
	if readError != nil || string(buffer[:count]) != "first\n" {
		test.Errorf("first read: %q %v", buffer[:count], readError)
	}

	count, readError = reader.Read(buffer)
	if readError != nil || string(buffer[:count]) != "second" || !reader.ended {
		test.Errorf("second read: %q %v ended=%v", buffer[:count], readError, reader.ended)
	}

	if _, readError = reader.Read(buffer); !errors.Is(readError, io.EOF) {
		test.Errorf("want EOF, got %v", readError)
	}
}
