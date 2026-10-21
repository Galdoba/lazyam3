package ingest

import (
	"slices"
	"strings"
)

// sortUnitFiles orders unit files for deterministic processing. Files are
// grouped by content rank — video-containing first, then audio-only, then
// subtitle-only, then others — and lexicographically within each group.
// Filenames[0] of the result is the unit's primary media file whenever one
// exists (units without a video file start with their first audio file).
func sortUnitFiles(files map[string][]Stream) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b string) int {
		if ra, rb := fileRank(files[a]), fileRank(files[b]); ra != rb {
			return ra - rb
		}
		return strings.Compare(a, b)
	})
	return names
}

// fileRank ranks a file by its most significant stream kind:
// video-containing=0, audio-only=1, subtitle-only=2, other=3.
func fileRank(streams []Stream) int {
	var hasVideo, hasAudio, hasSubtitle bool
	for _, s := range streams {
		switch s.Kind {
		case StreamKindVideo:
			hasVideo = true
		case StreamKindAudio:
			hasAudio = true
		case StreamKindSubtitle:
			hasSubtitle = true
		}
	}
	switch {
	case hasVideo:
		return 0
	case hasAudio:
		return 1
	case hasSubtitle:
		return 2
	default:
		return 3
	}
}
