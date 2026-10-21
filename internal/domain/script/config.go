package script

import (
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// ScriptConfig is the domain input for building a rendering script.
// It carries data from ingest (streams, filenames), metadata (product),
// and environment configuration.
type ScriptConfig struct {
	// From ingest domain
	Filenames        []string          // input filenames in unit order; Filenames[0] is the primary media file (ingest ranks video-containing files first)
	Streams          []ingest.Stream   // all parsed streams
	SRT              string            // subtitle filename (empty if no subtitles)
	HSUBFile         string            // hardsub file path (empty if no hardsub)
	Product          metadata.Product  // the episode/season/movie product (with SeriesTitle set)
	StreamSelections []StreamSelection // optional; auto-select if nil/empty

	// Naming components (populated from Product in BuildScript)
	BaseName   string
	SeasonTag  string
	EpisodeTag string
	// Ready-to-use tags (always present, may be empty — movies have neither).
	// The template composes names from these instead of concatenating raw tags.
	NameTag   string // suffix after BASENAME: "_s01e06", "_s01", "_e06", "" (movie)
	SeasonDir string // target-dir suffix after BASENAME: "_s01", "" (movie)

	// Environment paths (filled from app config)
	Env Environment

	// Output path for the .sh file
	Destination string
}

// Environment holds filesystem paths used by the generated script.
type Environment struct {
	Buffer        string
	InProgress    string
	Done          string
	Notifications string
	TargetRoot    string // e.g. "/mnt/ROOT/EDIT"
	Priority      int
}
