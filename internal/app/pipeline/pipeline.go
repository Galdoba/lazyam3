package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	appscript "github.com/Galdoba/lazyam3/internal/app/script"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script"
	"github.com/Galdoba/lazyam3/internal/infra/fso"
)

const (
	debugTumeout = 1
)

// Pipeline orchestrates the media processing workflow.
type Pipeline struct {
	logger     Logger
	in         ingest.Organizer
	md         metadata.ProductRepository
	rd         script.Renderer
	hardsubTag string
	harsub     script.HarsubConfig

	sourceDir    string
	destination  string
	execPaths    map[string]string
	dryRun       bool
	ffmpegBinary string
	events       chan PipelineEvent
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(opts Options) *Pipeline {
	return &Pipeline{
		logger:       opts.Logger,
		in:           opts.Ingest,
		md:           opts.Metadata,
		rd:           opts.Renderer,
		hardsubTag:   opts.HardsubTag,
		harsub:       opts.Harsub,
		ffmpegBinary: opts.FfmpegBinary,

		sourceDir:   opts.SourceDir,
		destination: opts.Destination,
		execPaths:   opts.ExecutorPaths,
		dryRun:      opts.DryRun,
		events:      opts.Events,
	}
}

// Run executes the pipeline loop until the context is cancelled.
// Totals are aggregated across all scan passes.
func (p *Pipeline) Run(ctx context.Context) (RunOutput, error) {
	var total RunOutput
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}

		out, err := p.RunOnce(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// keep the units of the interrupted pass in the totals
				total.Processed += out.Processed
				total.Failed += out.Failed
				return total, err
			}
			p.logger.Warn("scan failed", "error", err)
		}
		total.Processed += out.Processed
		total.Failed += out.Failed

		select {
		case <-ctx.Done():
			return total, ctx.Err()
		case <-p.afterPoll():
		}
	}
}

// RunOnce performs a single scan pass: scan → ingest → metadata → script
// build → render. Per-unit failures are counted and emitted as events, not
// returned; the error return covers scan failure and context cancellation.
func (p *Pipeline) RunOnce(ctx context.Context) (RunOutput, error) {
	var out RunOutput
	if err := ctx.Err(); err != nil {
		return out, err
	}

	p.logger.Info("scan")
	snap, err := fso.Scan(p.sourceDir, fso.DefaultOptions())
	if err != nil {
		return out, fmt.Errorf("scan %q: %w", p.sourceDir, err)
	}
	dirs := snap.TopDirsAllFilesWritable()
	p.logger.Info("dirs found", "count", len(dirs))

	for _, d := range dirs {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if err := p.processUnit(ctx, d.Path); err != nil {
			out.Failed++
			continue
		}
		out.Processed++
	}

	p.emit(EventScanDone, 0, nil)
	return out, nil
}

// processUnit runs the full chain for one media unit directory.
func (p *Pipeline) processUnit(ctx context.Context, dir string) error {
	p.logger.Info("processing", "dir", dir)
	unit, err := p.in.Ingest(dir)
	if err != nil {
		p.emit(EventUnitFailed, 0, err)
		return err
	}
	prod, err := p.md.FindByID(unit.ID)
	if err != nil {
		p.logger.Warn("metadata lookup failed, using fallback naming",
			"id", unit.ID, "dir", dir, "error", err)
		prod = fallbackProduct(filepath.Base(dir), unit.ID)
	} else if prod == nil {
		p.logger.Warn("metadata not found, using fallback naming",
			"id", unit.ID, "dir", dir, "fallback_base", filepath.Base(dir))
		prod = fallbackProduct(filepath.Base(dir), unit.ID)
	}
	if p.dryRun {
		p.logger.Info("dry run: script build skipped", "id", unit.ID)
		p.emit(EventUnitDone, unit.ID, nil)
		return nil
	}
	cfg, err := buildScriptConfig(unit, *prod, p.destination, p.execPaths, p.hardsubTag, p.ffmpegBinary, p.harsub)
	if err != nil {
		p.emit(EventUnitFailed, unit.ID, err)
		return err
	}
	// Move source files to OUTPUT directory (same place where scripts land).
	if err := p.prepareFiles(dir, cfg.CallNames, p.destination); err != nil {
		p.emit(EventUnitFailed, unit.ID, fmt.Errorf("prepare files: %w", err))
		return err
	}
	// Build and write the script.
	if err := appscript.Build(ctx, cfg, p.rd); err != nil {
		p.emit(EventUnitFailed, unit.ID, err)
		return err
	}
	p.emit(EventUnitDone, unit.ID, nil)
	return nil
}

// prepareFiles moves each file from the source unit directory to the
// BUFFER directory under its call name. Files that are not in Filenames
// (e.g. already-renamed sidecars) are skipped.
func (p *Pipeline) prepareFiles(sourceDir string, callNames map[string]string, bufferDir string) error {
	if bufferDir == "" {
		return fmt.Errorf("prepare files: buffer directory is required")
	}
	if err := os.MkdirAll(bufferDir, 0o755); err != nil {
		return fmt.Errorf("create buffer dir %q: %w", bufferDir, err)
	}
	for orig, call := range callNames {
		src := filepath.Join(sourceDir, orig)
		dst := filepath.Join(bufferDir, call)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("move %q → %q: %w", src, dst, err)
		}
	}
	return nil
}

func (p *Pipeline) emit(t EventType, id int64, err error) {
	if p.events != nil {
		p.events <- PipelineEvent{Type: t, ID: id, Error: err}
	}
}

// afterPoll returns a channel that fires after the poll interval.
// Overrideable for testing.
func (p *Pipeline) afterPoll() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		// TODO: configurable poll interval from config
		time.Sleep(debugTumeout * time.Second)
	}()
	return ch
}

// fallbackProduct creates a minimal Product from the source directory basename.
// It treats the unit as a movie (no season/episode numbers).
func fallbackProduct(seriesTitle string, unitID int64) *metadata.Product {
	return &metadata.Product{
		SeriesTitle: seriesTitle,
		Kind:        metadata.KindMovie,
		ID:          unitID,
	}
}
