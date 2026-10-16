package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Galdoba/lazyam3/internal/app"
	"github.com/urfave/cli/v3"
)

// CMD1 — образец команды шаблона.
//
// Что делает этот файл:
//  1. Собирает *cli.Command с флагами и описанием.
//  2. Создаёт scoped-логгер с атрибутом "command"="CMD1".
//  3. Передаёт модули и логгер в Action через замыкание.
//  4. В Action лениво поднимает нужные модули (Cache, Store).
//
// При копировании под новую команду:
//   - переименуй CMD1 → NewFooCommand (или оставь стиль CMDx);
//   - задай Name, Usage, Description, Flags;
//   - перепиши тело cmd1Action;
//   - удали неиспользуемые вызовы modules.Cache()/modules.Store().
func CMD1(modules *app.AppModules) *cli.Command {

	return &cli.Command{
		Name:        "cmd1",
		Usage:       "example command that uses cache and store",
		Description: "Demonstrates the template's command shape: flags, args, lazy modules.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "key to write to store",
				Value:   "cmd1_count",
			},
			&cli.IntFlag{
				Name:  "delta",
				Usage: "amount to add to the counter",
				Value: 1,
			},
		},
		Action: cmd1Action(modules),
		// Остальные поля cli.Command (Before, After, Commands и т.д.)
		// добавляй по мере необходимости. Полный список полей v3 —
		// см. cmd2.go / cmd3.go, где они закомментированы как справка.
	}
}

func cmd1Action(modules *app.AppModules) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		logger := modules.Logger.With("command", "CMD1")
		// 1. Логируем факт запуска и аргументы.
		args := cmd.Args().Slice()
		logger.Info("cmd1 started", "args", args)

		// 2. Читаем флаги.
		key := cmd.String("key")
		delta := cmd.Int("delta")
		if delta <= 0 {
			return logger.Errorf("delta must be positive, got %d", delta)
		}

		// 3. Cache — дешёвый, создаётся при первом вызове.
		//    Если бы команда его не трогала, он бы не создался.
		c := modules.Cache()
		c.Set("last_cmd", "cmd1")

		// 4. Store — ленивый, файл откроется здесь при первом обращении.
		store, err := modules.Store()
		if err != nil {
			return logger.Errorf("failed to open store: %w", err)
		}

		// 5. Бизнес-логика. JSON-числа после Unmarshal — float64.
		raw, _ := store.Get(key)
		prev, _ := raw.(float64)
		next := prev + float64(delta)
		store.Set(key, next)

		logger.Info("counter updated",
			"key", key,
			"prev", prev,
			"next", next,
		)

		// 6. Печатаем результат в stdout — это вывод команды, не лог.
		fmt.Fprintf(cmd.Root().Writer, "%s = %s\n",
			key, strconv.FormatFloat(next, 'f', -1, 64))

		return nil
	}
}
