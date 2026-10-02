//go:build !windows

package firefox

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

// TestHelperProcessHoldsLock is not a real test: re-executed as a child, it holds an fcntl lock like Firefox.
func TestHelperProcessHoldsLock(test *testing.T) {
	if flag.NArg() != 1 {
		test.Skip("helper process only")
	}

	lockFile, openError := os.OpenFile(flag.Arg(0), os.O_RDWR|os.O_CREATE, 0o644)
	if openError != nil {
		test.Fatal(openError)
	}

	exclusive := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: io.SeekStart}
	if lockError := syscall.FcntlFlock(lockFile.Fd(), syscall.F_SETLK, &exclusive); lockError != nil {
		test.Fatal(lockError)
	}

	fmt.Println("locked")

	// Hold the lock until the parent closes our stdin.
	if _, readError := io.ReadAll(os.Stdin); readError != nil {
		test.Fatal(readError)
	}
}

func TestProbeLockSeesAnotherProcessHoldingTheProfile(test *testing.T) {
	profileDir := test.TempDir()

	helper := exec.CommandContext(test.Context(), os.Args[0], "-test.run=^TestHelperProcessHoldsLock$", "--", filepath.Join(profileDir, ".parentlock"))

	helperInput, inputError := helper.StdinPipe()
	if inputError != nil {
		test.Fatal(inputError)
	}

	helperOutput, outputError := helper.StdoutPipe()
	if outputError != nil {
		test.Fatal(outputError)
	}

	if startError := helper.Start(); startError != nil {
		test.Fatal(startError)
	}

	readyLine, readError := bufio.NewReader(helperOutput).ReadString('\n')
	if readError != nil || readyLine != "locked\n" {
		test.Fatalf("helper did not lock: %q, %v", readyLine, readError)
	}

	state, probeError := ProbeLock(profileDir)

	if closeError := helperInput.Close(); closeError != nil {
		test.Errorf("close helper stdin: %v", closeError)
	}

	if waitError := helper.Wait(); waitError != nil {
		test.Errorf("helper: %v", waitError)
	}

	if probeError != nil || !state.InUse || state.HolderPID != helper.Process.Pid {
		test.Errorf("got %+v, %v; want in use by pid %d", state, probeError, helper.Process.Pid)
	}

	released, releasedError := ProbeLock(profileDir)
	if releasedError != nil || released.InUse {
		test.Errorf("after helper exit: got %+v, %v; want not in use", released, releasedError)
	}
}

func TestProbeLockReportsStaleSymlink(test *testing.T) {
	profileDir := test.TempDir()
	writeTestFile(test, filepath.Join(profileDir, ".parentlock"), "")

	if linkError := os.Symlink("127.0.1.1:+3492142", filepath.Join(profileDir, "lock")); linkError != nil {
		test.Fatal(linkError)
	}

	state, probeError := ProbeLock(profileDir)
	if probeError != nil || state.InUse || state.StaleSymlink != "127.0.1.1:+3492142" {
		test.Errorf("got %+v, %v", state, probeError)
	}
}

func TestProbeLockWithoutLockFile(test *testing.T) {
	state, probeError := ProbeLock(test.TempDir())
	if probeError != nil || state != (LockState{}) {
		test.Errorf("got %+v, %v; want zero state", state, probeError)
	}
}

func TestAcquireLockBlocksAnotherProcessAndIsSeenByProbe(test *testing.T) {
	profileDir := test.TempDir()
	writeTestFile(test, filepath.Join(profileDir, ".parentlock"), "keep me")

	lock, acquireError := AcquireLock(profileDir)
	if acquireError != nil {
		test.Fatal(acquireError)
	}

	// A second process must see the lock; our own process never sees its own POSIX locks.
	helper := exec.CommandContext(test.Context(), os.Args[0], "-test.run=^TestHelperProcessHoldsLock$", "--", filepath.Join(profileDir, ".parentlock"))

	output, helperError := helper.CombinedOutput()
	if helperError == nil || bytes.Contains(output, []byte("locked")) {
		test.Errorf("helper should fail to lock while we hold it: %v %s", helperError, output)
	}

	if releaseError := lock.Release(); releaseError != nil {
		test.Fatal(releaseError)
	}

	// Read only after releasing: opening and closing any descriptor on the file would drop our lock.
	content, readError := os.ReadFile(filepath.Join(profileDir, ".parentlock"))
	if readError != nil || string(content) != "keep me" {
		test.Errorf(".parentlock must not be truncated: %q, %v", content, readError)
	}

	state, probeError := ProbeLock(profileDir)
	if probeError != nil || state.InUse {
		test.Errorf("after release: %+v, %v", state, probeError)
	}
}

func TestAcquireLockFailsWhileFirefoxHoldsIt(test *testing.T) {
	profileDir := test.TempDir()

	helper := exec.CommandContext(test.Context(), os.Args[0], "-test.run=^TestHelperProcessHoldsLock$", "--", filepath.Join(profileDir, ".parentlock"))

	helperInput, inputError := helper.StdinPipe()
	if inputError != nil {
		test.Fatal(inputError)
	}

	helperOutput, outputError := helper.StdoutPipe()
	if outputError != nil {
		test.Fatal(outputError)
	}

	if startError := helper.Start(); startError != nil {
		test.Fatal(startError)
	}

	if readyLine, readError := bufio.NewReader(helperOutput).ReadString('\n'); readError != nil || readyLine != "locked\n" {
		test.Fatalf("helper did not lock: %q, %v", readyLine, readError)
	}

	_, acquireError := AcquireLock(profileDir)

	if closeError := helperInput.Close(); closeError != nil {
		test.Error(closeError)
	}

	if waitError := helper.Wait(); waitError != nil {
		test.Error(waitError)
	}

	if !errors.Is(acquireError, ErrProfileInUse) {
		test.Errorf("want ErrProfileInUse, got %v", acquireError)
	}
}
