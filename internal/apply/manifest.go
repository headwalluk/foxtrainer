// Package apply turns an instance's answers into a user.js, and writes it safely with backups and prefs.js cleanup.
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
	"sort"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/headwalluk/foxtrainer/internal/fsutil"
)

// manifestSchemaVersion is the manifest layout this build reads and writes.
const manifestSchemaVersion = 1

// Manifest records what foxtrainer last wrote to one instance, so removed prefs can be reset later.
type Manifest struct {
	SchemaVersion    int       `toml:"schema_version"`
	InstallPath      string    `toml:"install_path"`
	ProfilePath      string    `toml:"profile_path"`
	CatalogueVersion string    `toml:"catalogue_version"`
	AppliedAt        time.Time `toml:"applied_at"`
	UserJSSHA256     string    `toml:"user_js_sha256"`
	Managed          []string  `toml:"managed"` // every pref name foxtrainer currently owns in this profile
}

// InstanceKey returns a stable, readable file-name key for a profile folder.
func InstanceKey(profileDir string) string {
	digest := sha256.Sum256([]byte(profileDir))

	return filepath.Base(profileDir) + "-" + hex.EncodeToString(digest[:4])
}

// manifestPath returns where an instance's manifest lives.
func manifestPath(stateDir, profileDir string) string {
	return filepath.Join(stateDir, "instances", InstanceKey(profileDir)+".toml")
}

// LoadManifest reads an instance's manifest; a missing manifest means nothing has been applied yet.
func LoadManifest(stateDir, profileDir string) (Manifest, error) {
	loaded := Manifest{SchemaVersion: manifestSchemaVersion}
	path := manifestPath(stateDir, profileDir)

	content, readError := os.ReadFile(path)

	var loadError error

	switch {
	case errors.Is(readError, fs.ErrNotExist):
		// Never applied to this instance.
	case readError != nil:
		loadError = fmt.Errorf("read manifest: %w", readError)
	default:
		if _, decodeError := toml.Decode(string(content), &loaded); decodeError != nil {
			loadError = fmt.Errorf("parse manifest %s: %w", path, decodeError)
		} else if loaded.SchemaVersion != manifestSchemaVersion {
			loadError = fmt.Errorf("manifest %s: schema_version %d is not supported", path, loaded.SchemaVersion)
		}
	}

	return loaded, loadError
}

// SaveManifest writes an instance's manifest atomically.
func SaveManifest(stateDir string, manifest Manifest) error {
	manifest.SchemaVersion = manifestSchemaVersion
	sort.Strings(manifest.Managed)

	var buffer bytes.Buffer
	if encodeError := toml.NewEncoder(&buffer).Encode(manifest); encodeError != nil {
		return fmt.Errorf("encode manifest: %w", encodeError)
	}

	path := manifestPath(stateDir, manifest.ProfilePath)
	if mkdirError := os.MkdirAll(filepath.Dir(path), 0o700); mkdirError != nil {
		return fmt.Errorf("create state folder: %w", mkdirError)
	}

	return fsutil.WriteFileAtomic(path, buffer.Bytes(), 0o600)
}
