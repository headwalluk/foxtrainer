package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicReplacesContentAndLeavesNoTempFiles(test *testing.T) {
	directory := test.TempDir()
	path := filepath.Join(directory, "user.js")

	for _, content := range []string{"first\n", "second\n"} {
		if writeError := WriteFileAtomic(path, []byte(content), 0o600); writeError != nil {
			test.Fatalf("write %q: %v", content, writeError)
		}
	}

	written, readError := os.ReadFile(path)
	if readError != nil || string(written) != "second\n" {
		test.Errorf("got %q, %v; want \"second\\n\"", written, readError)
	}

	entries, listError := os.ReadDir(directory)
	if listError != nil || len(entries) != 1 {
		test.Errorf("want only user.js in %s, got %d entries (%v)", directory, len(entries), listError)
	}
}

func TestWriteFileAtomicFailsForMissingDirectory(test *testing.T) {
	path := filepath.Join(test.TempDir(), "missing", "user.js")

	if writeError := WriteFileAtomic(path, []byte("x"), 0o600); writeError == nil {
		test.Error("want an error when the folder does not exist")
	}
}
