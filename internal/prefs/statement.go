package prefs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Statement is one parsed pref call, e.g. user_pref("name", value); // trailing comment.
type Statement struct {
	Function string // user_pref, pref, sticky_pref or lockPref
	Name     string
	Value    Value
	Trailing string // text after ");", with any leading "//" removed
}

// errSyntax marks a statement that does not parse.
var errSyntax = errors.New("syntax error")

// statementFunctions lists the pref call names Firefox's parser accepts.
var statementFunctions = []string{"user_pref", "sticky_pref", "lockPref", "pref"}

// ParseStatement parses a single pref call starting at the function name, e.g. a line with "//" already removed.
func ParseStatement(text string) (Statement, error) {
	scanner := &tokenScanner{text: text}

	var statement Statement

	statement.Function = scanner.identifier()
	if !isStatementFunction(statement.Function) {
		return Statement{}, fmt.Errorf("%w: expected a pref call, got %q", errSyntax, firstWord(text))
	}

	steps := []func() error{
		func() error { return scanner.expect('(') },
		func() error {
			name, nameError := scanner.quotedString()
			statement.Name = name

			return nameError
		},
		func() error { return scanner.expect(',') },
		func() error {
			value, valueError := scanner.literal()
			statement.Value = value

			return valueError
		},
		func() error { return scanner.expect(')') },
		func() error { return scanner.expect(';') },
	}

	var parseError error

	for _, step := range steps {
		parseError = step()
		if parseError != nil {
			break
		}
	}

	if parseError != nil {
		return Statement{Function: statement.Function, Name: statement.Name}, parseError
	}

	trailing := strings.TrimSpace(scanner.rest())
	statement.Trailing = strings.TrimSpace(strings.TrimPrefix(trailing, "//"))

	return statement, nil
}

// FormatUserPref renders a user_pref line for user.js.
func FormatUserPref(name string, value Value) string {
	return "user_pref(" + quote(name) + ", " + value.Literal() + ");"
}

// isStatementFunction reports whether name is a pref call Firefox understands.
func isStatementFunction(name string) bool {
	found := false

	for _, candidate := range statementFunctions {
		if candidate == name {
			found = true

			break
		}
	}

	return found
}

// firstWord returns the leading run of non-space characters, for error messages.
func firstWord(text string) string {
	word, _, _ := strings.Cut(strings.TrimSpace(text), " ")

	return word
}

// tokenScanner walks a statement left to right.
type tokenScanner struct {
	text     string
	position int
}

// skipSpace advances past whitespace.
func (scanner *tokenScanner) skipSpace() {
	for scanner.position < len(scanner.text) && strings.ContainsRune(" \t", rune(scanner.text[scanner.position])) {
		scanner.position++
	}
}

// rest returns the unread remainder.
func (scanner *tokenScanner) rest() string {
	return scanner.text[scanner.position:]
}

// identifier reads [A-Za-z_]+ after optional whitespace.
func (scanner *tokenScanner) identifier() string {
	scanner.skipSpace()
	start := scanner.position

	for scanner.position < len(scanner.text) {
		character := scanner.text[scanner.position]
		if character != '_' && (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') {
			break
		}

		scanner.position++
	}

	return scanner.text[start:scanner.position]
}

// expect consumes the given punctuation after optional whitespace.
func (scanner *tokenScanner) expect(wanted byte) error {
	scanner.skipSpace()

	if scanner.position >= len(scanner.text) || scanner.text[scanner.position] != wanted {
		return fmt.Errorf("%w: expected %q at column %d", errSyntax, wanted, scanner.position+1)
	}

	scanner.position++

	return nil
}

// literal reads true, false, an integer or a quoted string.
func (scanner *tokenScanner) literal() (Value, error) {
	scanner.skipSpace()

	if scanner.position >= len(scanner.text) {
		return Value{}, fmt.Errorf("%w: missing value", errSyntax)
	}

	var parsed Value

	var literalError error

	switch character := scanner.text[scanner.position]; {
	case character == '"' || character == '\'':
		text, stringError := scanner.quotedString()
		parsed, literalError = String(text), stringError
	case character == '-' || character == '+' || (character >= '0' && character <= '9'):
		parsed, literalError = scanner.integer()
	default:
		word := scanner.identifier()

		switch word {
		case "true":
			parsed = Bool(true)
		case "false":
			parsed = Bool(false)
		default:
			literalError = fmt.Errorf("%w: unquoted value %q", errSyntax, word+firstWord(scanner.rest()))
		}
	}

	return parsed, literalError
}

// integer reads an optionally signed decimal int32.
func (scanner *tokenScanner) integer() (Value, error) {
	start := scanner.position

	if strings.ContainsRune("+-", rune(scanner.text[scanner.position])) {
		scanner.position++
	}

	for scanner.position < len(scanner.text) && scanner.text[scanner.position] >= '0' && scanner.text[scanner.position] <= '9' {
		scanner.position++
	}

	number, parseError := strconv.ParseInt(scanner.text[start:scanner.position], 10, 32)
	if parseError != nil {
		return Value{}, fmt.Errorf("%w: bad integer %q: %w", errSyntax, scanner.text[start:scanner.position], parseError)
	}

	return Int(int32(number)), nil
}

// quotedString reads a single- or double-quoted string with backslash escapes.
func (scanner *tokenScanner) quotedString() (string, error) {
	scanner.skipSpace()

	if scanner.position >= len(scanner.text) || !strings.ContainsRune(`"'`, rune(scanner.text[scanner.position])) {
		return "", fmt.Errorf("%w: expected a quoted string at column %d", errSyntax, scanner.position+1)
	}

	quoteCharacter := scanner.text[scanner.position]
	scanner.position++

	var builder strings.Builder

	closed := false

	for scanner.position < len(scanner.text) && !closed {
		character := scanner.text[scanner.position]

		switch {
		case character == quoteCharacter:
			closed = true
			scanner.position++
		case character == '\\' && scanner.position+1 < len(scanner.text):
			builder.WriteString(unescape(scanner.text[scanner.position+1]))
			scanner.position += 2
		default:
			decoded, width := utf8.DecodeRuneInString(scanner.text[scanner.position:])
			builder.WriteRune(decoded)
			scanner.position += width
		}
	}

	var stringError error
	if !closed {
		stringError = fmt.Errorf("%w: unterminated string", errSyntax)
	}

	return builder.String(), stringError
}

// unescape returns the character for a backslash escape; unknown escapes yield the character itself.
func unescape(escaped byte) string {
	replacements := map[byte]string{'n': "\n", 'r': "\r", 't': "\t"}

	replacement, known := replacements[escaped]
	if !known {
		replacement = string(escaped)
	}

	return replacement
}
