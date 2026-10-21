package script

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	dscript "github.com/Galdoba/lazyam3/internal/domain/script"
)

func TestRenderer_Render(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out", "nested", "12094.sh")
	r := New()
	err := r.Render(dscript.Script{
		Destination: dest,
		Content:     "#!/bin/bash\necho ok\n",
	})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read rendered script: %v", err)
	}
	if !strings.HasPrefix(string(data), "#!/bin/bash") {
		t.Errorf("content does not start with shebang: %q", data)
	}

	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat rendered script: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o755 {
		t.Errorf("mode = %04o, want 0755", got)
	}
}

func TestRenderer_WithMode(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "s.sh")
	err := New(WithMode(0o600)).Render(dscript.Script{
		Destination: dest,
		Content:     "#!/bin/bash\n",
	})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat rendered script: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

func TestRenderer_Render_BadDestination(t *testing.T) {
	// A regular file exists where a directory is required.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := New().Render(dscript.Script{
		Destination: filepath.Join(blocker, "sub", "s.sh"),
		Content:     "#!/bin/bash\n",
	})
	if err == nil {
		t.Fatal("expected error for destination under a regular file, got nil")
	}
	if !strings.Contains(err.Error(), "create script directory") {
		t.Errorf("error = %q, want it to mention create script directory", err)
	}
}
