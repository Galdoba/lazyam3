package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultCFG()

	if cfg.Paths.GeneratorInput() != "/tmp/lazyam3/input" {
		t.Errorf("Paths.Generator.InputDirectory = %q, want %q", cfg.Paths.GeneratorInput(), "/tmp/lazyam3/input")
	}
	if cfg.Paths.GeneratorOutput() != "/tmp/lazyam3/output" {
		t.Errorf("Paths.Generator.OutputDirectory = %q, want %q", cfg.Paths.GeneratorOutput(), "/tmp/lazyam3/output")
	}
	if cfg.Paths.GeneratorMetadata() != "/tmp/lazyam3/metadata.json" {
		t.Errorf("Paths.Generator.MetadataPath = %q, want %q", cfg.Paths.GeneratorMetadata(), "/tmp/lazyam3/metadata.json")
	}
	if cfg.Logger.Level != "info" {
		t.Errorf("Logger.Level = %q, want %q", cfg.Logger.Level, "info")
	}
	if cfg.Logger.Terminal != true {
		t.Errorf("Logger.Terminal = %v, want %v", cfg.Logger.Terminal, true)
	}
	if cfg.Storage.Path != "/path/to/file.json" {
		t.Errorf("Storage.Path = %q, want %q", cfg.Storage.Path, "/path/to/file.json")
	}
	if cfg.ScriptGeneration.Hardsub.Tag != "Forsirovannye" {
		t.Errorf("ScriptGeneration.Hardsub.Tag = %q, want %q", cfg.ScriptGeneration.Hardsub.Tag, "Forsirovannye")
	}
}

func TestDefaultConfig_FfprobeBinary(t *testing.T) {
	cfg := defaultCFG()
	if cfg.Paths.GeneratorFFprobe() != "ffprobe" {
		t.Errorf("Paths.Generator.FfprobeBinary = %q, want %q", cfg.Paths.GeneratorFFprobe(), "ffprobe")
	}
}

func TestDefaultConfig_LoggerPath(t *testing.T) {
	cfg := defaultCFG()
	if cfg.Logger.Path != "/path/to/file.log" {
		t.Errorf("Logger.Path = %q, want %q", cfg.Logger.Path, "/path/to/file.log")
	}
}

func TestDefaultConfig_PassesValidation(t *testing.T) {
	if err := defaultCFG().Validate(); err != nil {
		t.Errorf("defaultCFG().Validate() = %v, want nil", err)
	}
}

func TestConfig_ValidateBadLevel(t *testing.T) {
	tests := []string{"bogus", "WARNING", "infoo", ""}
	for _, lvl := range tests {
		cfg := defaultCFG()
		cfg.Logger.Level = lvl
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() with level %q: want error, got nil", lvl)
		}
	}
}

func TestConfig_ValidateEmptyGeneratorPath(t *testing.T) {
	cfg := defaultCFG()
	for _, field := range []string{"input_directory", "output_directory", "metadata_path"} {
		switch field {
		case "input_directory":
			cfg.Paths.Generator.InputDirectory = ""
		case "output_directory":
			cfg.Paths.Generator.OutputDirectory = ""
		case "metadata_path":
			cfg.Paths.Generator.MetadataPath = ""
		}
		if err := cfg.Validate(); err == nil {
			t.Errorf("Validate() with empty %s: want error, got nil", field)
		}
	}
}

func TestDefaultConfig_ExecutorMapSeeded(t *testing.T) {
	cfg := defaultCFG()
	if cfg.Paths.Executor == nil {
		t.Fatal("Paths.Executor should not be nil")
	}
	tests := map[string]string{
		"target":        "/tmp/lazyam3/output/target",
		"buffer":        "/tmp/lazyam3/output",
		"in_progress":   "/tmp/lazyam3/output",
		"done":          "/tmp/lazyam3/output",
		"notifications": "/tmp/lazyam3/output",
		"archive":       "/tmp/lazyam3/output/archive",
	}
	if len(cfg.Paths.Executor) != len(tests) {
		t.Errorf("Paths.Executor has %d entries, want %d", len(cfg.Paths.Executor), len(tests))
	}
	for key, want := range tests {
		if got := cfg.Paths.ExecutorPath(key); got != want {
			t.Errorf("Paths.ExecutorPath(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestLoad_ValidateRejectsBadLevel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAZYAM3_CONFIG", "")
	t.Setenv("LAZYAM3_CONFIG_PATH", "")
	toml := `[paths.generator]
input_directory = '/in'
output_directory = '/out'
metadata_path = '/meta.json'

[logger]
level = 'bogus'
`
	path := writeConfig(t, toml)
	if _, err := Load(path); err == nil {
		t.Fatal("Load() with invalid logger.level: want error, got nil")
	}
}

func TestLoad_ValidateRejectsEmptyGeneratorPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAZYAM3_CONFIG", "")
	t.Setenv("LAZYAM3_CONFIG_PATH", "")
	toml := `[paths.generator]
input_directory = ''
output_directory = '/out'
metadata_path = '/meta.json'
`
	path := writeConfig(t, toml)
	if _, err := Load(path); err == nil {
		t.Fatal("Load() with empty input_directory: want error, got nil")
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTOML_ParseExecutorPaths(t *testing.T) {
	toml := `[paths]
[paths.generator]
input_directory = '\\192.168.31.4\buffer\IN\examplesCopy\'
output_directory = '\\192.168.31.4\buffer\IN\'
metadata_path = '\\192.168.31.4\buffer\IN\@AMEDIA_IN\metadata.json'

[paths.executor]
archive = "/home/pemaltynov/IN/_DONE/bash/"
buffer = "/home/pemaltynov/IN/"
done = "/home/pemaltynov/IN/_DONE/"
in_progress = "/home/pemaltynov/IN/_IN_PROGRESS/"
target = "/mnt/pemaltynov/ROOT/EDIT/_amedia/"

[logger]
level = 'debug'
path = 'c:\Users\pemaltynov\.config\lazyam3\lazyam3.log'
terminal = true

[storage]
path = 'c:\Users\pemaltynov\.config\lazyam3\storage.json'

[script_generation]
[script_generation.hardsub]
tag = 'Forsirovannye'
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.toml")
	if err := os.WriteFile(path, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Paths.Executor == nil {
		t.Fatal("Paths.Executor is nil")
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"buffer", "/home/pemaltynov/IN/"},
		{"in_progress", "/home/pemaltynov/IN/_IN_PROGRESS/"},
		{"done", "/home/pemaltynov/IN/_DONE/"},
		{"target", "/mnt/pemaltynov/ROOT/EDIT/_amedia/"},
		{"archive", "/home/pemaltynov/IN/_DONE/bash/"},
	}

	for _, tc := range tests {
		got := cfg.Paths.ExecutorPath(tc.key)
		if got != tc.expected {
			t.Errorf("Paths.ExecutorPath(%q) = %q, want %q", tc.key, got, tc.expected)
		}
	}
}

func TestScriptGeneration_HardsubTag(t *testing.T) {
	cfg := defaultCFG()
	if cfg.ScriptGeneration.Hardsub.Tag != "Forsirovannye" {
		t.Errorf("ScriptGeneration.Hardsub.Tag = %q, want %q", cfg.ScriptGeneration.Hardsub.Tag, "Forsirovannye")
	}
}

func TestScriptGeneration_FfmpegBinary(t *testing.T) {
	cfg := defaultCFG()
	if cfg.ScriptGeneration.FfmpegBinary != "ffmpeg" {
		t.Errorf("ScriptGeneration.FfmpegBinary = %q, want %q", cfg.ScriptGeneration.FfmpegBinary, "ffmpeg")
	}
}

func TestScriptGeneration_TOMLNewFormat(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAZYAM3_CONFIG", "")
	t.Setenv("LAZYAM3_CONFIG_PATH", "")
	toml := `[paths.generator]
input_directory = '/in'
output_directory = '/out'
metadata_path = '/meta.json'

[script_generation]
ffmpeg_binary = '/opt/bin/ffmpeg'

[script_generation.hardsub]
tag = 'NewFormatTag'
`
	path := writeConfig(t, toml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.ScriptGeneration.Hardsub.Tag != "NewFormatTag" {
		t.Errorf("ScriptGeneration.Hardsub.Tag = %q, want %q", cfg.ScriptGeneration.Hardsub.Tag, "NewFormatTag")
	}
	if cfg.ScriptGeneration.FfmpegBinary != "/opt/bin/ffmpeg" {
		t.Errorf("ScriptGeneration.FfmpegBinary = %q, want %q", cfg.ScriptGeneration.FfmpegBinary, "/opt/bin/ffmpeg")
	}
}

func TestScriptGeneration_DefaultConfig(t *testing.T) {
	cfg := defaultCFG()
	if cfg.ScriptGeneration.Hardsub.Tag == "" {
		t.Error("ScriptGeneration.Hardsub.Tag should be non-empty")
	}
	if cfg.ScriptGeneration.FfmpegBinary == "" {
		t.Error("ScriptGeneration.FfmpegBinary should be non-empty")
	}
}
