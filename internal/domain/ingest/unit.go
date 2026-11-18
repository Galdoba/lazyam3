package ingest

import (
	"fmt"
)

// MediaUnit aggregates all information about a single media product:
// its physical files and streams across those files.
type MediaUnit struct {
	ID           int64
	Filenames    []string
	Streams      map[string]Stream
	PRTTimestamp string // pipeline-provided timestamp for output file naming (e.g. "261023120000")
	Synthetic    bool   // true when the candidate names agreed on no product ID
}

func newMediaUnit() MediaUnit {
	mu := MediaUnit{}
	mu.Streams = make(map[string]Stream)
	return mu
}

func NewMediaUnit(data map[string][]byte) (MediaUnit, error) {
	mu := newMediaUnit()
	parsed := make(map[string][]Stream, len(data))
	for filename, probe := range data {
		streams, err := parseStreams(probe)
		if err != nil {
			return MediaUnit{}, fmt.Errorf("failed to parse data for %q: %w", filename, err)
		}
		parsed[filename] = streams
	}

	filenames := collectMediaFiles(parsed)
	streams := buildStreams(parsed, filenames)

	id := extractID(filenames...)

	mu.Filenames = filenames
	mu.Streams = streams
	mu.ID = id
	mu.Synthetic = id == 0

	return mu, nil
}

// collectMediaFiles returns sorted filenames that contain at least one
// non-data stream, filtering out pure data files.
func collectMediaFiles(parsed map[string][]Stream) []string {
	files := sortUnitFiles(parsed)
	var result []string
	for _, filename := range files {
		streams := parsed[filename]
		hasMedia := false
		for _, s := range streams {
			if s.Kind != StreamKindData {
				hasMedia = true
				break
			}
		}
		if hasMedia {
			result = append(result, filename)
		}
	}
	return result
}

// buildStreams assigns FileIndex to each stream and returns the coordinate map.
func buildStreams(parsed map[string][]Stream, filenames []string) map[string]Stream {
	streams := make(map[string]Stream)
	for fi, filename := range filenames {
		for _, stream := range parsed[filename] {
			stream.FileIndex = fi
			streams[stream.coordinate()] = stream
		}
	}
	return streams
}

// collectMediaFiles returns sorted filenames that contain at least one
// non-data stream, filtering out pure data files.
