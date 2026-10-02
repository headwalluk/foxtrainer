package sources

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// pinFor returns a pin serving content as file.js from serverURL.
func pinFor(serverURL string, content []byte) Pin {
	digest := sha256.Sum256(content)

	return Pin{
		SourceID: "upstream",
		Commit:   "0123456789abcdef0123456789abcdef01234567",
		RawURL:   serverURL + "/{commit}/{file}",
		Files:    map[string]string{"file.js": hex.EncodeToString(digest[:])},
	}
}

func TestFetchDownloadsVerifiesAndCaches(test *testing.T) {
	content := []byte(`user_pref("a.b", true);` + "\n")

	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)

		if !strings.HasSuffix(request.URL.Path, "/0123456789abcdef0123456789abcdef01234567/file.js") {
			http.NotFound(writer, request)

			return
		}

		_, _ = writer.Write(content)
	}))
	defer server.Close()

	store := Store{CacheDir: test.TempDir(), Client: server.Client()}
	pin := pinFor(server.URL, content)

	first, firstError := store.Fetch(test.Context(), pin, "file.js")
	if firstError != nil || first.FromCache || string(first.Content) != string(content) {
		test.Fatalf("first fetch: %+v, %v", first, firstError)
	}

	second, secondError := store.Fetch(test.Context(), pin, "file.js")
	if secondError != nil || !second.FromCache {
		test.Fatalf("second fetch should come from cache: %+v, %v", second, secondError)
	}

	if requestCount.Load() != 1 {
		test.Errorf("want 1 download, got %d", requestCount.Load())
	}

	offline := Store{CacheDir: store.CacheDir, Offline: true}
	if _, offlineError := offline.Fetch(test.Context(), pin, "file.js"); offlineError != nil {
		test.Errorf("offline fetch from cache: %v", offlineError)
	}
}

func TestFetchRejectsTamperedContent(test *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`user_pref("evil", true);`))
	}))
	defer server.Close()

	cacheDir := test.TempDir()
	store := Store{CacheDir: cacheDir, Client: server.Client()}

	_, fetchError := store.Fetch(test.Context(), pinFor(server.URL, []byte("the real content")), "file.js")
	if fetchError == nil || !strings.Contains(fetchError.Error(), "sha256 mismatch") {
		test.Fatalf("want a sha256 mismatch, got %v", fetchError)
	}

	cached, _ := filepath.Glob(filepath.Join(cacheDir, "sources", "*", "*", "*"))
	if len(cached) != 0 {
		test.Errorf("tampered content must not be cached: %v", cached)
	}
}

func TestFetchRefetchesCorruptCache(test *testing.T) {
	content := []byte("good")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write(content)
	}))
	defer server.Close()

	store := Store{CacheDir: test.TempDir(), Client: server.Client()}
	pin := pinFor(server.URL, content)
	cachePath := filepath.Join(store.CacheDir, "sources", pin.SourceID, pin.Commit, "file.js")

	if mkdirError := os.MkdirAll(filepath.Dir(cachePath), 0o700); mkdirError != nil {
		test.Fatal(mkdirError)
	}

	if writeError := os.WriteFile(cachePath, []byte("corrupt"), 0o600); writeError != nil {
		test.Fatal(writeError)
	}

	fetched, fetchError := store.Fetch(test.Context(), pin, "file.js")
	if fetchError != nil || fetched.FromCache || string(fetched.Content) != "good" {
		test.Errorf("got %+v, %v", fetched, fetchError)
	}
}

func TestFetchUnpinnedFileIsAnError(test *testing.T) {
	_, fetchError := Store{CacheDir: test.TempDir(), Offline: true}.Fetch(test.Context(), Pin{SourceID: "upstream"}, "other.js")
	if fetchError == nil {
		test.Error("want an error for a file not in the pin")
	}
}
