package ingest

import "fmt"

// MediaUnit aggregates all information about a single media product:
// its physical files, streams across those files, and validation issues.
type MediaUnit struct {
	ID         int64
	Filepaths  []string
	Streams    []Stream
	Validation ValidationIssues
}

// ValidationIssues accumulates problems found during ingest.
type ValidationIssues []ValidationIssue

// ValidationIssue describes a specific problem found during ingest.
type ValidationIssue struct {
	StreamIndex int    // -1 if file-level, >= 0 if stream-level
	Field       string // e.g. "codec", "encoding", "readiness"
	Problem     string // human-readable description
}

// ValidateMediaUnit returns domain-level validation issues for the given streams.
func ValidateMediaUnit(streams []Stream) ValidationIssues {
	var issues ValidationIssues
	//TODO: fill with validation rules
	for i, s := range streams {
		if s.Kind == StreamKindVideo && s.BitRate == 0 {
			issues = append(issues, ValidationIssue{
				StreamIndex: i,
				Field:       "bitrate",
				Problem:     "video stream has no bitrate",
			})
		}
	}
	return issues
}

// ValidateMediaUnitWithFilepaths returns validation issues for a complete MediaUnit.
func ValidateMediaUnitWithFilepaths(m MediaUnit) ValidationIssues {
	return ValidateMediaUnit(m.Streams)
}

// validate returns domain-level validation issues for this unit.
// Internal to domain — exported only when app layer needs to inspect issues.
func (m *MediaUnit) validate() ValidationIssues {
	return ValidateMediaUnit(m.Streams)
}

// String returns a human-readable representation of a ValidationIssue.
func (v ValidationIssue) String() string {
	return fmt.Sprintf("stream[%d].%s: %s", v.StreamIndex, v.Field, v.Problem)
}
