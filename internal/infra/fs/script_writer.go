package fs

import (
	"os"

	"github.com/Galdoba/lazyam3/internal/domain/script"
)

// ScriptWriter implements script.Renderer by writing text to a file.
type ScriptWriter struct{}

// Write writes the script content to the destination path.
func (w *ScriptWriter) Write(dest string, content script.Text) error {
	return os.WriteFile(dest, []byte(content), 0o755)
}
