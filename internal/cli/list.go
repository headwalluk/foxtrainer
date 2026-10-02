package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/headwalluk/foxtrainer/internal/firefox"
)

// runList discovers Firefox installs and profiles and prints them as instances.
func runList(environment Environment, _ []string) error {
	resolved := environment.Config.Paths

	inventory := firefox.Discover(firefox.DiscoverOptions{
		ProfileRoots:          resolved.FirefoxRoots,
		InstallSearchPatterns: resolved.InstallSearchPatterns,
		ExecutableDirs:        resolved.ExecutableDirs,
	})

	for _, warning := range inventory.Warnings {
		environment.Logger.Warnf("%s", warning)
	}

	for _, root := range inventory.Roots {
		for _, status := range root.Profiles {
			if status.Lock.StaleSymlink != "" {
				// Firefox leaves the lock symlink behind even after a clean exit, so this is normal.
				environment.Logger.Debugf("profile %s: lock symlink -> %s, but no process holds .parentlock", status.Profile.Name, status.Lock.StaleSymlink)
			}
		}
	}

	_, writeError := io.WriteString(environment.Stdout, renderInventory(inventory, resolved.HomeDir))

	return writeError
}

// renderInventory formats an inventory for the terminal; homeDir is shown as "~".
func renderInventory(inventory firefox.Inventory, homeDir string) string {
	shorten := func(path string) string { return shortenHome(path, homeDir) }

	var builder strings.Builder

	instances := inventory.Instances()
	inInstance := map[string]bool{}

	builder.WriteString("Instances (Firefox install + profile)\n")

	if len(instances) == 0 {
		builder.WriteString("  none found\n")
	}

	for _, instance := range instances {
		inInstance[instance.Profile.Profile.Dir] = true

		fmt.Fprintf(&builder, "  ● %s %s › %s\n", instance.Install.Name, instance.Install.Version, instance.Profile.Profile.Name)
		fmt.Fprintf(&builder, "      install: %s\n", shorten(instance.Install.Dir))
		fmt.Fprintf(&builder, "      profile: %s\n", shorten(instance.Profile.Profile.Dir))
		fmt.Fprintf(&builder, "      status:  %s\n", instanceStatus(instance))
	}

	writeOrphans(&builder, inventory, shorten)
	writeOtherProfiles(&builder, inventory, inInstance, shorten)
	writeInstalls(&builder, inventory, shorten)
	writeNotes(&builder, inventory, shorten)

	return builder.String()
}

// instanceStatus summarises whether the profile is the install's default and whether Firefox is using it.
func instanceStatus(instance firefox.Instance) string {
	var parts []string

	if instance.IsDefault {
		parts = append(parts, "default profile")
	} else {
		parts = append(parts, "not the default profile (start with -P "+instance.Profile.Profile.Name+")")
	}

	parts = append(parts, lockStatus(instance.Profile.Lock))

	return strings.Join(parts, " · ")
}

// lockStatus describes a profile's lock in plain words.
func lockStatus(lock firefox.LockState) string {
	status := "not running"

	switch {
	case lock.InUse && lock.HolderPID > 0:
		status = fmt.Sprintf("RUNNING (pid %d): close Firefox before applying", lock.HolderPID)
	case lock.InUse:
		status = "RUNNING: close Firefox before applying"
	}

	return status
}

// writeOrphans lists profiles whose Firefox install no longer exists.
func writeOrphans(builder *strings.Builder, inventory firefox.Inventory, shorten func(string) string) {
	var lines []string

	for _, root := range inventory.Roots {
		for _, status := range root.Profiles {
			if status.Orphan {
				lines = append(lines, fmt.Sprintf("  ○ %s  %s\n      last used by %s (Firefox %s), which no longer exists\n",
					status.Profile.Name, shorten(status.Profile.Dir),
					shorten(status.Compatibility.LastPlatformDir), status.Compatibility.AppVersion))
			}
		}
	}

	if len(lines) > 0 {
		builder.WriteString("\nOrphaned profiles (their Firefox install no longer exists)\n")
		builder.WriteString(strings.Join(lines, ""))
	}
}

// writeOtherProfiles lists profiles that are neither part of an instance nor orphaned.
func writeOtherProfiles(builder *strings.Builder, inventory firefox.Inventory, inInstance map[string]bool, shorten func(string) string) {
	var lines []string

	for _, root := range inventory.Roots {
		for _, status := range root.Profiles {
			if inInstance[status.Profile.Dir] || status.Orphan {
				continue
			}

			lines = append(lines, fmt.Sprintf("  ○ %s  %s  %s\n", status.Profile.Name, shorten(status.Profile.Dir), otherProfileReason(status)))
		}
	}

	if len(lines) > 0 {
		builder.WriteString("\nOther profiles\n")
		builder.WriteString(strings.Join(lines, ""))
	}
}

// otherProfileReason explains why a profile is not attached to any install.
func otherProfileReason(status firefox.ProfileStatus) string {
	reason := "last used by an unrecognised install at " + status.Compatibility.LastPlatformDir

	switch {
	case status.DirMissing:
		reason = "folder is missing"
	case !status.HasRun:
		reason = "never started"
	}

	return reason
}

// writeInstalls lists every Firefox install found.
func writeInstalls(builder *strings.Builder, inventory firefox.Inventory, shorten func(string) string) {
	builder.WriteString("\nFirefox installs\n")

	if len(inventory.Installs) == 0 {
		builder.WriteString("  none found\n")
	}

	for _, install := range inventory.Installs {
		origin := ""
		if install.Packaged {
			origin = ", Mozilla package"
		}

		fmt.Fprintf(builder, "  %s %s (%s%s)  %s  [%s]\n", install.Name, install.Version, install.Channel, origin, shorten(install.Dir), install.Hash)
	}
}

// writeNotes lists housekeeping findings: dead install sections and sections Firefox ignores.
func writeNotes(builder *strings.Builder, inventory firefox.Inventory, shorten func(string) string) {
	var notes []string

	for _, root := range inventory.Roots {
		for _, dead := range root.DeadInstallSections {
			location := "an install that no longer exists"
			if dead.LastKnownDir != "" {
				location = shorten(dead.LastKnownDir) + ", which no longer exists"
			}

			notes = append(notes, fmt.Sprintf("profiles.ini has [Install%s] for %s", dead.Section.Hash, location))
		}

		if len(root.File.StraySections) > 0 {
			notes = append(notes, fmt.Sprintf("profiles.ini has %d section(s) Firefox ignores: [%s]",
				len(root.File.StraySections), strings.Join(root.File.StraySections, "], [")))
		}
	}

	if len(notes) > 0 {
		builder.WriteString("\nNotes\n")

		for _, note := range notes {
			fmt.Fprintf(builder, "  ! %s\n", note)
		}
	}
}

// shortenHome replaces a leading homeDir with "~".
func shortenHome(path, homeDir string) string {
	shortened := path

	if homeDir != "" {
		relative, relativeError := filepath.Rel(homeDir, path)
		if relativeError == nil && !strings.HasPrefix(relative, "..") {
			shortened = filepath.Join("~", relative)
		}
	}

	return shortened
}
