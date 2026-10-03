package cli

import (
	"errors"
	"flag"
	"fmt"
	"runtime"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/answers"
	"github.com/headwalluk/foxtrainer/internal/apply"
	"github.com/headwalluk/foxtrainer/internal/buildinfo"
	"github.com/headwalluk/foxtrainer/internal/firefox"
)

// errSomeInstancesFailed is returned when apply could not complete for every instance.
var errSomeInstancesFailed = errors.New("not every instance was applied")

// runApply writes every configured instance's user.js; with --dry-run it only shows the changes.
func runApply(environment Environment, arguments []string) error {
	return applyInstances(environment, arguments, false)
}

// runDiff shows what apply would change, without writing anything.
func runDiff(environment Environment, arguments []string) error {
	return applyInstances(environment, arguments, true)
}

// applyInstances is the shared body of apply and diff.
func applyInstances(environment Environment, arguments []string, alwaysDryRun bool) error {
	flags := flag.NewFlagSet("apply", flag.ContinueOnError)
	flags.SetOutput(environment.Stderr)
	dryRun := flags.Bool("dry-run", false, "show what would change without writing anything")
	offline := flags.Bool("offline", false, "use cached upstream files only")
	resetPrevious := flags.Bool("reset-previous", false, "also reset prefs set by a replaced user.js that foxtrainer did not write")

	if parseError := flags.Parse(arguments); parseError != nil {
		return parseError
	}

	showOnly := alwaysDryRun || *dryRun

	answersFile, loadError := answers.Load(environment.Config.Paths.AnswersFile)
	if loadError != nil {
		return loadError
	}

	if len(answersFile.Instances) == 0 {
		return errors.New("no instances are configured yet; run `foxtrainer configure` first")
	}

	inventory := discover(environment)

	loaded, catalogueError := loadCatalogue(environment, *offline, false)
	if catalogueError != nil {
		return catalogueError
	}

	options := applyOptions(environment, *resetPrevious)

	failures := 0

	for _, configured := range answersFile.Instances {
		if instanceError := applyOne(environment, loaded, inventory, configured, options, showOnly); instanceError != nil {
			environment.Logger.Errorf("%v", instanceError)

			failures++
		}
	}

	var applyError error
	if failures > 0 {
		applyError = fmt.Errorf("%w: %d failed", errSomeInstancesFailed, failures)
	}

	return applyError
}

// applyOne prepares one configured instance and either shows or commits the result.
func applyOne(environment Environment, loaded loadedCatalogue, inventory firefox.Inventory, configured answers.Instance, options apply.Options, showOnly bool) error {
	instance, found := matchConfigured(inventory, configured)
	if !found {
		return fmt.Errorf("configured instance %s with %s was not found; run `foxtrainer list`", configured.ProfilePath, configured.InstallPath)
	}

	label := instanceLabel(instance)

	if lock := instance.Profile.Lock; lock.InUse && !showOnly {
		return fmt.Errorf("%s is running (pid %d). Exit it first", label, lock.HolderPID)
	}

	prepared, prepareError := apply.Prepare(loaded.catalogue, loaded.upstream, targetFor(instance), configured.Answers, options)
	if prepareError != nil {
		return prepareError
	}

	if len(prepared.GroupShared) > 0 {
		environment.Logger.Warnf("%s is in a Profile Group: %d group-wide pref(s) may be overridden by the group's shared store (recorded in the manifest when applied)",
			label, len(prepared.GroupShared))
	}

	var report strings.Builder

	if showOnly {
		writeDiff(&report, prepared)
	} else {
		committed, commitError := apply.Commit(prepared, options)
		if commitError != nil {
			return commitError
		}

		writeApplied(&report, prepared, committed, environment.Config.Paths.HomeDir)
	}

	_, writeError := fmt.Fprint(environment.Stdout, report.String())

	return writeError
}

// matchConfigured finds the discovered instance for saved answers.
func matchConfigured(inventory firefox.Inventory, configured answers.Instance) (firefox.Instance, bool) {
	var found firefox.Instance

	matched := false

	for _, instance := range inventory.Instances() {
		if instance.Install.Dir == configured.InstallPath && instance.Profile.Profile.Dir == configured.ProfilePath {
			found = instance
			matched = true

			break
		}
	}

	return found, matched
}

// writeApplied summarises a committed apply.
func writeApplied(report *strings.Builder, prepared apply.Prepared, committed apply.Committed, homeDir string) {
	userJSState := "unchanged"
	if committed.WroteUserJS {
		userJSState = fmt.Sprintf("written (+%d added, %d changed, %d removed)", len(prepared.Added), len(prepared.Changed), len(prepared.Removed))
	}

	fmt.Fprintf(report, "Applied to %s\n", prepared.Target.Label)
	fmt.Fprintf(report, "  user.js:  %d prefs, %s\n", prepared.PrefCount(), userJSState)
	fmt.Fprintf(report, "  prefs.js: %d pref(s) reset to Firefox's default\n", len(committed.ResetInPrefs))
	if committed.BackupDir != "" {
		fmt.Fprintf(report, "  backup:   %s\n", shortenHome(committed.BackupDir, homeDir))
	}
	writeApplyNotes(report, prepared)
}

// writeDiff lists every change an apply would make.
func writeDiff(report *strings.Builder, prepared apply.Prepared) {
	fmt.Fprintf(report, "%s\n", prepared.Target.Label)

	if !prepared.UserJSChanged() && len(prepared.ResetNames) == 0 {
		report.WriteString("  up to date\n")
	}

	for _, change := range prepared.Added {
		fmt.Fprintf(report, "  + %s = %s\n", change.Name, change.NewValue)
	}

	for _, change := range prepared.Changed {
		fmt.Fprintf(report, "  ~ %s = %s (was %s)\n", change.Name, change.NewValue, change.OldValue)
	}

	for _, name := range prepared.Removed {
		fmt.Fprintf(report, "  - %s (removed from user.js)\n", name)
	}

	for _, name := range prepared.ResetNames {
		fmt.Fprintf(report, "  ↺ %s (reset to Firefox's default in prefs.js)\n", name)
	}

	if prepared.UserJSChanged() && len(prepared.Added)+len(prepared.Changed)+len(prepared.Removed) == 0 {
		report.WriteString("  user.js header changes only (foxtrainer or catalogue version)\n")
	}

	writeApplyNotes(report, prepared)
}

// writeApplyNotes adds an apply's notes and leftover warning.
func writeApplyNotes(report *strings.Builder, prepared apply.Prepared) {
	for _, note := range prepared.Notes {
		fmt.Fprintf(report, "  note: %s\n", note)
	}

	if len(prepared.Leftovers) > 0 && !prepared.ResetsLeftovers() {
		fmt.Fprintf(report, "  note: %d pref(s) set by the replaced user.js stay in prefs.js; add --reset-previous to reset them\n", len(prepared.Leftovers))
	}
}

// applyOptions builds the apply options from the resolved configuration.
func applyOptions(environment Environment, resetPrevious bool) apply.Options {
	return apply.Options{
		StateDir:            environment.Config.Paths.StateDir,
		ScopedExtensionDirs: environment.Config.Paths.ScopedExtensionDirs,
		FoxtrainerVersion:   buildinfo.Version(),
		Platform:            runtime.GOOS,
		ResetPrevious:       resetPrevious,
		Generators:          generators(environment),
	}
}

// targetFor describes a discovered instance as an apply target.
func targetFor(instance firefox.Instance) apply.Target {
	return apply.Target{
		InstallDir: instance.Install.Dir, ProfileDir: instance.Profile.Profile.Dir,
		Label: instanceLabel(instance), FirefoxMajor: instance.Install.MajorVersion,
		ProfileGroupID: instance.Profile.Profile.StoreID,
	}
}
