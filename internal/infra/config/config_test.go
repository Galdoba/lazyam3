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
	if cfg.Hardsub.Tag != "Forsirovannye" {
		t.Errorf("Hardsub.Tag = %q, want %q", cfg.Hardsub.Tag, "Forsirovannye")
	}
}

func TestDefaultConfig_FfprobeBinary(t *testing.T) {
	cfg := defaultCFG()
	if cfg.Paths.GeneratorFFprobe() != "" {
		t.Errorf("Paths.Generator.FfprobeBinary = %q, want empty string", cfg.Paths.GeneratorFFprobe())
	}
}

func TestDefaultConfig_LoggerPath(t *testing.T) {
	cfg := defaultCFG()
	if cfg.Logger.Path != "/path/to/file" {
		t.Errorf("Logger.Path = %q, want %q", cfg.Logger.Path, "/path/to/file")
	}
}

func TestDefaultConfig_ExecutorMap(t *testing.T) {
	cfg := defaultCFG()
	if cfg.Paths.Executor == nil {
		t.Error("Paths.Executor should not be nil")
	}
	if len(cfg.Paths.Executor) != 0 {
		t.Errorf("Paths.Executor should be empty, got %d entries", len(cfg.Paths.Executor))
	}
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

[hardsub]
tag = 'Forsirovannye'
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.toml")
	if err := os.WriteFile(path, []byte(toml), 0644); err != nil {
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
