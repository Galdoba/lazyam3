package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/lazyam3/internal/app"
	"github.com/urfave/cli/v3"
)

// CMD2 — команда, использующая только in-memory cache.
// Демонстрирует, что store и другие ленивые модули поднимаются
// строго по требованию: здесь modules.Store() не вызывается,
// поэтому файл хранилища не откроется.
func CMD2(modules *app.AppModules) *cli.Command {

	return &cli.Command{
		Name:        "cmd2",
		Usage:       "example command that uses only cache",
		Description: "Writes a marker to the in-memory cache and reports its state.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "marker",
				Aliases: []string{"m"},
				Usage:   "value to store under the 'last_cmd' key",
				Value:   "cmd2",
			},
		},
		Action: cmd2Action(modules),
	}
}

func cmd2Action(modules *app.AppModules) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		logger := modules.Logger.With("command", "CMD2")
		// 1. Логируем старт.
		logger.Info("cmd2 started", "args", cmd.Args().Slice())

		// 2. Читаем флаг.
		marker := cmd.String("marker")
		if marker == "" {
			return logger.Errorf("marker must not be empty")
		}

		// 3. Cache — ленивый аксессор. При первом вызове создаётся,
		//    при последующих возвращается тот же экземпляр.
		c := modules.Cache()
		c.Set("last_cmd", marker)

		// 4. Читаем обратно — демонстрируем, что значение доступно.
		value, ok := c.Get("last_cmd")
		if !ok {
			return logger.Errorf("cache write failed: key not found after Set")
		}

		logger.Info("cache updated",
			"key", "last_cmd",
			"value", value,
			"cache_size", c.Len(),
		)

		// 5. Результат — в stdout, не в лог.
		fmt.Fprintf(cmd.Root().Writer, "last_cmd = %v\n", value)

		return nil
	}
}
