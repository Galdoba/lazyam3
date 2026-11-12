package platform

import (
	"path"
	"path/filepath"
	"strings"
)

// PathBuilder constructs paths in a platform-agnostic way.
// native() returns host-OS-native separators; canonical() returns
// forward-slash strings for metadata markers and cross-platform use.
type PathBuilder struct {
	root string
}

// IsEmpty returns true when the PathBuilder has no root set.
func (b PathBuilder) IsEmpty() bool {
	return b.root == ""
}

// NewPathBuilder returns a PathBuilder rooted at root.
func NewPathBuilder(root string) PathBuilder {
	return PathBuilder{root: root}
}

// native joins path components using the host OS separator.
func (b PathBuilder) native(components ...string) string {
	return filepath.Join(b.root, filepath.Join(components...))
}

// Canonical joins path components using forward slashes, regardless of OS,
// preserving a leading "//" UNC prefix on the root.
func (b PathBuilder) Canonical(components ...string) string {
	isUnc := strings.HasPrefix(b.root, "//")
	got := path.Join(b.root, path.Join(components...))
	if isUnc && strings.HasPrefix(got, "/") && !strings.HasPrefix(got, "//") {
		return "//" + got[1:]
	}
	return got
}
