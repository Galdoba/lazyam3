package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Galdoba/lazyam3/internal/app"
	"github.com/Galdoba/lazyam3/internal/infra/commands"
	"github.com/urfave/cli/v3"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Empty container. Config and Logger will be populated in Root.Before
	// after flag parsing. Other modules are lazy accessors.
	modules := app.New()

	root := &cli.Command{
		Name: "", // пусто — urfave/cli подставит имя бинарника
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "path to config file (overrides env and default)",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Root flags are already parsed here, --config is available.
			// Create exactly what is needed for logging startup errors:
			// config and logger. Everything else — on demand from commands.
			if err := modules.Init(cmd.String("config")); err != nil {
				return ctx, err
			}
			return ctx, nil
		},
		After: func(ctx context.Context, cmd *cli.Command) error {
			// Closes everything that was actually created by lazy accessors.
			return modules.Shutdown()
		},
		Commands: []*cli.Command{
			commands.CMDProcess(),
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		// Before may have failed before Logger appeared.
		if modules.Logger != nil {
			modules.Logger.Error("execution failed", "error", err)
		} else {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
		return 1
	}
	return 0
}
