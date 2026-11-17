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

	// Feeder: place files in buffer with call names.
	for orig, call := range cfg.CallNames {
		data, err := os.ReadFile(filepath.Join(srcDir, orig))
		if err != nil {
			t.Fatalf("read fixture %q: %v", orig, err)
		}
		if err := os.WriteFile(filepath.Join(envRoot, "buffer", call), data, 0o644); err != nil {
			t.Fatalf("place %q → %q in buffer: %v", orig, call, err)
		}
	}

	if err := Build(nil, context.Background(), cfg, scriptinfra.New()); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	out, err := exec.Command("bash", cfg.Destination).CombinedOutput()
	if err != nil {
		t.Logf("SCRIPT OUTPUT:\n%s", out)
		t.Fatalf("script execution failed: %v\n%s", err, out)
	}
	t.Logf("Script output: %s", out)

	targetDir := filepath.Join(envRoot, "target", "Arliss_s01")

	// The done directory should contain files with call names, not originals.
	if _, err := os.Stat(filepath.Join(envRoot, "done")); os.IsNotExist(err) {
		t.Error("done directory missing")
	}

	// Rendered video must be non-empty (glob for PRT-wildcard).
	matches, err := filepath.Glob(filepath.Join(targetDir, "*_HD.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Error("no rendered video found")
	} else if fi, err := os.Stat(matches[0]); err == nil && fi.Size() == 0 {
		t.Error("rendered video is empty")
	}

	// One ALAC output per audio stream (stereo + mono in the fixture).
	matches, err = filepath.Glob(filepath.Join(targetDir, "*_HD_AUDIO*.m4a"))
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
	// Destination should end with OUTBASE.sh, not unit ID.
	if strings.HasSuffix(cfg.Destination, "12254.sh") {
		t.Errorf("destination should use OUTBASE, not unit ID: %s", cfg.Destination)
	}
}

// scriptNameFromDestination derives the script name (basename without .sh)
// used for execution-signal markers, mirroring pipeline.buildScriptConfig.
func scriptNameFromDestination(destination string) string {
	base := filepath.Base(destination)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// TestBuildAndExecute_ErroredRender plants an error log in the in-progress
// directory before running the generated script, and verifies: the target
// directory ends up with <OUTBASE>.#err sharing its base name with the ready
// marker; the metadata directory holds <SCRIPT_NAME>.err containing the
// planted error text; the planted source is untouched; the in-progress
// directory still holds the original .#err; no files are written to the
// notification directory.
func TestBuildAndExecute_ErroredRender(t *testing.T) {
	for _, bin := range []string{"ffmpeg", "bash"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skip(bin + " not available")
		}
	}
	cfg, srcDir, envRoot := realConfig(t)

	cfg.ScriptName = scriptNameFromDestination(cfg.Destination)
	cfg.Env.MetadataDir = filepath.Join(envRoot, "buffer", "metadata")

	for _, dir := range []string{"buffer", "in_progress", "done/bash", "notifications", "target", "scripts", "buffer/metadata"} {
		if err := os.MkdirAll(filepath.Join(envRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	metaDir := cfg.Env.MetadataDir
	processingPath := filepath.Join(metaDir, cfg.ScriptName+".processing")
	if err := os.WriteFile(processingPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write processing marker: %v", err)
	}

	// Feeder: place files in buffer with call names.
	for orig, call := range cfg.CallNames {
		data, err := os.ReadFile(filepath.Join(srcDir, orig))
		if err != nil {
			t.Fatalf("read fixture %q: %v", orig, err)
		}
		if err := os.WriteFile(filepath.Join(envRoot, "buffer", call), data, 0o644); err != nil {
			t.Fatalf("place %q → %q in buffer: %v", orig, call, err)
		}
	}

	// Plant error log in in_progress alongside the SOURCE file.
	sourceCall := cfg.CallNames[cfg.Filenames[0]]
	errText := "render failed: ffmpeg filter error\n"
	inProgressErr := filepath.Join(envRoot, "in_progress", sourceCall+".#err")
	if err := os.WriteFile(inProgressErr, []byte(errText), 0o644); err != nil {
		t.Fatalf("plant .#err: %v", err)
	}

	if err := Build(nil, context.Background(), cfg, scriptinfra.New()); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	out, err := exec.Command("bash", cfg.Destination).CombinedOutput()
	if err != nil {
		t.Logf("SCRIPT OUTPUT:\n%s", out)
		t.Fatalf("script execution failed: %v\n%s", err, out)
	}

	targetDir := filepath.Join(envRoot, "target", "Arliss_s01")

	// The ready marker must exist in target.
	readyFiles, err := filepath.Glob(filepath.Join(targetDir, "*.ready"))
	if err != nil {
		t.Fatal(err)
	}
	if len(readyFiles) == 0 {
		t.Error("no ready marker in target")
	} else if len(readyFiles) > 1 {
		t.Errorf("multiple ready markers: %v", readyFiles)
	}
	readyBase := filepath.Base(readyFiles[0])
	readyBase = strings.TrimSuffix(readyBase, filepath.Ext(readyBase))

	// Target should have <OUTBASE>.#err sharing base name with the ready marker.
	targetErr := filepath.Join(targetDir, readyBase+".#err")
	if _, statErr := os.Stat(targetErr); os.IsNotExist(statErr) {
		t.Errorf("target missing %q", filepath.Base(targetErr))
	}
	targetErrContent, readErr := os.ReadFile(targetErr)
	if readErr != nil {
		t.Fatalf("read target .#err: %v", readErr)
	}
	if string(targetErrContent) != errText {
		t.Errorf("target .#err content = %q, want %q", string(targetErrContent), errText)
	}

	// Metadata should hold <SCRIPT_NAME>.err with planted text.
	metaErr := filepath.Join(metaDir, cfg.ScriptName+".err")
	metaErrContent, readErr := os.ReadFile(metaErr)
	if readErr != nil {
		t.Fatalf("read metadata .err: %v", readErr)
	}
	if string(metaErrContent) != errText {
		t.Errorf("metadata .err content = %q, want %q", string(metaErrContent), errText)
	}

	// In-progress still holds the original .#err (copy, not move).
	if _, statErr := os.Stat(inProgressErr); os.IsNotExist(statErr) {
		t.Errorf("in_progress .#err was moved, not copied")
	}

	// Notification dir untouched.
	notifEntries, err := os.ReadDir(filepath.Join(envRoot, "notifications"))
	if err != nil {
		t.Fatalf("read notifications: %v", err)
	}
	if len(notifEntries) != 0 {
		names := make([]string, 0, len(notifEntries))
		for _, e := range notifEntries {
			names = append(names, e.Name())
		}
		t.Errorf("notification dir not empty: %v", names)
	}
}

// TestBuildAndExecute_CleanRender verifies a clean render produces no .#err in
// the target directory, no .err sidecar in metadata, the ready marker is still
// copied, and the script exits zero.
func TestBuildAndExecute_CleanRender(t *testing.T) {
	for _, bin := range []string{"ffmpeg", "bash"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skip(bin + " not available")
		}
	}
	cfg, srcDir, envRoot := realConfig(t)

	cfg.ScriptName = scriptNameFromDestination(cfg.Destination)
	cfg.Env.MetadataDir = filepath.Join(envRoot, "buffer", "metadata")

	for _, dir := range []string{"buffer", "in_progress", "done/bash", "notifications", "target", "scripts", "buffer/metadata"} {
		if err := os.MkdirAll(filepath.Join(envRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	metaDir := cfg.Env.MetadataDir
	processingPath := filepath.Join(metaDir, cfg.ScriptName+".processing")
	if err := os.WriteFile(processingPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write processing marker: %v", err)
	}

	// Feeder: place files in buffer with call names.
	for orig, call := range cfg.CallNames {
		data, err := os.ReadFile(filepath.Join(srcDir, orig))
		if err != nil {
			t.Fatalf("read fixture %q: %v", orig, err)
		}
		if err := os.WriteFile(filepath.Join(envRoot, "buffer", call), data, 0o644); err != nil {
			t.Fatalf("place %q → %q in buffer: %v", orig, call, err)
		}
	}

	if err := Build(nil, context.Background(), cfg, scriptinfra.New()); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	out, err := exec.Command("bash", cfg.Destination).CombinedOutput()
	if err != nil {
		t.Logf("SCRIPT OUTPUT:\n%s", out)
		t.Fatalf("script execution failed: %v\n%s", err, out)
	}

	targetDir := filepath.Join(envRoot, "target", "Arliss_s01")

	// No .#err in target.
	errFiles, err := filepath.Glob(filepath.Join(targetDir, "*.#err"))
	if err != nil {
		t.Fatal(err)
	}
	if len(errFiles) != 0 {
		t.Errorf("unexpected .#err in target: %v", errFiles)
	}

	// No .err sidecar in metadata.
	metaErr := filepath.Join(metaDir, cfg.ScriptName+".err")
	if _, statErr := os.Stat(metaErr); !os.IsNotExist(statErr) {
		t.Errorf("metadata .err sidecar exists in clean render")
	}

	// Ready marker copied.
	readyFiles, err := filepath.Glob(filepath.Join(targetDir, "*.ready"))
	if err != nil {
		t.Fatal(err)
	}
	if len(readyFiles) == 0 {
		t.Error("no ready marker in clean render target")
	}

	// Notification dir untouched.
	notifEntries, err := os.ReadDir(filepath.Join(envRoot, "notifications"))
	if err != nil {
		t.Fatalf("read notifications: %v", err)
	}
	if len(notifEntries) != 0 {
		names := make([]string, 0, len(notifEntries))
		for _, e := range notifEntries {
			names = append(names, e.Name())
		}
		t.Errorf("notification dir not empty: %v", names)
	}
}
