// Package script implements the domain script.Renderer port by materializing
// built scripts as executable files on disk.
package script

import (
	"fmt"
	"os"
	"path/filepath"

	dscript "github.com/Galdoba/lazyam3/internal/domain/script"
)

// DefaultMode is the file mode used for rendered scripts unless overridden.
const DefaultMode os.FileMode = 0o755

// Renderer materializes a built script as a usable file artifact.
type Renderer struct {
	mode os.FileMode
}

// Option configures a Renderer.
type Option func(*Renderer)

// WithMode sets the file mode for written scripts.
func WithMode(m os.FileMode) Option {
	return func(r *Renderer) { r.mode = m }
}

// New returns a Renderer with the given options.
//log=debug msg="script renderer created with mode=%v", mode
func New(opts ...Option) *Renderer {
	r := &Renderer{mode: DefaultMode}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Render writes sc.Content to sc.Destination, creating parent directories
// as needed and enforcing the configured file mode regardless of umask.
//log=info msg="writing script to: %s", sc.Destination
func (r *Renderer) Render(sc dscript.Script) error {
	dir := filepath.Dir(sc.Destination)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		//log=error msg="failed to create script directory %q: %v", dir, err
		return fmt.Errorf("create script directory %q: %w", dir, err)
	}
	if err := os.WriteFile(sc.Destination, []byte(sc.Content), r.mode); err != nil {
		//log=error msg="failed to write script %q: %v", sc.Destination, err
		return fmt.Errorf("write script %q: %w", sc.Destination, err)
	}
	if err := os.Chmod(sc.Destination, r.mode); err != nil {
		//log=error msg="failed to set mode on %q: %v", sc.Destination, err
		return fmt.Errorf("set script mode %q: %w", sc.Destination, err)
	}
	return nil
}
