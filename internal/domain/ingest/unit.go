package ingest

import (
	"fmt"
)

const (
	// SyntheticID is the fallback product ID assigned when a filename
	// does not start with a digit (no product ID could be extracted).
	SyntheticID int64 = 9999
)

// MediaUnit aggregates all information about a single media product:
// its physical files, streams across those files, and validation issues.
type MediaUnit struct {
	ID           int64
	Filenames    []string
	Streams      map[string]Stream
	Validation   ValidationIssues
	PRTTimestamp string // pipeline-provided timestamp for output file naming (e.g. "261023120000")
	Synthetic    bool   // true when ID was synthetically assigned (filename had no leading digits)
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
	streams, _ := buildStreams(parsed, filenames)

	id, synthetic, err := resolveID(filenames)
	if err != nil {
		return MediaUnit{}, err
	}

	mu.Filenames = filenames
	mu.Streams = streams
	mu.ID = id
	mu.Synthetic = synthetic

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
func buildStreams(parsed map[string][]Stream, filenames []string) (map[string]Stream, int) {
	streams := make(map[string]Stream)
	for fi, filename := range filenames {
		for _, stream := range parsed[filename] {
			stream.FileIndex = fi
			streams[stream.coordinate()] = stream
		}
	}
	return streams, len(filenames)
}

// resolveID extracts product IDs from filenames, checks consistency, and
// returns the unit ID along with whether it was synthetically assigned.
func resolveID(filenames []string) (int64, bool, error) {
	id := int64(0)
	synthetic := false
	for _, filename := range filenames {
		fid, err := ExtractID(filename)
		if err != nil {
			return 0, false, err
		}
		if id != 0 && id != fid {
			return 0, false, fmt.Errorf("inconsistent IDs: %d and %d in same unit", id, fid)
		}
		if id == 0 {
			id = fid
			if fid == SyntheticID {
				synthetic = true
			}
		}
	}
	return id, synthetic, nil
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

		if s.Kind == StreamKindVideo || s.Kind == StreamKindAudio {
			durations = append(durations, s.Duration)
		}

		issues = validateVideo(s, i, issues)
		issues = validateAudio(s, i, issues)
	}

	return validateDurations(durations, issues)
}

// validateVideo appends validation issues for a video stream.
func validateVideo(s Stream, i int, issues ValidationIssues) ValidationIssues {
	if s.Kind != StreamKindVideo {
		return issues
	}
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
	// Bitrate sanity
	if s.BitRate > 0 && s.BitRate < 100000 {
		issues = append(issues, ValidationIssue{
			StreamIndex: i,
			Field:       "bitrate",
			Problem:     fmt.Sprintf("video bitrate %d bps is suspiciously low", s.BitRate),
		})
	}
	return issues
}

// validateAudio appends validation issues for an audio stream.
func validateAudio(s Stream, i int, issues ValidationIssues) ValidationIssues {
	if s.Kind != StreamKindAudio {
		return issues
	}
	if s.Duration.Num < 0 {
		issues = append(issues, ValidationIssue{
			StreamIndex: i,
			Field:       "duration",
			Problem:     "audio has negative duration",
		})
	}
	if s.BitRate > 0 && s.BitRate < 32000 {
		issues = append(issues, ValidationIssue{
			StreamIndex: i,
			Field:       "bitrate",
			Problem:     fmt.Sprintf("audio bitrate %d bps is suspiciously low", s.BitRate),
		})
	}
	return issues
}

// validateDurations appends a duration consistency issue if streams vary by more than 10%.
func validateDurations(durations []Rational, issues ValidationIssues) ValidationIssues {
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
