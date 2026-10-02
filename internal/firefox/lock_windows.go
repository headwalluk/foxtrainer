//go:build windows

package firefox

import "errors"

// errLockProbeUnsupported is returned until the Windows parent.lock probe is implemented.
var errLockProbeUnsupported = errors.New("profile lock detection is not implemented on Windows yet")

// probeProfileLock is not yet implemented on Windows.
func probeProfileLock(string) (LockState, error) {
	return LockState{}, errLockProbeUnsupported
}

// ProfileLock is not yet implemented on Windows.
type ProfileLock struct{}

// AcquireLock is not yet implemented on Windows.
func AcquireLock(string) (*ProfileLock, error) {
	return nil, errLockProbeUnsupported
}

// Release is not yet implemented on Windows.
func (lock *ProfileLock) Release() error {
	return errLockProbeUnsupported
}
