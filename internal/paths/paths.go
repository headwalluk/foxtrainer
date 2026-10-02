// Package paths resolves foxtrainer's own folders and the Firefox profile roots for a platform.
package paths

import (
	"errors"
	"fmt"
	"path/filepath"
)

const (
	appDirName = "foxtrainer"

	// AnswersFileName is the file holding the user's saved answers, inside ConfigDir.
	AnswersFileName = "config.toml"
)

// Environment carries the raw environment values that path resolution depends on.
type Environment struct {
	OperatingSystem string // runtime.GOOS value
	HomeDir         string // HOME, or USERPROFILE on Windows
	XDGConfigHome   string
	XDGCacheHome    string
	XDGStateHome    string
	AppData         string // Windows APPDATA
	LocalAppData    string // Windows LOCALAPPDATA
	SearchPath      string // PATH, used to find firefox* launchers
}

// Paths holds every filesystem location foxtrainer uses.
type Paths struct {
	HomeDir      string   // the user's home, used to shorten paths for display
	ConfigDir    string   // saved answers
	AnswersFile  string   // ConfigDir/config.toml
	CacheDir     string   // downloaded upstream files and indexes
	StateDir     string   // per-instance manifests and backups
	FirefoxRoots []string // candidate folders holding profiles.ini, most likely first

	InstallSearchPatterns []string // globs for Firefox install folders
	ExecutableDirs        []string // PATH entries, searched for firefox* launchers
}

// Resolve builds Paths for environment, returning warnings and every problem found.
func Resolve(environment Environment) (Paths, []string, error) {
	resolver := &resolver{environment: environment}

	var resolved Paths

	switch environment.OperatingSystem {
	case "darwin":
		resolved = resolver.darwin()
	case "windows":
		resolved = resolver.windows()
	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly":
		resolved = resolver.xdg()
	default:
		resolver.fail("unsupported operating system %q", environment.OperatingSystem)
	}

	resolved.HomeDir = environment.HomeDir
	resolved.AnswersFile = filepath.Join(resolved.ConfigDir, AnswersFileName)
	resolved.ExecutableDirs = searchPathDirs(environment.SearchPath)

	return resolved, resolver.warnings, errors.Join(resolver.problems...)
}

// resolver accumulates warnings and problems while resolving paths.
type resolver struct {
	environment Environment
	warnings    []string
	problems    []error
}

// fail records a problem without stopping resolution.
func (resolver *resolver) fail(format string, arguments ...any) {
	resolver.problems = append(resolver.problems, fmt.Errorf(format, arguments...))
}

// home returns the validated home directory, recording a problem if it is unusable.
func (resolver *resolver) home(variableName string) string {
	homeDir := resolver.environment.HomeDir

	switch {
	case homeDir == "":
		resolver.fail("%s is not set", variableName)
	case !filepath.IsAbs(homeDir):
		resolver.fail("%s must be an absolute path, got %q", variableName, homeDir)
	}

	return homeDir
}

// xdgBase returns value when it is an absolute path, otherwise fallback.
func (resolver *resolver) xdgBase(variableName, value, fallback string) string {
	base := fallback

	switch {
	case value == "":
		// Unset: the XDG spec default applies.
	case filepath.IsAbs(value):
		base = value
	default:
		// The XDG Base Directory spec says relative values must be ignored.
		resolver.warnings = append(resolver.warnings,
			fmt.Sprintf("ignoring %s=%q: XDG paths must be absolute; using %s", variableName, value, fallback))
	}

	return base
}

// xdg resolves paths for Linux and other XDG desktops.
func (resolver *resolver) xdg() Paths {
	homeDir := resolver.home("HOME")
	configHome := resolver.xdgBase("XDG_CONFIG_HOME", resolver.environment.XDGConfigHome, filepath.Join(homeDir, ".config"))
	cacheHome := resolver.xdgBase("XDG_CACHE_HOME", resolver.environment.XDGCacheHome, filepath.Join(homeDir, ".cache"))
	stateHome := resolver.xdgBase("XDG_STATE_HOME", resolver.environment.XDGStateHome, filepath.Join(homeDir, ".local", "state"))

	return Paths{
		ConfigDir: filepath.Join(configHome, appDirName),
		CacheDir:  filepath.Join(cacheHome, appDirName),
		StateDir:  filepath.Join(stateHome, appDirName),
		// Firefox 147+ uses the XDG root only when ~/.mozilla does not exist; see docs/how-it-works.md.
		FirefoxRoots: []string{
			filepath.Join(homeDir, ".mozilla", "firefox"),
			filepath.Join(configHome, "mozilla", "firefox"),
		},
		// Mozilla/distro packages, then tarballs in common places; see docs/how-it-works.md.
		InstallSearchPatterns: []string{
			"/usr/lib/firefox*",
			"/usr/lib64/firefox*",
			"/opt/firefox*",
			filepath.Join(homeDir, "firefox*"),
			filepath.Join(homeDir, ".local", "opt", "firefox*"),
		},
	}
}

// darwin resolves paths for macOS.
func (resolver *resolver) darwin() Paths {
	homeDir := resolver.home("HOME")
	applicationSupport := filepath.Join(homeDir, "Library", "Application Support")

	return Paths{
		ConfigDir:    filepath.Join(applicationSupport, appDirName),
		CacheDir:     filepath.Join(homeDir, "Library", "Caches", appDirName),
		StateDir:     filepath.Join(applicationSupport, appDirName, "state"),
		FirefoxRoots: []string{filepath.Join(applicationSupport, "Firefox")},
	}
}

// windows resolves paths for Windows.
func (resolver *resolver) windows() Paths {
	appData := resolver.environment.AppData
	localAppData := resolver.environment.LocalAppData

	if appData == "" {
		resolver.fail("APPDATA is not set")
	}

	if localAppData == "" {
		resolver.fail("LOCALAPPDATA is not set")
	}

	return Paths{
		ConfigDir:    filepath.Join(appData, appDirName),
		CacheDir:     filepath.Join(localAppData, appDirName, "cache"),
		StateDir:     filepath.Join(localAppData, appDirName, "state"),
		FirefoxRoots: []string{filepath.Join(appData, "Mozilla", "Firefox")},
	}
}

// searchPathDirs splits PATH into absolute directories, dropping empty and relative entries.
func searchPathDirs(searchPath string) []string {
	var dirs []string

	for _, entry := range filepath.SplitList(searchPath) {
		if filepath.IsAbs(entry) {
			dirs = append(dirs, entry)
		}
	}

	return dirs
}
