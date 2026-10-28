package script

import (
	"context"
	"fmt"

	"github.com/Galdoba/lazyam3/internal/domain/script"
	"github.com/Galdoba/lazyam3/internal/infra/logging"
)

// Build is the use case that builds a rendering script from the config and
// materializes it as a usable artifact via the Renderer.
func Build(logger *logging.Logger, ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error {
	if logger != nil {
		logger.Info("building rendering script")
	}
	if err := ctx.Err(); err != nil {
		if logger != nil {
			logger.Warn("context cancelled during script build", "error", err)
		}
		return err
	}
	sc, err := script.BuildScript(config)
	if err != nil {
		return fmt.Errorf("build script: %w", err)
	}
	if logger != nil {
		logger.Info("writing rendered script to destination", "path", config.Destination)
	}
	return renderer.Render(sc)
}
