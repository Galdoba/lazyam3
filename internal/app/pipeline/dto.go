package pipeline

import (
	"log/slog"

	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script"
	"github.com/Galdoba/lazyam3/internal/infra/config"
)

// Options configures the Pipeline at construction time.
type Options struct {
	// Infra
	Logger *slog.Logger
	Events chan PipelineEvent

	// Domain ports
	Ingest   ingest.Organizer
	Metadata metadata.ProductRepository
	Renderer script.Renderer

	// Config
	SourceDir     string
	Destination   string
	DryRun        bool
	HardsubTag    string            // filename tag marking sidecar subtitles as hardsub (burn-in)
	ExecutorPaths map[string]string // executor filesystem paths from config
	CFG           config.Config
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
