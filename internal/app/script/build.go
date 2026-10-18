package script

import (
	"context"

	"github.com/Galdoba/lazyam3/internal/domain/script"
)

// Build is the use case that accepts domain inputs and produces a script text.
func Build(ctx context.Context, config script.ScriptConfig, renderer script.Renderer) error {
	text := script.Text("// TODO: generate rendering script")
	return renderer.Write(config.Destination, text)
}
