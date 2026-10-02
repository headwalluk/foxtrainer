package firefox

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	profilesFileName      = "profiles.ini"
	compatibilityFileName = "compatibility.ini"
	installSectionPrefix  = "Install"
)

// hexSectionPattern matches a bare install hash used as a section name without the Install prefix.
var hexSectionPattern = regexp.MustCompile(`^[0-9A-F]{1,16}$`)

// Profile is one [ProfileN] entry from profiles.ini.
type Profile struct {
	Section         string // e.g. "Profile3"
	Name            string
	Descriptor      string // raw Path= value; [Install…] Default= refers to profiles by this
	Dir             string // absolute profile directory
	IsLegacyDefault bool   // Default=1, used only by non-dedicated builds such as Snap
	StoreID         string // set when the profile represents a Profile Group
}

// InstallSection is one [Install<hash>] entry from profiles.ini.
type InstallSection struct {
	Hash              string
	DefaultDescriptor string // Default= value; empty when the install has no default profile
	Locked            bool
}

// ProfilesFile is the parsed content of one profile root's profiles.ini.
type ProfilesFile struct {
	RootDir         string
	Version         string // [General] Version; "2" since Firefox 67
	Profiles        []Profile
	InstallSections []InstallSection
	StraySections   []string // sections Firefox ignores, e.g. bare hashes copied from installs.ini
	Warnings        []string
}

// ReadProfilesFile reads profiles.ini from rootDir; found is false when the root has none.
func ReadProfilesFile(rootDir string) (ProfilesFile, bool, error) {
	parsedFile := ProfilesFile{RootDir: rootDir}

	iniPath := filepath.Join(rootDir, profilesFileName)

	rawFile, readError := readINIFile(iniPath)
	if errors.Is(readError, fs.ErrNotExist) {
		return parsedFile, false, nil
	}

	if readError != nil {
		return parsedFile, false, readError
	}

	parsedFile.Warnings = append(parsedFile.Warnings, rawFile.warnings...)
	parsedFile.Version = rawFile.section("General").value("Version")
	parsedFile.Profiles = readProfileSections(rootDir, rawFile)

	contiguousCount := len(parsedFile.Profiles)

	for _, section := range rawFile.sections {
		switch {
		case section.name == "General":
			continue
		case strings.HasPrefix(section.name, installSectionPrefix):
			parsedFile.InstallSections = append(parsedFile.InstallSections, InstallSection{
				Hash:              strings.TrimPrefix(section.name, installSectionPrefix),
				DefaultDescriptor: section.value("Default"),
				Locked:            section.value("Locked") == "1",
			})
		case strings.HasPrefix(section.name, "Profile"):
			index, convertError := strconv.Atoi(strings.TrimPrefix(section.name, "Profile"))
			if convertError != nil || index >= contiguousCount {
				parsedFile.Warnings = append(parsedFile.Warnings,
					fmt.Sprintf("%s: [%s] is ignored by Firefox (profile sections must be numbered from 0 without gaps)", iniPath, section.name))
			}
		default:
			parsedFile.StraySections = append(parsedFile.StraySections, section.name)
		}
	}

	return parsedFile, true, nil
}

// readProfileSections reads [Profile0], [Profile1]… stopping at the first gap, as Firefox does.
func readProfileSections(rootDir string, rawFile *iniFile) []Profile {
	var profiles []Profile

	for index := 0; ; index++ {
		sectionName := "Profile" + strconv.Itoa(index)

		section := rawFile.section(sectionName)
		if section == nil {
			break
		}

		descriptor := section.value("Path")

		profiles = append(profiles, Profile{
			Section:         sectionName,
			Name:            section.value("Name"),
			Descriptor:      descriptor,
			Dir:             resolveProfileDir(rootDir, descriptor, section.value("IsRelative") == "1"),
			IsLegacyDefault: section.value("Default") == "1",
			StoreID:         section.value("StoreID"),
		})
	}

	return profiles
}

// resolveProfileDir turns a Path= value into an absolute directory; relative paths always use '/'.
func resolveProfileDir(rootDir, descriptor string, isRelative bool) string {
	resolved := filepath.Clean(descriptor)
	if isRelative {
		resolved = filepath.Join(rootDir, filepath.FromSlash(descriptor))
	}

	return resolved
}

// IsBareHashSection reports whether a stray section name looks like an install hash missing its prefix.
func IsBareHashSection(sectionName string) bool {
	return hexSectionPattern.MatchString(sectionName)
}

// Compatibility is a profile's compatibility.ini: the Firefox that last ran it.
type Compatibility struct {
	LastVersion     string // e.g. 158.0_20261002090342/20261002090342
	AppVersion      string // e.g. 158.0
	MajorVersion    int    // 0 when unknown
	LastOSABI       string
	LastPlatformDir string // install (GRE) dir; on Linux and Windows the directory that is hashed
	LastAppDir      string
}

// ReadCompatibility reads compatibility.ini from profileDir; found is false when the profile has never run.
func ReadCompatibility(profileDir string) (Compatibility, bool, error) {
	var compatibility Compatibility

	rawFile, readError := readINIFile(filepath.Join(profileDir, compatibilityFileName))
	if errors.Is(readError, fs.ErrNotExist) {
		return compatibility, false, nil
	}

	if readError != nil {
		return compatibility, false, readError
	}

	section := rawFile.section("Compatibility")
	compatibility.LastVersion = section.value("LastVersion")
	compatibility.LastOSABI = section.value("LastOSABI")
	compatibility.LastPlatformDir = section.value("LastPlatformDir")
	compatibility.LastAppDir = section.value("LastAppDir")
	compatibility.AppVersion, _, _ = strings.Cut(compatibility.LastVersion, "_")
	compatibility.MajorVersion = majorVersion(compatibility.AppVersion)

	return compatibility, true, nil
}

// majorVersion returns the number before the first '.', or 0 when there is none.
func majorVersion(version string) int {
	majorText, _, _ := strings.Cut(version, ".")

	major, convertError := strconv.Atoi(majorText)
	if convertError != nil {
		major = 0
	}

	return major
}
