package firefox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// channelPrefPattern extracts app.update.channel from defaults/pref/channel-prefs.js.
var channelPrefPattern = regexp.MustCompile(`pref\(\s*"app\.update\.channel"\s*,\s*"([^"]+)"\s*\)`)

// Install is one Firefox installation on disk.
type Install struct {
	Dir              string // symlink-resolved directory holding the executable; this is what is hashed
	Hash             string // InstallHash(Dir)
	Name             string // display name, e.g. "Firefox Developer Edition"
	Version          string // e.g. 158.0
	MajorVersion     int
	BuildID          string
	Channel          string // release, beta, aurora, nightly, esr, or unknown
	SourceRepository string
	Packaged         bool // Mozilla deb/rpm package (is-packaged-app present)
}

// ReadInstall reads the install in installDir; found is false when it holds no Firefox.
func ReadInstall(installDir string) (Install, bool, error) {
	resolvedDir, resolveError := filepath.EvalSymlinks(installDir)
	if errors.Is(resolveError, fs.ErrNotExist) {
		return Install{}, false, nil
	}

	if resolveError != nil {
		return Install{}, false, fmt.Errorf("resolve %s: %w", installDir, resolveError)
	}

	applicationFile, readError := readINIFile(filepath.Join(resolvedDir, "application.ini"))
	if errors.Is(readError, fs.ErrNotExist) {
		return Install{}, false, nil
	}

	if readError != nil {
		return Install{}, false, readError
	}

	appSection := applicationFile.section("App")
	if appSection.value("Version") == "" {
		return Install{}, false, nil
	}

	channel, channelError := readChannel(resolvedDir, appSection.value("SourceRepository"))

	install := Install{
		Dir:              resolvedDir,
		Hash:             InstallHash(resolvedDir),
		Version:          appSection.value("Version"),
		MajorVersion:     majorVersion(appSection.value("Version")),
		BuildID:          appSection.value("BuildID"),
		Channel:          channel,
		SourceRepository: appSection.value("SourceRepository"),
		Packaged:         fileExists(filepath.Join(resolvedDir, "is-packaged-app")),
	}
	install.Name = displayName(appSection.value("Name"), appSection.value("CodeName"), channel)

	return install, true, channelError
}

// readChannel works out the update channel: channel-prefs.js, then update-settings.ini, then the source repository.
func readChannel(installDir, sourceRepository string) (string, error) {
	channel := ""

	var problems []error

	prefsContent, prefsError := os.ReadFile(filepath.Join(installDir, "defaults", "pref", "channel-prefs.js"))

	switch {
	case prefsError == nil:
		if match := channelPrefPattern.FindSubmatch(prefsContent); match != nil {
			channel = string(match[1])
		}
	case !errors.Is(prefsError, fs.ErrNotExist):
		problems = append(problems, fmt.Errorf("read channel-prefs.js: %w", prefsError))
	}

	if channel == "" || channel == "default" {
		updateChannel, updateError := channelFromUpdateSettings(installDir)
		if updateError != nil {
			problems = append(problems, updateError)
		}

		if updateChannel != "" {
			channel = updateChannel
		}
	}

	if channel == "" || channel == "default" {
		channel = channelFromRepository(sourceRepository)
	}

	return channel, errors.Join(problems...)
}

// channelFromUpdateSettings maps the first ACCEPTED_MAR_CHANNEL_IDS entry to a channel name.
func channelFromUpdateSettings(installDir string) (string, error) {
	settingsFile, readError := readINIFile(filepath.Join(installDir, "update-settings.ini"))
	if errors.Is(readError, fs.ErrNotExist) {
		return "", nil
	}

	if readError != nil {
		return "", readError
	}

	accepted := settingsFile.section("Settings").value("ACCEPTED_MAR_CHANNEL_IDS")
	firstID, _, _ := strings.Cut(accepted, ",")

	marChannels := map[string]string{
		"firefox-mozilla-release": "release",
		"firefox-mozilla-beta":    "beta",
		"firefox-mozilla-aurora":  "aurora",
		"firefox-mozilla-central": "nightly",
		"firefox-mozilla-esr":     "esr",
	}

	return marChannels[strings.TrimSpace(firstID)], nil
}

// channelFromRepository infers the channel from application.ini SourceRepository.
func channelFromRepository(sourceRepository string) string {
	repositoryName := filepath.Base(sourceRepository)

	channel := "unknown"

	switch {
	case repositoryName == "mozilla-release":
		channel = "release"
	case repositoryName == "mozilla-beta":
		channel = "beta"
	case repositoryName == "mozilla-central":
		channel = "nightly"
	case strings.HasPrefix(repositoryName, "mozilla-esr"):
		channel = "esr"
	}

	return channel
}

// displayName returns a human name such as "Firefox Developer Edition" or "Firefox ESR".
func displayName(appName, codeName, channel string) string {
	name := appName
	if name == "" {
		name = "Firefox"
	}

	channelNames := map[string]string{
		"aurora":  name + " Developer Edition",
		"beta":    name + " Beta",
		"nightly": name + " Nightly",
		"esr":     name + " ESR",
	}

	switch {
	case codeName != "":
		name = codeName
	case channelNames[channel] != "":
		name = channelNames[channel]
	}

	return name
}

// fileExists reports whether path exists; any error other than not-exist counts as existing.
func fileExists(path string) bool {
	_, statError := os.Stat(path)

	return !errors.Is(statError, fs.ErrNotExist)
}

// DiscoverInstalls finds Firefox installs from glob patterns, launchers in executable dirs, and hint dirs.
func DiscoverInstalls(searchPatterns, executableDirs, hintDirs []string) ([]Install, []string) {
	var warnings []string

	candidates := map[string]bool{}

	for _, pattern := range searchPatterns {
		matches, globError := filepath.Glob(pattern)
		if globError != nil {
			warnings = append(warnings, fmt.Sprintf("bad install search pattern %q: %v", pattern, globError))
		}

		for _, match := range matches {
			candidates[match] = true
		}
	}

	launcherDirs, launcherWarnings := launcherInstallDirs(executableDirs)
	warnings = append(warnings, launcherWarnings...)

	for _, launcherDir := range launcherDirs {
		candidates[launcherDir] = true
	}

	for _, hintDir := range hintDirs {
		candidates[hintDir] = true
	}

	installsByDir := map[string]Install{}

	for candidate := range candidates {
		install, found, readError := ReadInstall(candidate)
		if readError != nil {
			warnings = append(warnings, fmt.Sprintf("install %s: %v", candidate, readError))
		}

		if found {
			installsByDir[install.Dir] = install
		}
	}

	installs := make([]Install, 0, len(installsByDir))
	for _, install := range installsByDir {
		installs = append(installs, install)
	}

	sort.Slice(installs, func(left, right int) bool { return installs[left].Dir < installs[right].Dir })
	sort.Strings(warnings)

	return installs, warnings
}

// launcherInstallDirs resolves firefox* launcher symlinks in executableDirs to the directories they point into.
func launcherInstallDirs(executableDirs []string) ([]string, []string) {
	var installDirs, warnings []string

	for _, executableDir := range executableDirs {
		launchers, globError := filepath.Glob(filepath.Join(executableDir, "firefox*"))
		if globError != nil {
			warnings = append(warnings, fmt.Sprintf("cannot search %s for launchers: %v", executableDir, globError))
		}

		for _, launcher := range launchers {
			target, resolveError := filepath.EvalSymlinks(launcher)

			switch {
			case resolveError != nil:
				warnings = append(warnings, fmt.Sprintf("launcher %s is broken: %v", launcher, resolveError))
			case target != launcher:
				installDirs = append(installDirs, filepath.Dir(target))
			default:
				// Not a symlink (e.g. a shell wrapper); its folder holds no install.
			}
		}
	}

	return installDirs, warnings
}
