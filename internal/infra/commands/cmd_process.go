package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/lazyam3/internal/app"
	"github.com/Galdoba/lazyam3/internal/app/ingest"
	"github.com/Galdoba/lazyam3/internal/app/pipeline"
	dom "github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/urfave/cli/v3"
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
				Name:    "source",
				Aliases: []string{"s"},
				Usage:   "input directory",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output directory",
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

		sourceDir := nonEmpty(c.String("source"), cfg.Paths.InputDirectory)
		outputDir := c.String("output")
		if sourceDir == "" {
			logger.Error("source directory required (flag or config field) ")
			return fmt.Errorf("source directory required (--source)")
		}
		fmt.Println("create pipe")
		ing := &ingest.Ingest{}
		p := pipeline.NewPipeline(
			modules.Logger.Slog().WithGroup("pipeline"),
			dom.IngestPort(ing),
		)
		result, err := p.Run(pipeline.RunInput{
			SourceDir:   sourceDir,
			Destination: outputDir,
		})
		if err != nil {
			return fmt.Errorf("pipeline failed: %w", err)
		}

		_, err = fmt.Fprintf(c.Root().Writer, "processed %d files → %s\n", result.FilesProcessed, result.ScriptWritten)
		return err

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
