package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Galdoba/lazyam3/internal/app"
	"github.com/Galdoba/lazyam3/internal/app/commands"
	"github.com/urfave/cli/v3"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Пустой контейнер. Config и Logger наполнятся в Root.Before
	// после парсинга флагов. Остальные модули — ленивые аксессоры.
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
			// Здесь флаги корня уже распарсены, --config доступен.
			// Создаём ровно то, что нужно для логирования ошибок запуска:
			// конфиг и логгер. Всё остальное — по требованию из команд.
			if err := modules.Init(cmd.String("config")); err != nil {
				return ctx, err
			}
			return ctx, nil
		},
		After: func(ctx context.Context, cmd *cli.Command) error {
			// Закрывает всё, что было реально создано ленивыми аксессорами.
			return modules.Shutdown()
		},
		Commands: []*cli.Command{
			commands.CMD1(modules),
			commands.CMD2(modules),
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		// Before мог упасть до того, как Logger появился.
		if modules.Logger != nil {
			modules.Logger.Error("execution failed", "error", err)
		} else {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
		return 1
	}
	return 0
}
