package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/lazyam3/internal/app/ingest"
	"github.com/Galdoba/lazyam3/internal/app/pipeline"
	dom "github.com/Galdoba/lazyam3/internal/domain/ingest"
	"github.com/urfave/cli/v3"
)

// CMDProcess is the process command — scans input directory, resolves metadata,
// and generates rendering scripts.
func CMDProcess() *cli.Command {
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
		Action: cmdProcessAction,
	}
}

func cmdProcessAction(ctx context.Context, cmd *cli.Command) error {
	sourceDir := cmd.String("source")
	outputDir := cmd.String("output")

	if sourceDir == "" {
		return fmt.Errorf("source directory required (--source)")
	}

	ing := &ingest.Ingest{}
	p := pipeline.NewPipeline(dom.IngestPort(ing))
	result, err := p.Run(pipeline.RunInput{
		SourceDir:   sourceDir,
		Destination: outputDir,
	})
	if err != nil {
		return fmt.Errorf("pipeline failed: %w", err)
	}

	_, err = fmt.Fprintf(cmd.Root().Writer, "processed %d files → %s\n", result.FilesProcessed, result.ScriptWritten)
	return err
}
