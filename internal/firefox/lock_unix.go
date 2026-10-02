//go:build !windows

package firefox

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// probeProfileLock asks the kernel whether any process holds a POSIX write lock on .parentlock.
//
// The file is opened read-only, never created or truncated, and F_GETLK only queries; it takes no lock.
func probeProfileLock(profileDir string) (LockState, error) {
	lockPath := filepath.Join(profileDir, ".parentlock")

	lockFile, openError := os.Open(lockPath)
	if errors.Is(openError, fs.ErrNotExist) {
		return LockState{}, nil
	}

	if openError != nil {
		return LockState{}, fmt.Errorf("open %s: %w", lockPath, openError)
	}

	query := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: io.SeekStart,
		Start:  0,
		Len:    0, // whole file
	}

	queryError := syscall.FcntlFlock(lockFile.Fd(), syscall.F_GETLK, &query)
	closeError := lockFile.Close()

	var state LockState

	if queryError == nil {
		state.InUse = query.Type != syscall.F_UNLCK
		state.HolderPID = int(query.Pid)
	}

	probeError := errors.Join(queryError, closeError)
	if probeError != nil {
		probeError = fmt.Errorf("query lock on %s: %w", lockPath, probeError)
	}

	return state, probeError
}
