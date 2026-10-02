// Package sources downloads pinned upstream files, verifies them by sha256 and caches them.
package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/headwalluk/foxtrainer/internal/fsutil"
)

// maxFileBytes caps a single upstream download; Betterfox's largest file is about 100 KB.
const maxFileBytes = 8 << 20

// Pin identifies an upstream file set at an exact commit.
type Pin struct {
	SourceID string
	Commit   string            // full 40-character SHA
	RawURL   string            // template containing {commit} and {file}
	Files    map[string]string // file name -> expected sha256 (hex)
}

// Store fetches pinned files, preferring a verified cached copy.
type Store struct {
	CacheDir string
	Client   *http.Client
	Offline  bool // never download; use the cache only
}

// Fetched is a verified upstream file.
type Fetched struct {
	Content   []byte
	FromCache bool
	URL       string
}

// errChecksumMismatch means a file's content does not match its pinned sha256.
var errChecksumMismatch = errors.New("sha256 mismatch")

// Fetch returns the verified content of fileName at the pin, downloading it if the cache lacks a good copy.
func (store Store) Fetch(ctx context.Context, pin Pin, fileName string) (Fetched, error) {
	expected, pinned := pin.Files[fileName]
	if !pinned {
		return Fetched{}, fmt.Errorf("%s: %s is not pinned in the catalogue", pin.SourceID, fileName)
	}

	fileURL := strings.NewReplacer("{commit}", pin.Commit, "{file}", fileName).Replace(pin.RawURL)
	cachePath := filepath.Join(store.CacheDir, "sources", pin.SourceID, pin.Commit, fileName)

	cached, cacheError := readVerified(cachePath, expected)
	if cacheError == nil {
		return Fetched{Content: cached, FromCache: true, URL: fileURL}, nil
	}

	if store.Offline {
		return Fetched{}, fmt.Errorf("%s %s: not cached and offline: %w", pin.SourceID, fileName, cacheError)
	}

	downloaded, downloadError := store.download(ctx, fileURL)
	if downloadError != nil {
		return Fetched{}, fmt.Errorf("%s %s: %w", pin.SourceID, fileName, downloadError)
	}

	if verifyError := verify(downloaded, expected); verifyError != nil {
		return Fetched{}, fmt.Errorf("%s %s from %s: %w", pin.SourceID, fileName, fileURL, verifyError)
	}

	if mkdirError := os.MkdirAll(filepath.Dir(cachePath), 0o700); mkdirError != nil {
		return Fetched{}, fmt.Errorf("create cache folder: %w", mkdirError)
	}

	if writeError := fsutil.WriteFileAtomic(cachePath, downloaded, 0o600); writeError != nil {
		return Fetched{}, fmt.Errorf("cache %s: %w", fileName, writeError)
	}

	return Fetched{Content: downloaded, URL: fileURL}, nil
}

// download GETs fileURL with a size cap.
func (store Store) download(ctx context.Context, fileURL string) ([]byte, error) {
	client := store.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	request, requestError := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if requestError != nil {
		return nil, fmt.Errorf("build request: %w", requestError)
	}

	response, responseError := client.Do(request)
	if responseError != nil {
		return nil, fmt.Errorf("download: %w", responseError)
	}

	body, readError := io.ReadAll(io.LimitReader(response.Body, maxFileBytes+1))
	closeError := response.Body.Close()

	var downloadError error

	switch {
	case readError != nil:
		downloadError = fmt.Errorf("read %s: %w", fileURL, readError)
	case closeError != nil:
		downloadError = fmt.Errorf("close %s: %w", fileURL, closeError)
	case response.StatusCode != http.StatusOK:
		downloadError = fmt.Errorf("download %s: HTTP %d", fileURL, response.StatusCode)
	case len(body) > maxFileBytes:
		downloadError = fmt.Errorf("download %s: larger than %d bytes", fileURL, maxFileBytes)
	}

	return body, downloadError
}

// readVerified returns the cached file at path if it exists and matches expected.
func readVerified(path, expected string) ([]byte, error) {
	content, readError := os.ReadFile(path)
	if readError != nil {
		return nil, fmt.Errorf("read cache: %w", readError)
	}

	return content, verify(content, expected)
}

// verify checks content against an expected hex sha256.
func verify(content []byte, expected string) error {
	digest := sha256.Sum256(content)
	actual := hex.EncodeToString(digest[:])

	var verifyError error
	if !strings.EqualFold(actual, expected) {
		verifyError = fmt.Errorf("%w: got %s, want %s", errChecksumMismatch, actual, expected)
	}

	return verifyError
}
