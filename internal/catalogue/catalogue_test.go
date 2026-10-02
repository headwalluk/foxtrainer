package catalogue

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	cataloguedata "github.com/headwalluk/foxtrainer/catalogue"
	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/prefs"
	"github.com/headwalluk/foxtrainer/internal/sources/betterfox"
)

const pinnedBetterfoxDir = "../../testdata/upstream/betterfox/067172a4b0dc90e78e5b8b94d9abfe6430c6a7be"

// pinnedUpstream parses the pinned Betterfox 154.0 test data, as LoadUpstream would.
func pinnedUpstream(test *testing.T) Upstream {
	test.Helper()

	files := map[string][]betterfox.Record{}

	for _, fileName := range []string{"user.js", "Peskyfox.js"} {
		content, readError := os.ReadFile(filepath.Join(pinnedBetterfoxDir, fileName))
		if readError != nil {
			test.Fatal(readError)
		}

		records, parseError := betterfox.Parse(fileName, content)
		if parseError != nil {
			test.Fatal(parseError)
		}

		files[fileName] = records
	}

	return Upstream{"betterfox": files}
}

// embeddedCatalogue loads the catalogue compiled into the binary.
func embeddedCatalogue(test *testing.T) Catalogue {
	test.Helper()

	loaded, loadError := Load(cataloguedata.Files)
	if loadError != nil {
		test.Fatalf("load embedded catalogue: %v", loadError)
	}

	return loaded
}

func TestEmbeddedCatalogueIsValid(test *testing.T) {
	loaded := embeddedCatalogue(test)

	for _, problem := range Validate(loaded, pinnedUpstream(test)) {
		test.Error(problem)
	}

	if len(loaded.Groups) < 10 {
		test.Errorf("want the full set of groups, got %d", len(loaded.Groups))
	}
}

// defaultAnswers are the agreed defaults for a new instance.
func defaultAnswers() answers.Answers {
	return answers.Answers{Feel: "balanced", AI: "off", Privacy: "strict", HTTPSOnly: true, Languages: []string{"en-GB"}}
}

func TestResolveDefaultAnswers(test *testing.T) {
	loaded := embeddedCatalogue(test)

	plan, resolveError := Resolve(loaded, pinnedUpstream(test), defaultAnswers(), Target{FirefoxMajor: 158, Platform: "linux"}, nil)
	if resolveError != nil {
		test.Fatal(resolveError)
	}

	groupIDs := map[string]bool{}
	values := map[string]PlannedPref{}

	for _, planned := range plan.Groups {
		groupIDs[planned.Group.ID] = true

		for _, pref := range planned.Prefs {
			if _, duplicate := values[pref.Name]; duplicate {
				test.Errorf("%s planned twice", pref.Name)
			}

			values[pref.Name] = pref
		}
	}

	for _, wanted := range []string{"baseline.telemetry", "feel.quiet-ui", "feel.new-tab", "privacy.strict", "https.only", "ai.block-all"} {
		if !groupIDs[wanted] {
			test.Errorf("group %s should be active", wanted)
		}
	}

	for _, unwanted := range []string{"feel.lean-extras", "ai.block-cloud"} {
		if groupIDs[unwanted] {
			test.Errorf("group %s should not be active", unwanted)
		}
	}

	checks := map[string]prefs.Value{
		"browser.ai.control.default":                           prefs.String("blocked"),
		"browser.ai.control.speechRecognition":                 prefs.String("blocked"),
		"dom.security.https_only_mode":                         prefs.Bool(true),
		"browser.contentblocking.category":                     prefs.String("strict"),
		"browser.newtabpage.activity-stream.feeds.weatherfeed": prefs.Bool(false),
	}

	for name, want := range checks {
		if got, found := values[name]; !found || got.Value != want {
			test.Errorf("%s: got %+v, want %s", name, got, want)
		}
	}

	telemetry := values["toolkit.telemetry.enabled"]
	if telemetry.Origin.Source != "betterfox" || telemetry.Origin.Subsection != "TELEMETRY" || telemetry.Origin.Line == 0 {
		test.Errorf("telemetry origin should point into Betterfox user.js: %+v", telemetry.Origin)
	}

	if _, written := values["browser.profiles.enabled"]; written {
		test.Error("browser.profiles.enabled must never be planned")
	}
}

func TestResolveFiltersByFirefoxVersion(test *testing.T) {
	loaded := embeddedCatalogue(test)

	plan, resolveError := Resolve(loaded, pinnedUpstream(test), defaultAnswers(), Target{FirefoxMajor: 140, Platform: "linux"}, nil)
	if resolveError != nil {
		test.Fatal(resolveError)
	}

	skippedNames := map[string]bool{}
	for _, skipped := range plan.Skipped {
		skippedNames[skipped.Name] = true
	}

	for _, name := range []string{"browser.ai.control.default", "browser.ai.control.smartWindow", "browser.ai.control.speechRecognition"} {
		if !skippedNames[name] {
			test.Errorf("%s does not exist in Firefox 140 and should be skipped", name)
		}
	}
}

func TestResolveFullFatKeepsNewTabButNotSponsored(test *testing.T) {
	loaded := embeddedCatalogue(test)
	fullFat := answers.Answers{Feel: "full", AI: "all", Privacy: "standard", HTTPSOnly: false}

	plan, resolveError := Resolve(loaded, pinnedUpstream(test), fullFat, Target{FirefoxMajor: 158, Platform: "linux"}, nil)
	if resolveError != nil {
		test.Fatal(resolveError)
	}

	active := map[string]bool{}
	for _, planned := range plan.Groups {
		active[planned.Group.ID] = true
	}

	for groupID, want := range map[string]bool{
		"baseline.sponsored": true, "baseline.telemetry": true, "feel.performance": true,
		"feel.new-tab": false, "privacy.strict": false, "https.only": false, "ai.block-all": false,
	} {
		if active[groupID] != want {
			test.Errorf("%s active = %v, want %v", groupID, active[groupID], want)
		}
	}
}

// catalogueWith builds an in-memory catalogue from catalogue.toml text and group files.
func catalogueWith(test *testing.T, groupFiles map[string]string) Catalogue {
	test.Helper()

	embedded, readError := cataloguedata.Files.ReadFile("catalogue.toml")
	if readError != nil {
		test.Fatal(readError)
	}

	memory := fstest.MapFS{"catalogue.toml": {Data: embedded}}
	for name, content := range groupFiles {
		memory["groups/"+name] = &fstest.MapFile{Data: []byte(content)}
	}

	loaded, loadError := Load(memory)
	if loadError != nil {
		test.Fatal(loadError)
	}

	return loaded
}

// problemText joins problems for substring checks.
func problemText(problems []error) string {
	var texts []string
	for _, problem := range problems {
		texts = append(texts, problem.Error())
	}

	return strings.Join(texts, "\n")
}

func TestValidateCatchesMistakes(test *testing.T) {
	loaded := catalogueWith(test, map[string]string{
		"a.toml": `
id = "test.one"
title = "One"
description = "One"
when = { feel = ["balanced", "sideways"] }

[[include]]
source = "betterfox"
file = "user.js"
section = "PESKYFOX"
subsection = "MOZILLA UI"

[[pick]]
source = "betterfox"
file = "Peskyfox.js"
prefs = ["browser.urlbar.suggest.engines"]

[prefs]
"browser.ml.enable" = false
`,
		"b.toml": `
id = "test.two"
title = "Two"
description = "Two"
when = { feel = ["balanced"] }

[prefs]
"browser.ml.enable" = true
`,
	})

	text := problemText(Validate(loaded, pinnedUpstream(test)))

	for _, expected := range []string{
		`feel has no answer sideways`,
		`browser.profiles.enabled is forbidden`,
		`browser.urlbar.suggest.engines appears`,
		`conflict: test.one sets browser.ml.enable = false but test.two sets true`,
		`toolkit.telemetry.enabled (SECUREFOX › TELEMETRY) is not used by any group`,
		`browser.profiles.enabled is claimed more than once`,
	} {
		if !strings.Contains(text, expected) {
			test.Errorf("problems are missing %q", expected)
		}
	}
}

func TestValidateAllowsOverridesAndDisjointGroups(test *testing.T) {
	loaded := catalogueWith(test, map[string]string{
		"a.toml": "id = \"test.off\"\ntitle = \"x\"\ndescription = \"x\"\nwhen = { ai = [\"off\"] }\n[prefs]\n\"browser.ml.enable\" = false\n",
		"b.toml": "id = \"test.all\"\ntitle = \"x\"\ndescription = \"x\"\nwhen = { ai = [\"all\"] }\n[prefs]\n\"browser.ml.enable\" = true\n",
		"c.toml": "id = \"test.over\"\ntitle = \"x\"\ndescription = \"x\"\noverrides = [\"test.off\"]\n[prefs]\n\"browser.ml.enable\" = true\n",
	})

	if conflicts := problemText(Validate(loaded, pinnedUpstream(test))); strings.Contains(conflicts, "conflict:") {
		test.Errorf("unexpected conflicts:\n%s", conflicts)
	}

	plan, resolveError := Resolve(loaded, pinnedUpstream(test), answers.Answers{AI: "off"}, Target{}, nil)
	if resolveError != nil {
		test.Fatal(resolveError)
	}

	for _, planned := range plan.Groups {
		if planned.Group.ID == "test.off" && len(planned.Prefs) != 0 {
			test.Errorf("test.off's pref should be overridden by test.over: %+v", planned.Prefs)
		}
	}
}

func TestLoadRejectsBadValues(test *testing.T) {
	memory := fstest.MapFS{
		"catalogue.toml": {Data: []byte("schema_version = 1\ncatalogue_version = \"x\"\n")},
		"groups/a.toml": {Data: []byte("id = \"test.bad\"\ntitle = \"x\"\ndescription = \"x\"\ncolour = \"red\"\n[prefs]\n" +
			"\"a.float\" = 1.5\n\"a.table\" = { value = true, since = \"soon\" }\n")},
	}

	_, loadError := Load(memory)
	if loadError == nil {
		test.Fatal("want an error")
	}

	for _, expected := range []string{"unknown keys: colour", "a.float", "since must be a Firefox major version"} {
		if !strings.Contains(loadError.Error(), expected) {
			test.Errorf("error is missing %q:\n%v", expected, loadError)
		}
	}
}
