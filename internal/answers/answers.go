// Package answers loads and saves the user's per-instance answers in config.toml.
package answers

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/headwalluk/foxtrainer/internal/fsutil"
)

// SchemaVersion is the config.toml layout this build reads and writes.
const SchemaVersion = 1

// File is the whole of config.toml.
type File struct {
	SchemaVersion int        `toml:"schema_version"`
	Instances     []Instance `toml:"instance"`
}

// Instance is one Firefox install + profile and the answers chosen for it.
type Instance struct {
	InstallPath string  `toml:"install_path"`
	ProfilePath string  `toml:"profile_path"`
	Answers     Answers `toml:"answers"`
}

// Answers are the wizard's high-level choices for one instance.
type Answers struct {
	Feel      string   `toml:"feel"`    // lean | balanced | full
	AI        string   `toml:"ai"`      // off | local | all
	Privacy   string   `toml:"privacy"` // standard | strict | hardened
	HTTPSOnly bool     `toml:"https_only"`
	Languages []string `toml:"languages"` // BCP 47 tags, preferred first
}

// Load reads path, returning an empty File when it does not exist yet.
func Load(path string) (File, error) {
	loaded := File{SchemaVersion: SchemaVersion}

	content, readError := os.ReadFile(path)

	var loadError error

	switch {
	case errors.Is(readError, fs.ErrNotExist):
		// First run: no answers saved yet.
	case readError != nil:
		loadError = fmt.Errorf("read answers: %w", readError)
	default:
		loaded, loadError = decode(path, content)
	}

	return loaded, loadError
}

// decode parses config.toml content, rejecting unknown keys and unsupported schema versions.
func decode(path string, content []byte) (File, error) {
	var decoded File

	metadata, decodeError := toml.Decode(string(content), &decoded)
	if decodeError != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, decodeError)
	}

	var problems []error

	if undecoded := metadata.Undecoded(); len(undecoded) > 0 {
		keyNames := make([]string, 0, len(undecoded))
		for _, key := range undecoded {
			keyNames = append(keyNames, key.String())
		}

		problems = append(problems, fmt.Errorf("unknown keys: %s", strings.Join(keyNames, ", ")))
	}

	if decoded.SchemaVersion != SchemaVersion {
		problems = append(problems, fmt.Errorf("schema_version %d is not supported (want %d)", decoded.SchemaVersion, SchemaVersion))
	}

	var validationError error
	if len(problems) > 0 {
		validationError = fmt.Errorf("invalid %s: %w", path, errors.Join(problems...))
	}

	return decoded, validationError
}

// Save writes file to path atomically with owner-only permissions.
func Save(path string, file File) error {
	file.SchemaVersion = SchemaVersion

	var buffer bytes.Buffer
	if encodeError := toml.NewEncoder(&buffer).Encode(file); encodeError != nil {
		return fmt.Errorf("encode answers: %w", encodeError)
	}

	if mkdirError := os.MkdirAll(filepath.Dir(path), 0o700); mkdirError != nil {
		return fmt.Errorf("create answers folder: %w", mkdirError)
	}

	return fsutil.WriteFileAtomic(path, buffer.Bytes(), 0o600)
}

// Find returns the saved answers for an install + profile pair.
func (file *File) Find(installPath, profilePath string) (Instance, bool) {
	var found Instance

	matched := false

	for _, instance := range file.Instances {
		if instance.InstallPath == installPath && instance.ProfilePath == profilePath {
			found = instance
			matched = true

			break
		}
	}

	return found, matched
}

// Upsert stores answers for an instance, replacing any earlier answers for the same install + profile.
func (file *File) Upsert(updated Instance) {
	replaced := false

	for index := range file.Instances {
		if file.Instances[index].InstallPath == updated.InstallPath && file.Instances[index].ProfilePath == updated.ProfilePath {
			file.Instances[index] = updated
			replaced = true

			break
		}
	}

	if !replaced {
		file.Instances = append(file.Instances, updated)
	}
}
