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

// ProfileLock is a held Firefox profile lock; Firefox refuses to start on the profile while it is held.
type ProfileLock struct {
	file *os.File
}

// AcquireLock takes the profile's .parentlock write lock without blocking, failing if Firefox holds it.
//
// The file is opened without O_TRUNC, as Firefox does. Closing any descriptor on it drops the
// process's POSIX locks, so it must be opened exactly once and released with Release.
func AcquireLock(profileDir string) (*ProfileLock, error) {
	lockPath := filepath.Join(profileDir, ".parentlock")

	lockFile, openError := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if openError != nil {
		return nil, fmt.Errorf("open %s: %w", lockPath, openError)
	}

	exclusive := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: io.SeekStart}

	lockError := syscall.FcntlFlock(lockFile.Fd(), syscall.F_SETLK, &exclusive)
	if lockError == nil {
		return &ProfileLock{file: lockFile}, nil
	}

	closeError := lockFile.Close()

	if errors.Is(lockError, syscall.EAGAIN) || errors.Is(lockError, syscall.EACCES) {
		lockError = ErrProfileInUse
	}

	return nil, errors.Join(fmt.Errorf("lock %s: %w", lockPath, lockError), closeError)
}

// Release drops the lock.
func (lock *ProfileLock) Release() error {
	return lock.file.Close()
}
