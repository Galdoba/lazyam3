package script

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	scriptinfra "github.com/Galdoba/lazyam3/internal/infra/script"
)

// TestBuildAndExecute_RealInputs is a miniature live test: it feeds the
// synthetic fixture through the buffer convention and executes the rendered
// script end to end with bash + ffmpeg, then checks the whole artifact tree.
func TestBuildAndExecute_RealInputs(t *testing.T) {
	for _, bin := range []string{"ffmpeg", "bash"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skip(bin + " not available")
		}
	}
	cfg, srcDir, envRoot := realConfig(t)

	// Mirror the work-directory layout the pipeline creates at startup.
	for _, dir := range []string{"buffer", "in_progress", "done/bash", "notifications", "target", "scripts"} {
		if err := os.MkdirAll(filepath.Join(envRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Feeder: the unit's files must sit in buffer when the script runs.
	for _, name := range cfg.Filenames {
		data, err := os.ReadFile(filepath.Join(srcDir, name))
		if err != nil {
			t.Fatalf("read fixture %q: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(envRoot, "buffer", name), data, 0o644); err != nil {
			t.Fatalf("place %q in buffer: %v", name, err)
		}
	}

	if err := Build(context.Background(), cfg, scriptinfra.New()); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	out, err := exec.Command("bash", cfg.Destination).CombinedOutput()
	if err != nil {
		t.Fatalf("script execution failed: %v\n%s", err, out)
	}

	targetDir := filepath.Join(envRoot, "target", "Arliss_s01")
	expected := []string{
		filepath.Join(targetDir, "Arliss_s01e06_HD.mp4"),
		filepath.Join(targetDir, "Arliss_s01e06.ready"),
		filepath.Join(envRoot, "notifications", "Arliss_s01e06.ready"),
		filepath.Join(envRoot, "done", "12254.mp4"),
		filepath.Join(envRoot, "done", "12254-rus_Russkii.srt"),
		filepath.Join(envRoot, "done", "12254-rus_Russkii_Forsirovannye.srt"),
		filepath.Join(envRoot, "done", "bash", "12254.sh"),
	}
	for _, p := range expected {
		fi, err := os.Stat(p)
		if err != nil {
			t.Errorf("expected artifact missing: %s (%v)", p, err)
			continue
		}
		if fi.IsDir() {
			t.Errorf("expected file, got directory: %s", p)
		}
	}

	// Rendered video must be non-empty.
	if fi, err := os.Stat(filepath.Join(targetDir, "Arliss_s01e06_HD.mp4")); err == nil && fi.Size() == 0 {
		t.Error("rendered video is empty")
	}

	// One ALAC output per audio stream (stereo + mono in the fixture).
	matches, err := filepath.Glob(filepath.Join(targetDir, "Arliss_s01e06_HD_AUDIO_*.m4a"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Errorf("audio outputs = %v, want 2 files", matches)
	}

	// Buffer must be drained; in_progress must be empty.
	for _, dir := range []string{"buffer", "in_progress"} {
		entries, err := os.ReadDir(filepath.Join(envRoot, dir))
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		if len(entries) != 0 {
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Errorf("%s not drained: %v", dir, names)
		}
	}

	// The script must have archived itself.
	if _, err := os.Stat(cfg.Destination); !os.IsNotExist(err) {
		t.Errorf("script still at %s after execution (err=%v)", cfg.Destination, err)
	}
	if !strings.HasSuffix(cfg.Destination, filepath.Join("scripts", "12254.sh")) {
		t.Errorf("unexpected destination layout: %s", cfg.Destination)
	}
}
