// Package logger writes levelled log lines to a writer, dropping anything below the configured level.
package logger

import (
	"fmt"
	"io"
	"strings"
)

// Level is a log severity; a higher value is more verbose.
type Level int

// Log levels, from least to most verbose.
const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

var levelNames = map[Level]string{
	LevelError: "error",
	LevelWarn:  "warn",
	LevelInfo:  "info",
	LevelDebug: "debug",
}

// String returns the lower-case name of the level.
func (level Level) String() string {
	name, found := levelNames[level]
	if !found {
		name = fmt.Sprintf("level(%d)", int(level))
	}

	return name
}

// ParseLevel converts a level name such as "warn" into a Level.
func ParseLevel(text string) (Level, error) {
	wanted := strings.ToLower(strings.TrimSpace(text))
	parsed := LevelInfo
	found := false

	for level, name := range levelNames {
		if name == wanted {
			parsed = level
			found = true

			break
		}
	}

	var parseError error
	if !found {
		parseError = fmt.Errorf("unknown log level %q (want error, warn, info or debug)", text)
	}

	return parsed, parseError
}

// Logger writes log lines at or above its level to an output.
type Logger struct {
	level  Level
	output io.Writer
}

// New returns a Logger that writes to output.
func New(level Level, output io.Writer) *Logger {
	return &Logger{level: level, output: output}
}

// Errorf logs at error level.
func (logger *Logger) Errorf(format string, arguments ...any) {
	logger.write(LevelError, format, arguments)
}

// Warnf logs at warn level.
func (logger *Logger) Warnf(format string, arguments ...any) {
	logger.write(LevelWarn, format, arguments)
}

// Infof logs at info level.
func (logger *Logger) Infof(format string, arguments ...any) {
	logger.write(LevelInfo, format, arguments)
}

// Debugf logs at debug level.
func (logger *Logger) Debugf(format string, arguments ...any) {
	logger.write(LevelDebug, format, arguments)
}

// write formats and emits one line when level is enabled.
func (logger *Logger) write(level Level, format string, arguments []any) {
	if level > logger.level {
		return
	}

	message := fmt.Sprintf(format, arguments...)
	// A failed write to the log destination has nowhere else to be reported.
	_, _ = fmt.Fprintf(logger.output, "%s: %s\n", strings.ToUpper(level.String()), message)
}
