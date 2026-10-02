package logger

import (
	"bytes"
	"testing"
)

func TestLoggerDropsLinesBelowLevel(test *testing.T) {
	var output bytes.Buffer

	warnLogger := New(LevelWarn, &output)
	warnLogger.Debugf("hidden %d", 1)
	warnLogger.Infof("hidden %d", 2)
	warnLogger.Warnf("shown %d", 3)
	warnLogger.Errorf("shown %d", 4)

	want := "WARN: shown 3\nERROR: shown 4\n"
	if output.String() != want {
		test.Errorf("got %q, want %q", output.String(), want)
	}
}

func TestParseLevel(test *testing.T) {
	parsed, parseError := ParseLevel(" Debug ")
	if parseError != nil || parsed != LevelDebug {
		test.Errorf("got %v, %v; want debug, nil", parsed, parseError)
	}

	if _, unknownError := ParseLevel("chatty"); unknownError == nil {
		test.Error("want an error for an unknown level")
	}
}
