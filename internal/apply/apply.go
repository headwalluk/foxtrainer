package apply

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/catalogue"
	"github.com/headwalluk/foxtrainer/internal/firefox"
	"github.com/headwalluk/foxtrainer/internal/fsutil"
	"github.com/headwalluk/foxtrainer/internal/prefs"
)

// enabledScopesPref blocks user- and system-scope extensions; see decision E6 in docs/how-it-works.md.
const enabledScopesPref = "extensions.enabledScopes"

// defaultKeepBackups is how many backups per instance are kept.
const defaultKeepBackups = 10

// Target is the instance being configured.
type Target struct {
	InstallDir   string
	ProfileDir   string
	Label        string // e.g. "Firefox Developer Edition 158.0 › dev-edition-default-1"
	FirefoxMajor int
}

// Options controls how plans are built and written.
type Options struct {
	StateDir            string
	ScopedExtensionDirs []string
	FoxtrainerVersion   string
	Platform            string
	Generators          map[string]catalogue.Generator
	ResetPrevious       bool // also reset prefs set by a replaced user.js that foxtrainer did not write
	KeepBackups         int
	Now                 func() time.Time
}

// Change is one pref whose user.js value differs from what is there now.
type Change struct {
	Name     string
	OldValue prefs.Value
	NewValue prefs.Value
}

// Prepared is everything an apply would do to one instance, computed without writing anything.
type Prepared struct {
	Target         Target
	Answers        answers.Answers
	Plan           catalogue.Plan
	UserJS         []byte
	ExistingUserJS []byte
	Added          []Change // in the new user.js but not the current one
	Changed        []Change // in both, with a different value
	Removed        []string // in the current user.js but not the new one
	ResetNames     []string // will be removed from prefs.js so Firefox's default returns
	Leftovers      []string // set by a replaced, non-foxtrainer user.js; reset only with ResetPrevious
	Notes          []string
	manifest       Manifest
}

// UserJSChanged reports whether user.js would be rewritten.
func (prepared Prepared) UserJSChanged() bool {
	return !bytes.Equal(prepared.UserJS, prepared.ExistingUserJS)
}

// ResetsLeftovers reports whether every leftover from a replaced user.js will be reset.
func (prepared Prepared) ResetsLeftovers() bool {
	resetsAll := true

	for _, name := range prepared.Leftovers {
		if !slices.Contains(prepared.ResetNames, name) {
			resetsAll = false

			break
		}
	}

	return resetsAll
}

// PrefCount returns the number of prefs in the new user.js.
func (prepared Prepared) PrefCount() int {
	total := 0
	for _, planned := range prepared.Plan.Groups {
		total += len(planned.Prefs)
	}

	return total
}

// Prepare resolves the answers for target and compares the result with the profile as it is now.
func Prepare(loaded catalogue.Catalogue, upstream catalogue.Upstream, target Target, chosen answers.Answers, options Options) (Prepared, error) {
	prepared := Prepared{Target: target, Answers: chosen}

	plan, resolveError := catalogue.Resolve(loaded, upstream, chosen,
		catalogue.Target{FirefoxMajor: target.FirefoxMajor, Platform: options.Platform}, options.Generators)
	if resolveError != nil {
		return Prepared{}, fmt.Errorf("resolve %s: %w", target.Label, resolveError)
	}

	plan, extensionNotes, scanError := keepScopedExtensionsWorking(plan, options.ScopedExtensionDirs)
	if scanError != nil {
		return Prepared{}, scanError
	}

	prepared.Plan = plan
	prepared.Notes = append(slices.Clone(plan.Notes), extensionNotes...)
	prepared.UserJS = RenderUserJS(loaded, plan, HeaderInfo{
		FoxtrainerVersion: options.FoxtrainerVersion, InstanceLabel: target.Label, Answers: chosen,
	})

	manifest, manifestError := LoadManifest(options.StateDir, target.ProfileDir)
	if manifestError != nil {
		return Prepared{}, manifestError
	}

	prepared.manifest = manifest

	existing, readError := readOptional(filepath.Join(target.ProfileDir, "user.js"))
	if readError != nil {
		return Prepared{}, readError
	}

	prepared.ExistingUserJS = existing
	prepared.compare(options)

	return prepared, nil
}

// compare fills the user.js differences and the prefs.js reset list.
func (prepared *Prepared) compare(options Options) {
	planned := map[string]prefs.Value{}

	for _, group := range prepared.Plan.Groups {
		for _, pref := range group.Prefs {
			planned[pref.Name] = pref.Value
		}
	}

	current, badLines := ParseUserJS(prepared.ExistingUserJS)
	if len(badLines) > 0 {
		prepared.Notes = append(prepared.Notes, fmt.Sprintf("the current user.js has %d unparseable pref line(s) (lines %v); it will be backed up and replaced", len(badLines), badLines))
	}

	for _, name := range sortedNames(planned) {
		oldValue, existed := current[name]

		switch {
		case !existed:
			prepared.Added = append(prepared.Added, Change{Name: name, NewValue: planned[name]})
		case oldValue != planned[name]:
			prepared.Changed = append(prepared.Changed, Change{Name: name, OldValue: oldValue, NewValue: planned[name]})
		}
	}

	for _, name := range sortedNames(current) {
		if _, stillPlanned := planned[name]; !stillPlanned {
			prepared.Removed = append(prepared.Removed, name)
		}
	}

	for _, name := range prepared.manifest.Managed {
		if _, stillPlanned := planned[name]; !stillPlanned {
			prepared.ResetNames = append(prepared.ResetNames, name)
		}
	}

	if len(prepared.ExistingUserJS) > 0 && !IsGenerated(prepared.ExistingUserJS) {
		prepared.Notes = append(prepared.Notes, "the current user.js was not written by foxtrainer; it will be backed up and replaced")

		for _, name := range prepared.Removed {
			if !slices.Contains(prepared.ResetNames, name) {
				prepared.Leftovers = append(prepared.Leftovers, name)
			}
		}

		if options.ResetPrevious {
			prepared.ResetNames = append(prepared.ResetNames, prepared.Leftovers...)
		}
	}

	sort.Strings(prepared.ResetNames)
}

// keepScopedExtensionsWorking drops extensions.enabledScopes when user- or system-scope extensions exist.
func keepScopedExtensionsWorking(plan catalogue.Plan, scopedDirs []string) (catalogue.Plan, []string, error) {
	var occupied []string

	var problems []error

	for _, scopedDir := range scopedDirs {
		entries, readError := os.ReadDir(scopedDir)

		switch {
		case errors.Is(readError, fs.ErrNotExist):
			// No extensions installed in this scope.
		case readError != nil:
			problems = append(problems, fmt.Errorf("scan %s: %w", scopedDir, readError))
		case len(entries) > 0:
			occupied = append(occupied, scopedDir)
		}
	}

	var notes []string

	if len(occupied) > 0 {
		for index := range plan.Groups {
			before := len(plan.Groups[index].Prefs)
			plan.Groups[index].Prefs = slices.DeleteFunc(plan.Groups[index].Prefs, func(pref catalogue.PlannedPref) bool {
				return pref.Name == enabledScopesPref
			})

			if len(plan.Groups[index].Prefs) < before {
				notes = append(notes, fmt.Sprintf("kept %s at Firefox's default because extensions are installed in %v", enabledScopesPref, occupied))
			}
		}
	}

	return plan, notes, errors.Join(problems...)
}

// Committed reports what Commit actually did.
type Committed struct {
	BackupDir    string
	WroteUserJS  bool
	ResetInPrefs []string // names actually removed from prefs.js
}

// Commit writes a prepared apply: lock the profile, back up, write user.js, reset prefs.js, save the manifest.
func Commit(prepared Prepared, options Options) (Committed, error) {
	lock, lockError := firefox.AcquireLock(prepared.Target.ProfileDir)
	if lockError != nil {
		return Committed{}, fmt.Errorf("%s: %w", prepared.Target.Label, lockError)
	}

	committed, commitError := commitLocked(prepared, options)

	// Release only after every write: closing any descriptor on .parentlock would drop the lock early.
	if releaseError := lock.Release(); releaseError != nil {
		commitError = errors.Join(commitError, fmt.Errorf("release profile lock: %w", releaseError))
	}

	return committed, commitError
}

// commitLocked does the writes; the caller holds the profile lock.
func commitLocked(prepared Prepared, options Options) (Committed, error) {
	var committed Committed

	now := time.Now
	if options.Now != nil {
		now = options.Now
	}

	userJSPath := filepath.Join(prepared.Target.ProfileDir, "user.js")
	prefsJSPath := filepath.Join(prepared.Target.ProfileDir, "prefs.js")

	if symlinkError := refuseSymlink(userJSPath); symlinkError != nil {
		return committed, symlinkError
	}

	if prepared.UserJSChanged() || len(prepared.ResetNames) > 0 {
		backupDir, backupError := backUp(options, prepared.Target.ProfileDir, now(), userJSPath, prefsJSPath)
		if backupError != nil {
			return committed, backupError
		}

		committed.BackupDir = backupDir
	}

	if prepared.UserJSChanged() {
		if writeError := fsutil.WriteFileAtomic(userJSPath, prepared.UserJS, 0o600); writeError != nil {
			return committed, writeError
		}

		committed.WroteUserJS = true
	}

	resetDone, resetError := resetPrefs(prefsJSPath, prepared.ResetNames)
	committed.ResetInPrefs = resetDone

	managed := plannedNames(prepared.Plan)
	if resetError != nil {
		// Keep the names we failed to reset, so the next apply tries again.
		managed = append(managed, prepared.ResetNames...)
	}

	digest := sha256.Sum256(prepared.UserJS)
	manifestError := SaveManifest(options.StateDir, Manifest{
		InstallPath:      prepared.Target.InstallDir,
		ProfilePath:      prepared.Target.ProfileDir,
		CatalogueVersion: prepared.Plan.CatalogueVersion,
		AppliedAt:        now().UTC(),
		UserJSSHA256:     hex.EncodeToString(digest[:]),
		Managed:          managed,
	})

	return committed, errors.Join(resetError, manifestError)
}

// resetPrefs removes names from prefs.js so Firefox falls back to its defaults.
func resetPrefs(prefsJSPath string, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	content, readError := readOptional(prefsJSPath)
	if readError != nil || len(content) == 0 {
		return nil, readError
	}

	nameSet := map[string]bool{}
	for _, name := range names {
		nameSet[name] = true
	}

	cleaned, removed := prefs.RemoveUserPrefs(content, nameSet)
	if len(removed) == 0 {
		return nil, nil
	}

	info, statError := os.Stat(prefsJSPath)
	if statError != nil {
		return nil, fmt.Errorf("stat prefs.js: %w", statError)
	}

	if writeError := fsutil.WriteFileAtomic(prefsJSPath, cleaned, info.Mode().Perm()); writeError != nil {
		return nil, fmt.Errorf("reset prefs in prefs.js: %w", writeError)
	}

	return removed, nil
}

// backUp copies the files that exist into a new timestamped backup folder and prunes old backups.
func backUp(options Options, profileDir string, moment time.Time, filePaths ...string) (string, error) {
	instanceBackups := filepath.Join(options.StateDir, "backups", InstanceKey(profileDir))
	backupDir := filepath.Join(instanceBackups, moment.UTC().Format("20060102T150405.000000000Z"))

	if mkdirError := os.MkdirAll(backupDir, 0o700); mkdirError != nil {
		return "", fmt.Errorf("create backup folder: %w", mkdirError)
	}

	for _, filePath := range filePaths {
		content, readError := readOptional(filePath)
		if readError != nil {
			return "", readError
		}

		if content == nil {
			continue
		}

		if writeError := os.WriteFile(filepath.Join(backupDir, filepath.Base(filePath)), content, 0o600); writeError != nil {
			return "", fmt.Errorf("back up %s: %w", filePath, writeError)
		}
	}

	keep := options.KeepBackups
	if keep <= 0 {
		keep = defaultKeepBackups
	}

	return backupDir, pruneBackups(instanceBackups, keep)
}

// pruneBackups deletes the oldest backup folders beyond keep; folder names sort by time.
func pruneBackups(instanceBackups string, keep int) error {
	entries, readError := os.ReadDir(instanceBackups)
	if readError != nil {
		return fmt.Errorf("list backups: %w", readError)
	}

	var problems []error

	for index := 0; index < len(entries)-keep; index++ {
		if removeError := os.RemoveAll(filepath.Join(instanceBackups, entries[index].Name())); removeError != nil {
			problems = append(problems, fmt.Errorf("prune backup: %w", removeError))
		}
	}

	return errors.Join(problems...)
}

// refuseSymlink fails if path is a symlink: an atomic replace would silently turn it into a plain file.
func refuseSymlink(path string) error {
	info, statError := os.Lstat(path)

	var symlinkError error

	switch {
	case errors.Is(statError, fs.ErrNotExist):
		// Nothing there yet.
	case statError != nil:
		symlinkError = fmt.Errorf("stat %s: %w", path, statError)
	case info.Mode()&fs.ModeSymlink != 0:
		symlinkError = fmt.Errorf("%s is a symlink; foxtrainer will not replace it", path)
	}

	return symlinkError
}

// readOptional reads path, returning nil content (not an error) when it does not exist.
func readOptional(path string) ([]byte, error) {
	content, readError := os.ReadFile(path)
	if errors.Is(readError, fs.ErrNotExist) {
		return nil, nil
	}

	if readError != nil {
		return nil, fmt.Errorf("read %s: %w", path, readError)
	}

	return content, nil
}

// plannedNames lists every pref name in the plan.
func plannedNames(plan catalogue.Plan) []string {
	var names []string

	for _, planned := range plan.Groups {
		for _, pref := range planned.Prefs {
			names = append(names, pref.Name)
		}
	}

	return names
}

// sortedNames returns a map's keys in order.
func sortedNames(values map[string]prefs.Value) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
