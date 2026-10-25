package pipeline

import (
	"time"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script"
	"github.com/Galdoba/lazyam3/internal/infra/config"
)

// Logger is the structured logging port used by the pipeline.
// Both *slog.Logger and infra logging.Logger satisfy it.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Options configures the Pipeline at construction time.
type Options struct {
	// Infra
	Logger Logger
	Events chan PipelineEvent

	// Domain ports
	Ingest   ingest.Organizer
	Metadata metadata.ProductRepository
	Renderer script.Renderer

	// Config
	SourceDir      string
	Destination    string
	DryRun         bool
	HardsubTag     string              // filename tag marking sidecar subtitles as hardsub (burn-in)
	Harsub         script.HarsubConfig // hardsub burn-in parameters for the subtitles filter
	FfmpegBinary   string              // path to ffmpeg binary for generated scripts
	ExecutorPaths  map[string]string   // executor filesystem paths from config
	PollInterval   time.Duration       // interval between scan passes; 0 → default
	DormancyPeriod time.Duration       // sleep after each pass; 0 → disabled (polling)
	CFG            config.Config
}

// PipelineEvent is a signal emitted by the Pipeline during processing.
type PipelineEvent struct {
	Type  EventType
	ID    int64
	Error error
}

// RunOutput summarizes pipeline processing results.
type RunOutput struct {
	Processed int // units that completed the configured chain
	Failed    int // units that failed ingest/metadata/script
}

// EventType identifies the kind of event.
type EventType string

const (
	EventUnitDone   EventType = "unit_done"
	EventUnitFailed EventType = "unit_failed"
	EventScanDone   EventType = "scan_done"
)
