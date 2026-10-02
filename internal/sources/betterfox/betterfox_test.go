package betterfox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/headwalluk/foxtrainer/internal/prefs"
)

const pinnedDir = "../../../testdata/upstream/betterfox/067172a4b0dc90e78e5b8b94d9abfe6430c6a7be"

// parsePinned parses one file of the pinned Betterfox 154.0 test data.
func parsePinned(test *testing.T, fileName string) []Record {
	test.Helper()

	content, readError := os.ReadFile(filepath.Join(pinnedDir, fileName))
	if readError != nil {
		test.Fatal(readError)
	}

	records, parseError := Parse(fileName, content)
	if parseError != nil {
		test.Fatal(parseError)
	}

	return records
}

func TestParseUserJS154(test *testing.T) {
	records := parsePinned(test, "user.js")

	activeCount := 0
	perSubsection := map[string]int{}

	for _, record := range records {
		if record.State == StateActive {
			activeCount++
			perSubsection[record.Section+" › "+record.Subsection]++
		}
	}

	if activeCount != 115 {
		test.Errorf("active prefs: got %d, want 115", activeCount)
	}

	expectations := map[string]int{
		"FASTFOX › NETWORKING":    7,
		"SECUREFOX › TELEMETRY":   17,
		"SECUREFOX › EXPERIMENTS": 4,
		"PESKYFOX › MOZILLA UI":   11,
		"PESKYFOX › AI":           6,
	}

	for subsection, want := range expectations {
		if perSubsection[subsection] != want {
			test.Errorf("%s: got %d, want %d", subsection, perSubsection[subsection], want)
		}
	}

	first := records[0]
	if first.Name != "gfx.content.skia-font-cache-size" || first.Value != prefs.Int(20) || first.Line != 21 || first.Subsection != "GENERAL" {
		test.Errorf("first record: %+v", first)
	}
}

func TestParsePeskyfoxGuide(test *testing.T) {
	records := parsePinned(test, "Peskyfox.js")

	byName := map[string][]Record{}
	malformed := 0

	for _, record := range records {
		byName[record.Name] = append(byName[record.Name], record)
		if record.State == StateMalformed {
			malformed++
		}
	}

	whatsNew := byName["browser.messaging-system.whatsNewPanel.enabled"]
	if len(whatsNew) != 1 || whatsNew[0].State != StateCommented || whatsNew[0].Section != "MOZILLA UI" || whatsNew[0].Value != prefs.Bool(false) {
		test.Errorf("whatsNewPanel: %+v", whatsNew)
	}

	zoom := byName["pdfjs.defaultZoomValue"]
	if len(zoom) == 0 || zoom[0].State != StateMalformed {
		test.Errorf("unquoted pdfjs.defaultZoomValue should be malformed: %+v", zoom)
	}

	if len(byName["browser.urlbar.suggest.engines"]) < 2 {
		test.Error("browser.urlbar.suggest.engines appears twice (alternatives) in Peskyfox 154.0")
	}

	if malformed == 0 {
		test.Error("want at least one malformed record")
	}
}

func TestDocumentsDefault(test *testing.T) {
	content := []byte("/** X ***/\n//user_pref(\"a.b\", true); // DEFAULT\n//user_pref(\"c.d\", false); // optional\n")

	records, parseError := Parse("x.js", content)
	if parseError != nil || len(records) != 2 {
		test.Fatalf("got %+v, %v", records, parseError)
	}

	if !records[0].DocumentsDefault || records[1].DocumentsDefault {
		test.Errorf("DocumentsDefault: %+v", records)
	}
}
