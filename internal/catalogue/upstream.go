package catalogue

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/headwalluk/foxtrainer/internal/sources"
	"github.com/headwalluk/foxtrainer/internal/sources/betterfox"
)

// Upstream holds the parsed records of every pinned upstream file: source ID -> file -> records.
type Upstream map[string]map[string][]betterfox.Record

// FetchedFile describes where one upstream file came from, for progress output.
type FetchedFile struct {
	SourceID  string
	File      string
	FromCache bool
	URL       string
}

// LoadUpstream fetches, verifies and parses every pinned file of every source.
func LoadUpstream(ctx context.Context, loaded Catalogue, store sources.Store) (Upstream, []FetchedFile, error) {
	upstream := Upstream{}

	var fetched []FetchedFile

	var problems []error

	for _, sourceID := range sortedKeys(loaded.Sources) {
		source := loaded.Sources[sourceID]
		if source.Format != "betterfox" {
			problems = append(problems, fmt.Errorf("source %s: unsupported format %q", sourceID, source.Format))

			continue
		}

		pin := sources.Pin{SourceID: sourceID, Commit: source.Commit, RawURL: source.RawURL, Files: source.Files}
		upstream[sourceID] = map[string][]betterfox.Record{}

		for _, fileName := range sortedKeys(source.Files) {
			file, fetchError := store.Fetch(ctx, pin, fileName)
			if fetchError != nil {
				problems = append(problems, fetchError)

				continue
			}

			records, parseError := betterfox.Parse(fileName, file.Content)
			if parseError != nil {
				problems = append(problems, fmt.Errorf("source %s: %w", sourceID, parseError))

				continue
			}

			upstream[sourceID][fileName] = records
			fetched = append(fetched, FetchedFile{SourceID: sourceID, File: fileName, FromCache: file.FromCache, URL: file.URL})
		}
	}

	return upstream, fetched, errors.Join(problems...)
}

// sortedKeys returns a map's keys in order.
func sortedKeys[Value any](entries map[string]Value) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
