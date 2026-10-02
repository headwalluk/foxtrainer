package prefs

import (
	"bytes"
	"strings"
)

// RemoveUserPrefs deletes user_pref lines whose name is in names, leaving every other byte unchanged.
//
// It returns the cleaned content and the names actually removed, in file order. Lines that are
// not parseable pref statements are always kept.
func RemoveUserPrefs(content []byte, names map[string]bool) ([]byte, []string) {
	var cleaned bytes.Buffer

	var removed []string

	remaining := content

	for len(remaining) > 0 {
		lineEnd := bytes.IndexByte(remaining, '\n')

		line := remaining
		if lineEnd >= 0 {
			line = remaining[:lineEnd+1]
		}

		remaining = remaining[len(line):]

		name, isPref := userPrefName(line)
		if isPref && names[name] {
			removed = append(removed, name)

			continue
		}

		cleaned.Write(line)
	}

	return cleaned.Bytes(), removed
}

// userPrefName returns the pref name if line is a parseable user_pref statement.
func userPrefName(line []byte) (string, bool) {
	trimmed := strings.TrimSpace(string(line))
	if !strings.HasPrefix(trimmed, "user_pref(") {
		return "", false
	}

	statement, parseError := ParseStatement(trimmed)

	return statement.Name, parseError == nil
}
