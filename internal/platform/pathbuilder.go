package platform

import (
	"path"
	"path/filepath"
)

// PathBuilder constructs paths in a platform-agnostic way.
// native() returns host-OS-native separators; canonical() returns
// forward-slash strings for metadata markers and cross-platform use.
type PathBuilder struct {
	root string
}

// NewPathBuilder returns a PathBuilder rooted at root.
func NewPathBuilder(root string) PathBuilder {
	return PathBuilder{root: root}
}

// native joins path components using the host OS separator.
func (b PathBuilder) native(components ...string) string {
	return filepath.Join(b.root, filepath.Join(components...))
}

// canonical joins path components using forward slashes, regardless of OS.
func (b PathBuilder) canonical(components ...string) string {
	return path.Join(b.root, path.Join(components...))
}
