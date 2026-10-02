// Package prefs models Firefox pref values and parses and writes user_pref(...) statements.
package prefs

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Kind is the type of a pref value; Firefox prefs are only ever bool, 32-bit int or string.
type Kind int

// Pref value kinds.
const (
	KindBool Kind = iota + 1
	KindInt
	KindString
)

// Value is a typed Firefox pref value.
type Value struct {
	Kind   Kind
	Flag   bool
	Number int32
	Text   string
}

// Bool returns a bool Value.
func Bool(flag bool) Value { return Value{Kind: KindBool, Flag: flag} }

// Int returns an int Value.
func Int(number int32) Value { return Value{Kind: KindInt, Number: number} }

// String returns a string Value.
func String(text string) Value { return Value{Kind: KindString, Text: text} }

// errUnsupportedType reports a value Firefox prefs cannot hold.
var errUnsupportedType = errors.New("pref values must be bool, int or string")

// FromTOML converts a decoded TOML scalar into a Value, rejecting floats and out-of-range ints.
func FromTOML(raw any) (Value, error) {
	var converted Value

	var convertError error

	switch typed := raw.(type) {
	case bool:
		converted = Bool(typed)
	case int64:
		if typed < math.MinInt32 || typed > math.MaxInt32 {
			convertError = fmt.Errorf("int %d is outside Firefox's 32-bit range", typed)
		} else {
			converted = Int(int32(typed))
		}
	case string:
		converted = String(typed)
	default:
		convertError = fmt.Errorf("%w, got %T (%v)", errUnsupportedType, raw, raw)
	}

	return converted, convertError
}

// Literal returns the value as it is written in user.js: true, 42 or a double-quoted string.
func (value Value) Literal() string {
	literal := ""

	switch value.Kind {
	case KindBool:
		literal = strconv.FormatBool(value.Flag)
	case KindInt:
		literal = strconv.FormatInt(int64(value.Number), 10)
	case KindString:
		literal = quote(value.Text)
	}

	return literal
}

// String renders the value for humans, the same as Literal.
func (value Value) String() string {
	return value.Literal()
}

// quote double-quotes text using the escapes Firefox's pref parser understands.
func quote(text string) string {
	var builder strings.Builder

	builder.WriteByte('"')

	for _, character := range text {
		switch character {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		default:
			builder.WriteRune(character)
		}
	}

	builder.WriteByte('"')

	return builder.String()
}
