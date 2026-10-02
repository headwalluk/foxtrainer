// Package betterfox reads Betterfox's user.js and guide files into pref records with their sections.
//
// The format is described in docs/sources-and-credits.md: box-comment SECTION banners,
// "/** NAME ***/" subsections in user.js, and active or commented-out user_pref lines.
package betterfox

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/prefs"
)

// State says how a pref line appears in the file.
type State int

// Record states.
const (
	StateActive    State = iota + 1 // uncommented: Betterfox sets it
	StateCommented                  // commented out: optional, or documenting a default
	StateMalformed                  // looks like a pref line but does not parse
)

// Record is one user_pref line found in a Betterfox file.
type Record struct {
	File             string
	Line             int
	Section          string // banner, e.g. "SECUREFOX" in user.js or "URL BAR" in Peskyfox.js
	Subsection       string // "/** NAME ***/" heading; user.js only
	Name             string
	Value            prefs.Value
	State            State
	Trailing         string
	DocumentsDefault bool   // commented out and marked DEFAULT: Firefox already behaves this way
	Problem          string // parse error, for StateMalformed
}

var (
	sectionBannerPattern = regexp.MustCompile(`^\s*\*\s*(?:SECTION|OPTION):\s*(.+?)\s*\*?\s*$`)
	subsectionPattern    = regexp.MustCompile(`^/\*\*\s*(.+?)\s*\*{3}/\s*$`)
	defaultMarkerPattern = regexp.MustCompile(`\bDEFAULT\b`)
)

// Parse reads every pref line in a Betterfox file, tracking the section and subsection it sits under.
func Parse(fileName string, content []byte) ([]Record, error) {
	var records []Record

	section := ""
	subsection := ""
	lineNumber := 0

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if match := sectionBannerPattern.FindStringSubmatch(line); match != nil {
			section = match[1]
			subsection = ""

			continue
		}

		if match := subsectionPattern.FindStringSubmatch(trimmed); match != nil {
			subsection = match[1]

			continue
		}

		record, isPrefLine := parsePrefLine(trimmed)
		if !isPrefLine {
			continue
		}

		record.File = fileName
		record.Line = lineNumber
		record.Section = section
		record.Subsection = subsection
		records = append(records, record)
	}

	if scanError := scanner.Err(); scanError != nil {
		return nil, fmt.Errorf("read %s: %w", fileName, scanError)
	}

	return records, nil
}

// parsePrefLine recognises active ("user_pref(") and commented ("//user_pref(") pref lines.
func parsePrefLine(trimmed string) (Record, bool) {
	var record Record

	statementText := trimmed
	record.State = StateActive

	if strings.HasPrefix(trimmed, "//") {
		statementText = strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
		record.State = StateCommented
	}

	if !strings.HasPrefix(statementText, "user_pref(") {
		return Record{}, false
	}

	statement, parseError := prefs.ParseStatement(statementText)
	record.Name = statement.Name

	if parseError != nil {
		record.State = StateMalformed
		record.Problem = parseError.Error()

		return record, true
	}

	record.Value = statement.Value
	record.Trailing = statement.Trailing
	record.DocumentsDefault = record.State == StateCommented && defaultMarkerPattern.MatchString(statement.Trailing)

	return record, true
}
