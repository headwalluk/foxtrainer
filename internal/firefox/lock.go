package firefox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// LockState describes whether Firefox is using a profile.
type LockState struct {
	InUse        bool
	HolderPID    int    // 0 when unknown, e.g. the holder is in another PID namespace (Flatpak)
	StaleSymlink string // target of a "lock" symlink when no process holds the profile; Firefox leaves it after clean exits too
}

// ProbeLock reports whether a Firefox process holds profileDir, without taking or creating the lock.
func ProbeLock(profileDir string) (LockState, error) {
	state, probeError := probeProfileLock(profileDir)

	if !state.InUse && probeError == nil {
		symlinkTarget, readError := os.Readlink(filepath.Join(profileDir, "lock"))

		switch {
		case readError == nil:
			state.StaleSymlink = symlinkTarget
		case errors.Is(readError, fs.ErrNotExist):
			// No symlink: nothing to report.
		default:
			probeError = fmt.Errorf("read lock symlink in %s: %w", profileDir, readError)
		}
	}

	return state, probeError
}
