package pipeline

import (
	ingest "github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// Pipeline orchestrates the media processing workflow.
type Pipeline struct {
	Ingest ingest.IngestPort
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(in ingest.IngestPort) *Pipeline {
	return &Pipeline{
		Ingest: in,
	}
}

// Run executes the pipeline with the given input.
func (p *Pipeline) Run(input RunInput) (RunOutput, error) {
	// Ingest files.
	units, err := p.Ingest.Ingest(input.SourceDir)
	if err != nil {
		return RunOutput{}, err
	}

	// TODO: wire metadata resolution, script generation, rendering.

	return RunOutput{
		FilesProcessed: len(units),
	}, nil
}
