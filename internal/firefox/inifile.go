package firefox

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

// iniEntry is one key=value line.
type iniEntry struct {
	key   string
	value string
}

// iniSection is one [name] block with its entries in file order.
type iniSection struct {
	name    string
	entries []iniEntry
}

// iniFile is a parsed INI file that keeps section and entry order, as Firefox's nsINIParser does.
type iniFile struct {
	sections []*iniSection
	warnings []string // malformed lines, which Firefox also ignores
}

// readINIFile parses the INI file at path.
func readINIFile(path string) (*iniFile, error) {
	content, readError := os.ReadFile(path)
	if readError != nil {
		return nil, fmt.Errorf("read %s: %w", path, readError)
	}

	return parseINI(path, content)
}

// parseINI parses INI content; name labels warnings and errors.
func parseINI(name string, content []byte) (*iniFile, error) {
	parsed := &iniFile{}

	var current *iniSection

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		switch {
		case line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			current = &iniSection{name: strings.TrimSpace(line[1 : len(line)-1])}
			parsed.sections = append(parsed.sections, current)
		case current != nil && strings.Contains(line, "="):
			key, value, _ := strings.Cut(line, "=")
			current.entries = append(current.entries, iniEntry{key: strings.TrimSpace(key), value: strings.TrimSpace(value)})
		default:
			parsed.warnings = append(parsed.warnings, fmt.Sprintf("%s:%d: ignored malformed line %q", name, lineNumber, line))
		}
	}

	if scanError := scanner.Err(); scanError != nil {
		return nil, fmt.Errorf("parse %s: %w", name, scanError)
	}

	return parsed, nil
}

// section returns the first section called name, or nil.
func (file *iniFile) section(name string) *iniSection {
	var found *iniSection

	for _, candidate := range file.sections {
		if candidate.name == name {
			found = candidate

			break
		}
	}

	return found
}

// get returns the value of the first entry called key in the section, if present.
func (section *iniSection) get(key string) (string, bool) {
	value := ""
	found := false

	for _, entry := range section.entries {
		if entry.key == key {
			value = entry.value
			found = true

			break
		}
	}

	return value, found
}

// value returns the value of key, or "" when the section is nil or the key is absent.
func (section *iniSection) value(key string) string {
	text := ""
	if section != nil {
		text, _ = section.get(key)
	}

	return text
}
