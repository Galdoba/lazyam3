package script

import (
	"context"
	"fmt"

	"github.com/Galdoba/lazyam3/internal/domain/script"
)

// Build is the use case that builds a rendering script from the config and
// materializes it as a usable artifact via the Renderer.
//log=info msg="building rendering script"
func Build(ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error {
	if err := ctx.Err(); err != nil {
		//log=warn msg="context cancelled during script build: %v", err
		return err
	}
	sc, err := script.BuildScript(config)
	if err != nil {
		//log=error msg="build script failed: %v", err
		return fmt.Errorf("build script: %w", err)
	}
	//log=info msg="writing rendered script to %s", config.Destination
	return renderer.Render(sc)
}
