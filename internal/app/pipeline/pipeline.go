package pipeline

import (
	"fmt"
	"log/slog"

	filescanner "github.com/Galdoba/lazyam3/filescan"
	ingest "github.com/Galdoba/lazyam3/internal/domain/ingest"
)

// Pipeline orchestrates the media processing workflow.
type Pipeline struct {
	logger *slog.Logger
	Ingest ingest.IngestPort
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(logger *slog.Logger, in ingest.IngestPort) *Pipeline {
	p := Pipeline{
		logger: logger,
		Ingest: in,
	}
	return &p
}

// Run executes the pipeline with the given input.
func (p *Pipeline) Run(input RunInput) (RunOutput, error) {

	// TODO: wire metadata resolution, script generation, rendering.
	// 1. os.ReadDir(rootDir)  →  список директорий (отсортирован, детерминирован)
	// 2. Для каждой директории (последовательно, один воркер):
	//      a. structural check (дёшево, не мутирует)
	//         └─ неполно → reporter.Report(UnitUploading) → continue
	//      b. pipeline.Run(ctx с per-unit timeout, unit)
	//         ├─ err   → reporter.Report(PipelineFailed{ID, ErrClass, Err, At}) → continue
	//         └─ ok    → reporter.Report(UnitDone{ID})  (директорию удалил пайплайн)
	// 3. sleep(pollInterval)
	// 4. goto 1

	try := 0

	for try < 3 {
		try++
		p.logger.Info("scan", "try", try)
		snap, err := filescanner.Scan(input.SourceDir, filescanner.DefaultOptions())
		if err != nil {
			p.logger.Warn("failed to scan", "error", err)
		}
		dirs := snap.TopDirsAllFilesWritable()
		p.logger.Info("dir scanned", "dir", input.SourceDir, "quantity", len(dirs))
		if len(dirs) == 0 {
			continue
		}
		for _, d := range dirs {
			fmt.Println(d.Path)
			for _, f := range d.Files {
				fmt.Println(f.Name)
			}

			unit, err := p.Ingest.Ingest(d.Path)
			if err != nil {
				fmt.Println("ingest failed:", err)
				return RunOutput{}, err
			}
			fmt.Println("UNIT:", unit)
		}

		break
	}
	// Ingest files.
	units, err := p.Ingest.Ingest(input.SourceDir)
	if err != nil {
		return RunOutput{}, err
	}
	fmt.Println("units:", units)
	return RunOutput{
		FilesProcessed: -1,
	}, nil
}
