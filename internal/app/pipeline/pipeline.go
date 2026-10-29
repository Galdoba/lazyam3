package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appscript "github.com/Galdoba/lazyam3/internal/app/script"
	"github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/Galdoba/lazyam3/internal/domain/metadata"
	"github.com/Galdoba/lazyam3/internal/domain/script"
	"github.com/Galdoba/lazyam3/internal/infra/fso"
	"github.com/Galdoba/lazyam3/internal/infra/logging"
	"github.com/Galdoba/lazyam3/internal/platform"
)

const (
	// defaultPollInterval is used when Options.PollInterval is not set.
	defaultPollInterval = 1 * time.Second
)

// Pipeline orchestrates the media processing workflow.
type Pipeline struct {
	logger     Logger
	in         ingest.Organizer
	md         metadata.ProductRepository
	rd         script.Renderer
	hardsubTag string
	harsub     script.HarsubConfig

	sourceDir      string
	destination    string
	execPaths      map[string]string
	dryRun         bool
	ffmpegBinary   string
	pollInterval   time.Duration
	dormancyPeriod time.Duration
	events         chan PipelineEvent

	pathBuilder platform.PathBuilder

	statusPath string
}

// NewPipeline creates a new Pipeline instance.
func NewPipeline(opts Options) *Pipeline {
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	statusPath := ""
	if opts.CFG.Storage.Path != "" {
		statusPath = opts.CFG.Storage.Path
	}
	return &Pipeline{
		logger:         opts.Logger,
		in:             opts.Ingest,
		md:             opts.Metadata,
		rd:             opts.Renderer,
		hardsubTag:     opts.HardsubTag,
		harsub:         opts.Harsub,
		ffmpegBinary:   opts.FfmpegBinary,
		pollInterval:   pollInterval,
		dormancyPeriod: opts.DormancyPeriod,

		sourceDir:   opts.SourceDir,
		destination: opts.Destination,
		execPaths:   opts.ExecutorPaths,
		dryRun:      opts.DryRun,
		events:      opts.Events,
		pathBuilder: opts.PathBuilder,
		statusPath:  statusPath,
	}
}

// Run executes the pipeline loop until the context is cancelled.
// Totals are aggregated across all scan passes.
func (p *Pipeline) Run(ctx context.Context) (RunOutput, error) {
	var total RunOutput
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	p.logger.Info("pipeline loop started", "poll_interval", p.pollInterval)
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("pipeline stopped", "error", ctx.Err())
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
		if p.statusPath != "" {
			p.writeStatus(total)
		}

		if p.dormancyPeriod > 0 {
			timer := time.NewTimer(p.dormancyPeriod)
			select {
			case <-ctx.Done():
				total.Processed += out.Processed
				total.Failed += out.Failed
				timer.Stop()
				return total, ctx.Err()
			case <-timer.C:
			}
		} else {
			select {
			case <-ctx.Done():
				return total, ctx.Err()
			case <-ticker.C:
			}
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

	metaDir := p.pathBuilder.Canonical("metadata")
	matches := make([]doneMatch, 0)
	for _, d := range dirs {
		if len(snap.FilesIn(d.Path)) > 0 {
			continue
		}
		donePath, ok := p.findDoneFile(metaDir, d.Path)
		if ok {
			snap.SanitizeablePaths = append(snap.SanitizeablePaths, d.Path)
			matches = append(matches, doneMatch{sourceDir: d.Path, donePath: donePath})
		}
	}
	if len(snap.SanitizeablePaths) > 0 {
		if err := snap.Sanitize(); err != nil {
			p.logger.Warn("sanitize failed", "error", err)
		} else {
			p.logger.Info("sanitized empty dirs", "count", len(snap.SanitizeablePaths))
			for _, m := range matches {
				if err := os.Remove(m.donePath); err != nil && !errors.Is(err, os.ErrNotExist) {
					p.logger.Warn("remove done file failed", "path", m.donePath, "error", err)
				}
			}
		}
	}

	for _, d := range dirs {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if len(snap.FilesIn(d.Path)) == 0 {
			continue
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
	buildLogger, ok := p.logger.(*logging.Logger)
	if !ok {
		buildLogger = nil
	}
	if err := appscript.Build(buildLogger, ctx, cfg, p.rd); err != nil {
		p.emit(EventUnitFailed, unit.ID, err)
		return err
	}
	// Write processing artifact after successful script build, before execution.
	scriptName := filepath.Base(cfg.Destination)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(cfg.Destination))
	metaDir := p.pathBuilder.Canonical("metadata")
	if err := os.MkdirAll(metaDir, 0o755); err == nil {
		processingPath := p.pathBuilder.Canonical("metadata", scriptName+".processing")
		payload := map[string]any{
			"source_dir":  dir,
			"unit_id":     unit.ID,
			"script_name": scriptName,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			if p.logger != nil {
				p.logger.Warn("processing artifact marshal failed", "error", err)
			}
		} else if err := os.WriteFile(processingPath, payloadBytes, 0o644); err != nil {
			if p.logger != nil {
				p.logger.Warn("processing artifact write failed", "path", processingPath, "error", err)
			}
		}
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

// doneMatch pairs a source dir path with its consumed .done artifact.
type doneMatch struct {
	sourceDir string
	donePath  string
}

// findDoneFile scans metaDir for .done files and returns the path of the
// one whose JSON source_dir matches sourceDir exactly.
func (p *Pipeline) findDoneFile(metaDir, sourceDir string) (string, bool) {
	entries, err := os.ReadDir(metaDir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".done" {
			continue
		}
		donePath := p.pathBuilder.Canonical("metadata", e.Name())
		data, err := os.ReadFile(donePath)
		if err != nil {
			continue
		}
		payload := map[string]any{}
		if err := json.Unmarshal(data, &payload); err != nil {
			if p.logger != nil {
				p.logger.Warn("done file parse failed", "path", donePath, "error", err)
			}
			continue
		}
		sd, _ := payload["source_dir"].(string)
		if sd == sourceDir {
			return donePath, true
		}
	}
	return "", false
}

func (p *Pipeline) emit(t EventType, id int64, err error) {
	if p.events != nil {
		p.events <- PipelineEvent{Type: t, ID: id, Error: err}
	}
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

// StatusReport is the JSON structure written to status.json after each loop.
type StatusReport struct {
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
	Status    string `json:"status"`
}

// writeStatus persists the current processing totals to status.json.
func (p *Pipeline) writeStatus(out RunOutput) {
	if p.statusPath == "" {
		return
	}
	report := StatusReport{
		Processed: out.Processed,
		Failed:    out.Failed,
		Status:    "ok",
	}
	raw, err := json.Marshal(report)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("status marshal failed", "error", err)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(p.statusPath), 0o755); err != nil {
		if p.logger != nil {
			p.logger.Warn("status dir create failed", "path", p.statusPath, "error", err)
		}
		return
	}
	if err := os.WriteFile(p.statusPath, raw, 0o644); err != nil {
		if p.logger != nil {
			p.logger.Warn("status write failed", "path", p.statusPath, "error", err)
		}
	}
}
