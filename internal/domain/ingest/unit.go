package ingest

import (
	"fmt"
)

// MediaUnit aggregates all information about a single media product:
// its physical files, streams across those files, and validation issues.
type MediaUnit struct {
	ID           int64
	Filenames    []string
	Streams      map[string]Stream
	Validation   ValidationIssues
	PRTTimestamp string // pipeline-provided timestamp for output file naming (e.g. "261023120000")
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
	for fi, filename := range sortUnitFiles(parsed) {
		streams := parsed[filename]
		// skip files that have no non-data streams
		hasMedia := false
		for _, s := range streams {
			if s.Kind != StreamKindData {
				hasMedia = true
				break
			}
		}
		if !hasMedia {
			continue
		}
		mu.Filenames = append(mu.Filenames, filename)
		for _, stream := range streams {
			stream.FileIndex = fi
			mu.Streams[stream.coordinate()] = stream
		}
		id, err := ExtractID(filename)
		if err != nil {
			return MediaUnit{}, err
		}
		if mu.ID != 0 && mu.ID != id {
			return MediaUnit{}, fmt.Errorf("inconsistent IDs: %d and %d in same unit", mu.ID, id)
		} else {
			mu.ID = id
		}
	}
	return mu, nil
}

// ValidationIssues accumulates problems found during ingest.
type ValidationIssues []ValidationIssue

// ValidationIssue describes a specific problem found during ingest.
type ValidationIssue struct {
	StreamIndex int    // -1 if file-level, >= 0 if stream-level
	Field       string // e.g. "resolution", "bitrate", "duration", "coordinate", "duration_consistency"
	Problem     string // human-readable description
}

// rationalMean returns the average of a list of rational numbers as a float64.
func rationalMeanFloat(rs []Rational) float64 {
	if len(rs) == 0 {
		return 0
	}
	var sum float64
	for _, r := range rs {
		if r.Denom == 0 {
			continue
		}
		sum += float64(r.Num) / float64(r.Denom)
	}
	return sum / float64(len(rs))
}

// ValidateMediaUnit returns domain-level validation issues for the given streams.
func ValidateMediaUnit(streams []Stream) ValidationIssues {
	var issues ValidationIssues

	// Track durations for multi-file consistency check
	var durations []Rational

	// Track seen coordinates for duplicate detection
	seen := make(map[string]bool)

	for i, s := range streams {
		// Duplicate stream detection
		if seen[s.coordinate()] {
			issues = append(issues, ValidationIssue{
				StreamIndex: i,
				Field:       "coordinate",
				Problem:     "duplicate stream coordinate",
			})
		}
		seen[s.coordinate()] = true

		if s.Kind == StreamKindVideo {
			// Resolution must be at least SD
			if s.Width > 0 && s.Height > 0 && (s.Width < 720 || s.Height < 576) {
				issues = append(issues, ValidationIssue{
					StreamIndex: i,
					Field:       "resolution",
					Problem:     fmt.Sprintf("video resolution %dx%d is below SD (720x576)", s.Width, s.Height),
				})
			}
			// Duration must be non-negative
			if s.Duration.Num < 0 {
				issues = append(issues, ValidationIssue{
					StreamIndex: i,
					Field:       "duration",
					Problem:     "video has negative duration",
				})
			}
			durations = append(durations, s.Duration)
			// Bitrate sanity
			if s.BitRate > 0 && s.BitRate < 100000 {
				issues = append(issues, ValidationIssue{
					StreamIndex: i,
					Field:       "bitrate",
					Problem:     fmt.Sprintf("video bitrate %d bps is suspiciously low", s.BitRate),
				})
			}
		}

		if s.Kind == StreamKindAudio {
			if s.Duration.Num < 0 {
				issues = append(issues, ValidationIssue{
					StreamIndex: i,
					Field:       "duration",
					Problem:     "audio has negative duration",
				})
			}
			durations = append(durations, s.Duration)
			if s.BitRate > 0 && s.BitRate < 32000 {
				issues = append(issues, ValidationIssue{
					StreamIndex: i,
					Field:       "bitrate",
					Problem:     fmt.Sprintf("audio bitrate %d bps is suspiciously low", s.BitRate),
				})
			}
		}
	}

	// Multi-file duration consistency (audio/video only)
	if len(durations) > 1 {
		mean := rationalMeanFloat(durations)
		if mean > 0 {
			for _, d := range durations {
				if d.Num <= 0 {
					continue
				}
				val := float64(d.Num) / float64(d.Denom)
				diff := val - mean
				if diff < 0 {
					diff = -diff
				}
				if diff > mean*0.1 { // ±10%
					issues = append(issues, ValidationIssue{
						StreamIndex: -1,
						Field:       "duration_consistency",
						Problem:     "stream durations vary by more than 10%",
					})
					break
				}
			}
		}
	}

	return issues
}
