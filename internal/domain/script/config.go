package script

import (
	"github.com/Galdoba/lazyam3/internal/domain/callname"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

// ScriptConfig is the domain input for building a rendering script.
// It carries data from ingest (streams, filenames), metadata (product),
// and environment configuration.
type ScriptConfig struct {
	// From ingest domain
	Filenames        []string               // input filenames in unit order; Filenames[0] is the primary media file (ingest ranks video-containing files first)
	Streams          []ingest.Stream        // all parsed streams
	SRT              string                 // subtitle filename (empty if no subtitles)
	HSUBFile         string                 // hardsub file path (empty if no hardsub)
	Product          metadata.Product       // the episode/season/movie product (with SeriesTitle set)
	StreamSelections []StreamSelection      // optional; auto-select if nil/empty

	// Predictions is the single source of truth for output file names.
	// Populated by the pipeline layer via callname.PredictOutputs.
	Predictions []callname.PredictedOutput

	// CallNames maps original filenames to their call names.
	// Populated by the pipeline layer via callname.BuildInputNames.
	// Used by BuildScript to set SOURCE, SRT, HSUBFILE variables.
	CallNames map[string]string

	// Naming components (derived from Predictations for template use).
	NameTag      string // suffix after BASENAME: "_s01e06", "_s01", "_e06", "" (movie)
	SeasonDir    string // target-dir suffix after BASENAME: "_s01", "" (movie)
	PRTTimestamp string // pipeline-provided timestamp for output file naming

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
	Archive       string // script self-archive location (from TOML)
	TargetRoot    string // e.g. "/mnt/ROOT/EDIT"
	Priority      int
}
