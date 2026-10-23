package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/Galdoba/lazyam3/internal/app"
	"github.com/Galdoba/lazyam3/internal/app/ingest"
	"github.com/Galdoba/lazyam3/internal/app/pipeline"
	"github.com/Galdoba/lazyam3/internal/infra/metadata"
	scriptinfra "github.com/Galdoba/lazyam3/internal/infra/script"
)

const (
	pathIn       = "source-root"
	pathOut      = "target-root"
	pathFfprobe  = "ffprobe-path"
	pathMetadata = "metadata-path"
	pathOnce     = "once"
)

// CMDProcess is the process command — scans input directory, resolves metadata,
// and generates rendering scripts.
func CMDProcess(modules *app.AppModules) *cli.Command {
	return &cli.Command{
		Name:        "process",
		Usage:       "scan input directory and generate rendering scripts",
		Description: "Scans the input directory for media files, resolves product metadata, and generates ffmpeg rendering scripts.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    pathIn,
				Aliases: []string{"in"},
				Usage:   "input directory",
			},
			&cli.StringFlag{
				Name:    pathOut,
				Aliases: []string{"out"},
				Usage:   "output directory",
			},
			&cli.StringFlag{
				Name:  pathFfprobe,
				Usage: "path to ffprobe binary",
			},
			&cli.StringFlag{
				Name:  pathMetadata,
				Usage: "path to metadata.json",
			},
			&cli.BoolFlag{
				Name:  pathOnce,
				Usage: "run a single scan pass and exit instead of polling",
			},
		},
		Action: cmdProcessAction(modules),
	}
}

func cmdProcessAction(modules *app.AppModules) cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		cfg := modules.Config
		logger := modules.Logger.Slog().WithGroup("command")
		logger.Info("create pipeline")

		sourceDir := nonEmpty(c.String(pathIn), cfg.Paths.GeneratorInput())
		if sourceDir == "" {
			logger.Error("source directory required (flag or config field)")
			return fmt.Errorf("source directory required (--%s string or config field)", pathIn)
		}
		ffprobeBinary := nonEmpty(c.String(pathFfprobe), cfg.Paths.GeneratorFFprobe(), "ffprobe")

		metadataPath := nonEmpty(c.String(pathMetadata), cfg.Paths.GeneratorMetadata())
		if metadataPath == "" {
			logger.Error("metadata path required (flag or config field)")
			return fmt.Errorf("metadata path required (--%s string or config field)", pathMetadata)
		}
		mdRepo, err := metadata.New(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to load metadata from %q: %w", metadataPath, err)
		}

		destination := nonEmpty(c.String(pathOut), cfg.Paths.GeneratorOutput())
		if destination == "" {
			logger.Error("output directory required (flag or config field)")
			return fmt.Errorf("output directory required (--%s string or config field)", pathOut)
		}

		events := make(chan pipeline.PipelineEvent, 100)

		p := pipeline.NewPipeline(pipeline.Options{
			Logger:        modules.Logger.Slog().WithGroup("pipeline"),
			Ingest:        ingest.New(ffprobeBinary),
			Metadata:      mdRepo,
			Renderer:      scriptinfra.New(),
			Events:        events,
			SourceDir:     sourceDir,
			Destination:   destination,
			ExecutorPaths: cfg.Paths.Executor,
			HardsubTag:    cfg.Hardsub.Tag,
			CFG:           cfg,
		})

		// TODO: start event consumer goroutine (analytics, TUI, etc.)
		eventsDone := make(chan struct{})
		go func() {
			defer close(eventsDone)
			for e := range events {
				switch e.Type {
				case pipeline.EventUnitDone:
					logger.Info("unit done", "id", e.ID)
				case pipeline.EventUnitFailed:
					logger.Warn("unit failed", "id", e.ID, "error", e.Error)
				}
			}
		}()

		var out pipeline.RunOutput
		if c.Bool(pathOnce) {
			out, err = p.RunOnce(ctx)
		} else {
			out, err = p.Run(ctx)
		}
		// drain pending events before the summary so log order matches reality
		close(events)
		<-eventsDone
		switch {
		case err == nil:
			logger.Info("pipeline finished", "processed", out.Processed, "failed", out.Failed)
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			logger.Info("pipeline stopped", "processed", out.Processed, "failed", out.Failed)
		default:
			logger.Error("pipeline failed", "error", err, "processed", out.Processed, "failed", out.Failed)
			return err
		}
		return nil
	}
}

func nonEmpty(str ...string) string {
	for _, s := range str {
		if s != "" {
			return s
		}
	}
	return ""
}
