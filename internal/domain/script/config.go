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
	Filenames        []string          // input filenames in unit order; Filenames[0] is the primary media file (ingest ranks video-containing files first)
	Streams          []ingest.Stream   // all parsed streams
	SRT              string            // subtitle filename (empty if no subtitles)
	HSUBFile         string            // hardsub file path (empty if no hardsub)
	Product          metadata.Product  // the episode/season/movie product (with SeriesTitle set)
	StreamSelections []StreamSelection // optional; auto-select if nil/empty

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

	// FfmpegBinary is the path to the ffmpeg binary used in the generated script.
	FfmpegBinary string

	// Harsub holds hardsub burn-in parameters for the subtitles filter.
	// Empty values fall back to the built-in defaults.
	Harsub HarsubConfig

	// ScriptName is the base name of the script (e.g. "Show_s01e06_PRT261023120000") used for execution-signal markers.
	ScriptName string

	// Output path for the .sh file
	Destination string
}

// HarsubConfig holds hardsub burn-in parameters for the ffmpeg subtitles filter.
// Zero values fall back to the built-in defaults.
type HarsubConfig struct {
	OriginalSize    string  // subtitle stream size for the subtitles filter, e.g. "1920x1080"
	FontName        string  // force_style font name, e.g. "Segoe UI"
	FontSize        float64 // force_style font size, e.g. 19.4
	PrimaryColour   string  // force_style primary colour, e.g. "&H00EBEBEB"
	SecondaryColour string  // force_style secondary colour, e.g. "&H000000FF"
	OutlineColour   string  // force_style outline colour, e.g. "&H00000000"
	BackColour      string  // force_style back colour, e.g. "&H80000000"
	Underline       int     // force_style underline
	StrikeOut       int     // force_style strikeout
	ScaleX          float64 // force_style scale_x
	ScaleY          float64 // force_style scale_y
	Spacing         int     // force_style spacing
	Angle           int     // force_style angle
	BorderStyle     int     // force_style border_style
	Outline         float64 // force_style outline
	Shadow          int     // force_style shadow
	Alignment       int     // force_style alignment
	MarginL         int     // force_style margin_l
	MarginR         int     // force_style margin_r
	MarginV         int     // force_style margin_v
	Encoding        int     // subtitles filter encoding
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
	MetadataDir   string // metadata directory for execution-signal markers (.processing/.done)
}
