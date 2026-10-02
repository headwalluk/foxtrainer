// Package fsutil provides filesystem helpers shared across foxtrainer.
package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path via a synced temp file and rename, so readers never see a partial file.
func WriteFileAtomic(path string, data []byte, permissions os.FileMode) error {
	directory := filepath.Dir(path)

	tempFile, createError := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if createError != nil {
		return fmt.Errorf("create temp file for %s: %w", path, createError)
	}

	tempPath := tempFile.Name()

	writeError := writeAndClose(tempFile, data, permissions)
	if writeError == nil {
		writeError = os.Rename(tempPath, path)
	}

	if writeError != nil {
		removeError := os.Remove(tempPath)
		if removeError != nil && !errors.Is(removeError, os.ErrNotExist) {
			writeError = errors.Join(writeError, fmt.Errorf("remove temp file %s: %w", tempPath, removeError))
		}

		return fmt.Errorf("write %s: %w", path, writeError)
	}

	return syncDirectory(directory)
}

// writeAndClose writes, chmods, syncs and closes file, joining any close error with earlier ones.
func writeAndClose(file *os.File, data []byte, permissions os.FileMode) error {
	_, writeError := file.Write(data)
	if writeError == nil {
		writeError = file.Chmod(permissions)
	}

	if writeError == nil {
		writeError = file.Sync()
	}

	return errors.Join(writeError, file.Close())
}

// syncDirectory flushes a directory entry so a rename survives a crash.
func syncDirectory(directory string) error {
	handle, openError := os.Open(directory)
	if openError != nil {
		return fmt.Errorf("open %s for sync: %w", directory, openError)
	}

	syncError := handle.Sync()

	joined := errors.Join(syncError, handle.Close())
	if joined != nil {
		joined = fmt.Errorf("sync directory %s: %w", directory, joined)
	}

	return joined
}
