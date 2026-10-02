package firefox

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DiscoverOptions says where to look for profile roots and installs.
type DiscoverOptions struct {
	ProfileRoots          []string // folders that may hold profiles.ini
	InstallSearchPatterns []string // globs for install folders, e.g. /usr/lib/firefox*
	ExecutableDirs        []string // PATH entries searched for firefox* launchers
}

// Inventory is everything discovered about Firefox on this machine.
type Inventory struct {
	Installs []Install
	Roots    []Root
	Warnings []string
}

// Root is one profile root (a folder with profiles.ini) and what was found in it.
type Root struct {
	Dir                 string
	File                ProfilesFile
	Profiles            []ProfileStatus
	DeadInstallSections []DeadInstallSection
}

// DeadInstallSection is an [Install<hash>] whose install no longer exists.
type DeadInstallSection struct {
	Section      InstallSection
	LastKnownDir string // install folder recovered from a profile's compatibility.ini, if any
}

// ProfileStatus is a profile plus everything known about who uses it.
type ProfileStatus struct {
	Profile       Profile
	DirMissing    bool
	HasRun        bool // compatibility.ini exists
	Compatibility Compatibility
	LastInstall   *Install  // live install that last ran this profile
	DefaultFor    []Install // live installs whose default profile this is
	Lock          LockState
	Orphan        bool // last run by an install that no longer exists
}

// Instance pairs a live install with a profile it uses.
type Instance struct {
	Install   Install
	Profile   ProfileStatus
	RootDir   string
	IsDefault bool // the profile is this install's default ([Install<hash>] Default=)
}

// Discover reads every profile root and install it can find; problems are collected as warnings.
func Discover(options DiscoverOptions) Inventory {
	var inventory Inventory

	var hintDirs []string

	for _, rootDir := range options.ProfileRoots {
		root, found := readRoot(rootDir, &inventory.Warnings)
		if !found {
			continue
		}

		for _, status := range root.Profiles {
			if status.HasRun && !isSandboxPath(status.Compatibility.LastPlatformDir) {
				hintDirs = append(hintDirs, status.Compatibility.LastPlatformDir)
			}
		}

		inventory.Roots = append(inventory.Roots, root)
	}

	installs, installWarnings := DiscoverInstalls(options.InstallSearchPatterns, options.ExecutableDirs, hintDirs)
	inventory.Installs = installs
	inventory.Warnings = append(inventory.Warnings, installWarnings...)

	installsByHash := map[string]Install{}
	for _, install := range installs {
		installsByHash[install.Hash] = install
	}

	for index := range inventory.Roots {
		linkRoot(&inventory.Roots[index], installsByHash)
	}

	return inventory
}

// readRoot reads one profile root's profiles.ini and each profile's state.
func readRoot(rootDir string, warnings *[]string) (Root, bool) {
	root := Root{Dir: rootDir}

	profilesFile, found, readError := ReadProfilesFile(rootDir)
	if readError != nil {
		*warnings = append(*warnings, fmt.Sprintf("profile root %s: %v", rootDir, readError))
	}

	if !found {
		return root, false
	}

	root.File = profilesFile
	*warnings = append(*warnings, profilesFile.Warnings...)

	for _, profile := range profilesFile.Profiles {
		status := ProfileStatus{Profile: profile, DirMissing: !fileExists(profile.Dir)}

		if !status.DirMissing {
			compatibility, hasRun, compatibilityError := ReadCompatibility(profile.Dir)
			if compatibilityError != nil {
				*warnings = append(*warnings, fmt.Sprintf("profile %s: %v", profile.Name, compatibilityError))
			}

			status.Compatibility = compatibility
			status.HasRun = hasRun

			lockState, lockError := ProbeLock(profile.Dir)
			if lockError != nil {
				*warnings = append(*warnings, fmt.Sprintf("profile %s: %v", profile.Name, lockError))
			}

			status.Lock = lockState
		}

		root.Profiles = append(root.Profiles, status)
	}

	return root, true
}

// linkRoot connects a root's profiles and install sections to live installs and marks orphans.
func linkRoot(root *Root, installsByHash map[string]Install) {
	for index := range root.Profiles {
		status := &root.Profiles[index]

		if !status.HasRun || status.Compatibility.LastPlatformDir == "" {
			continue
		}

		lastInstall, isLive := installsByHash[InstallHash(status.Compatibility.LastPlatformDir)]
		if isLive {
			status.LastInstall = &lastInstall
		}

		status.Orphan = !isLive &&
			!isSandboxPath(status.Compatibility.LastPlatformDir) &&
			!fileExists(status.Compatibility.LastPlatformDir)
	}

	for _, section := range root.File.InstallSections {
		install, isLive := installsByHash[section.Hash]
		defaultDir := resolveDescriptor(root.Dir, section.DefaultDescriptor)

		if !isLive {
			root.DeadInstallSections = append(root.DeadInstallSections, DeadInstallSection{
				Section:      section,
				LastKnownDir: lastKnownInstallDir(root.Profiles, section.Hash),
			})

			continue
		}

		for index := range root.Profiles {
			if defaultDir != "" && root.Profiles[index].Profile.Dir == defaultDir {
				root.Profiles[index].DefaultFor = append(root.Profiles[index].DefaultFor, install)
			}
		}
	}
}

// resolveDescriptor turns an [Install…] Default= descriptor into an absolute profile folder.
func resolveDescriptor(rootDir, descriptor string) string {
	resolved := ""

	if descriptor != "" {
		resolved = resolveProfileDir(rootDir, descriptor, !filepath.IsAbs(descriptor))
	}

	return resolved
}

// lastKnownInstallDir finds the install folder a dead hash belonged to, via profiles' LastPlatformDir.
func lastKnownInstallDir(profiles []ProfileStatus, hash string) string {
	lastKnown := ""

	for _, status := range profiles {
		if status.HasRun && InstallHash(status.Compatibility.LastPlatformDir) == hash {
			lastKnown = status.Compatibility.LastPlatformDir

			break
		}
	}

	return lastKnown
}

// isSandboxPath reports whether dir is a Snap or Flatpak path, which never proves an install is gone.
func isSandboxPath(dir string) bool {
	return strings.HasPrefix(dir, "/snap/") || strings.HasPrefix(dir, "/app/")
}

// Instances returns every live install + profile pairing: each install's default profile and each profile's last install.
func (inventory Inventory) Instances() []Instance {
	var instances []Instance

	seen := map[string]bool{}

	add := func(install Install, status ProfileStatus, rootDir string) {
		key := install.Dir + "\x00" + status.Profile.Dir
		if seen[key] {
			return
		}

		seen[key] = true

		isDefault := false

		for _, defaultInstall := range status.DefaultFor {
			if defaultInstall.Dir == install.Dir {
				isDefault = true
			}
		}

		instances = append(instances, Instance{Install: install, Profile: status, RootDir: rootDir, IsDefault: isDefault})
	}

	for _, root := range inventory.Roots {
		for _, status := range root.Profiles {
			for _, install := range status.DefaultFor {
				add(install, status, root.Dir)
			}

			if status.LastInstall != nil {
				add(*status.LastInstall, status, root.Dir)
			}
		}
	}

	return instances
}
