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
	got := b.Canonical("dir", "file.mp4")
	if got != "/root/dir/file.mp4" {
		t.Errorf("Canonical() = %q, want %q", got, "/root/dir/file.mp4")
	}
}

func TestPathBuilder_canonicalUsesForwardSlashes(t *testing.T) {
	b := NewPathBuilder("/root")
	got := b.Canonical("a", "b", "c")
	for _, ch := range got {
		if ch == '\\' {
			t.Errorf("Canonical() contains backslash: %q", got)
		}
	}
}

func TestPathBuilder_canonicalEmptyRoot(t *testing.T) {
	b := NewPathBuilder("")
	got := b.Canonical("dir", "file.mp4")
	if got != "dir/file.mp4" {
		t.Errorf("Canonical() with empty root = %q, want %q", got, "dir/file.mp4")
	}
}

func TestPathBuilder_canonicalWindowsDriveLetter(t *testing.T) {
	b := NewPathBuilder("C:/Users/test")
	got := b.Canonical("dir", "file.mp4")
	if got != "C:/Users/test/dir/file.mp4" {
		t.Errorf("Canonical() with drive letter = %q, want %q", got, "C:/Users/test/dir/file.mp4")
	}
}

func TestPathBuilder_canonicalUncPath(t *testing.T) {
	b := NewPathBuilder("//server/share/root")
	got := b.Canonical("dir", "file.mp4")
	// UNC paths must preserve the leading // prefix.
	if got != "//server/share/root/dir/file.mp4" {
		t.Errorf("Canonical() with UNC path = %q, want %q", got, "//server/share/root/dir/file.mp4")
	}
}

func TestPathBuilder_canonicalMixedSeparators(t *testing.T) {
	b := NewPathBuilder("/root")
	got := b.Canonical("a/b", "c/d")
	if got != "/root/a/b/c/d" {
		t.Errorf("Canonical() with mixed separators = %q, want %q", got, "/root/a/b/c/d")
	}
}
