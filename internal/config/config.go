// Package config is the only place foxtrainer reads the environment; it normalises it into a typed Config.
package config

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/logger"
	"github.com/headwalluk/foxtrainer/internal/paths"
)

// EnvLogLevel names the variable that sets the log level (error, warn, info, debug).
const EnvLogLevel = "FOXTRAINER_LOG_LEVEL"

// LookupFunc reports an environment variable's value and whether it is set, like os.LookupEnv.
type LookupFunc func(name string) (string, bool)

// Config is foxtrainer's runtime configuration, resolved once at start-up.
type Config struct {
	LogLevel  logger.Level
	Paths     paths.Paths
	Languages []string // preferred languages from the locale (LC_ALL, LC_MESSAGES, LANG), as BCP 47 tags
	Warnings  []string // non-fatal problems to log once a logger exists
}

// FromEnvironment loads Config from the process environment.
func FromEnvironment() (Config, error) {
	return Load(os.LookupEnv, runtime.GOOS)
}

// Load builds Config from lookup for operatingSystem, reporting every problem at once.
func Load(lookup LookupFunc, operatingSystem string) (Config, error) {
	var problems []error

	value := func(name string) string {
		text, _ := lookup(name)

		return text
	}

	logLevel := logger.LevelInfo
	if levelText, isSet := lookup(EnvLogLevel); isSet {
		parsed, parseError := logger.ParseLevel(levelText)
		if parseError != nil {
			problems = append(problems, fmt.Errorf("%s: %w", EnvLogLevel, parseError))
		}

		logLevel = parsed
	}

	homeVariable := "HOME"
	if operatingSystem == "windows" {
		homeVariable = "USERPROFILE"
	}

	resolvedPaths, warnings, pathsError := paths.Resolve(paths.Environment{
		OperatingSystem: operatingSystem,
		HomeDir:         value(homeVariable),
		XDGConfigHome:   value("XDG_CONFIG_HOME"),
		XDGCacheHome:    value("XDG_CACHE_HOME"),
		XDGStateHome:    value("XDG_STATE_HOME"),
		AppData:         value("APPDATA"),
		LocalAppData:    value("LOCALAPPDATA"),
		SearchPath:      value("PATH"),
	})
	if pathsError != nil {
		problems = append(problems, pathsError)
	}

	loaded := Config{
		LogLevel:  logLevel,
		Paths:     resolvedPaths,
		Languages: localeLanguages(value("LC_ALL"), value("LC_MESSAGES"), value("LANG")),
		Warnings:  warnings,
	}

	var loadError error
	if len(problems) > 0 {
		loadError = fmt.Errorf("invalid configuration:\n%w", errors.Join(problems...))
	}

	return loaded, loadError
}

// localeLanguages turns the first set POSIX locale (e.g. en_GB.UTF-8) into BCP 47 tags: ["en-GB", "en"].
func localeLanguages(candidates ...string) []string {
	var languages []string

	for _, candidate := range candidates {
		localeName, _, _ := strings.Cut(candidate, ".")
		localeName, _, _ = strings.Cut(localeName, "@")

		if localeName == "" || localeName == "C" || localeName == "POSIX" {
			continue
		}

		tag := strings.ReplaceAll(localeName, "_", "-")
		languages = append(languages, tag)

		if primary, _, hasRegion := strings.Cut(tag, "-"); hasRegion {
			languages = append(languages, primary)
		}

		break
	}

	return languages
}
