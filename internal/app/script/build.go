package script

import (
	"context"
	"fmt"

	"github.com/Galdoba/lazyam3/internal/domain/script"
)

// Build is the use case that builds a rendering script from the config and
// materializes it as a usable artifact via the Renderer.
func Build(ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sc, err := script.BuildScript(config)
	if err != nil {
		return fmt.Errorf("build script: %w", err)
	}
	return renderer.Render(sc)
}
