package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/lazyam3/internal/app"
	"github.com/Galdoba/lazyam3/internal/presentation/tui"
	"github.com/urfave/cli/v3"
)

// CMD3 — команда с интерактивным TUI.
//
// Показывает, как совмещать presentation-слой с шаблоном:
//   - TUI запускается как отдельная программа внутри Action;
//   - логирование идёт до tui.Run и после него, но не внутри Update/View
//     (иначе вывод логгера врежется в отрисовку bubbletea);
//   - store поднимается лениво и только если пользователь выбрал элемент.
func CMD3(modules *app.AppModules) *cli.Command {
	return &cli.Command{
		Name:        "cmd3",
		Usage:       "interactive selection via TUI",
		Description: "Opens a TUI, lets the user pick an item, saves the choice.",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "item",
				Aliases: []string{"i"},
				Usage:   "item to show in the list (can be repeated); if empty, defaults are used",
			},
		},
		Action: cmd3Action(modules),
	}
}

func cmd3Action(modules *app.AppModules) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		logger := modules.Logger.With("command", "CMD3")
		logger.Info("cmd3 started")

		// 1. Собираем список элементов: флаги → позиционные аргументы → дефолт.
		items := cmd.StringSlice("item")
		if len(items) == 0 {
			items = cmd.Args().Slice()
		}
		if len(items) == 0 {
			items = []string{"alpha", "beta", "gamma"}
		}

		logger.Debug("tui items prepared", "count", len(items))

		// 2. Запускаем TUI как отдельную программу.
		//    Внутри tui.Run логировать нельзя — сломает отрисовку.
		final, err := tui.Run(tui.NewModel("choose an item", items))
		if err != nil {
			return logger.Errorf("tui failed: %w", err)
		}

		// 3. Управление вернулось — можно логировать снова.
		if final.Quit() {
			logger.Info("user cancelled")
			return nil
		}

		selected := final.Selected()
		if selected == "" {
			return logger.Errorf("tui returned without a selected item")
		}

		logger.Info("user selected", "item", selected, "index", final.Cursor())

		// 4. Store — ленивый. Откроется только здесь, потому что
		//    команда действительно хочет сохранить выбор.
		store, err := modules.Store()
		if err != nil {
			return logger.Errorf("failed to open store: %w", err)
		}
		store.Set("last_selected", selected)

		// 5. Результат — в stdout.
		fmt.Fprintf(cmd.Root().Writer, "%s\n", selected)

		return nil
	}
}
