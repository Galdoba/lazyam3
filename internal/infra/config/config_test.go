package config

import (
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
