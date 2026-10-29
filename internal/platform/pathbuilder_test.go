package platform

import (
	"path/filepath"
	"testing"
)

func TestPathBuilder_native(t *testing.T) {
	b := NewPathBuilder("/root")
	got := b.native("dir", "file.mp4")
	if got != filepath.Join("/root", "dir", "file.mp4") {
		t.Errorf("native() = %q, want %q", got, filepath.Join("/root", "dir", "file.mp4"))
	}
}

func TestPathBuilder_canonical(t *testing.T) {
	b := NewPathBuilder("/root")
	got := b.canonical("dir", "file.mp4")
	if got != "/root/dir/file.mp4" {
		t.Errorf("canonical() = %q, want %q", got, "/root/dir/file.mp4")
	}
}

func TestPathBuilder_canonicalUsesForwardSlashes(t *testing.T) {
	b := NewPathBuilder("/root")
	got := b.canonical("a", "b", "c")
	for _, ch := range got {
		if ch == '\\' {
			t.Errorf("canonical() contains backslash: %q", got)
		}
	}
}
