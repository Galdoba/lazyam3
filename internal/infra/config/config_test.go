package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultCFG()

	if cfg.Paths.InputDirectory != "/tmp/lazyam3/input" {
		t.Errorf("Paths.InputDirectory = %q, want %q", cfg.Paths.InputDirectory, "/tmp/lazyam3/input")
	}
	if cfg.Paths.OutputDirectory != "/tmp/lazyam3/output" {
		t.Errorf("Paths.OutputDirectory = %q, want %q", cfg.Paths.OutputDirectory, "/tmp/lazyam3/output")
	}
	if cfg.Paths.MetadataPath != "/tmp/lazyam3/metadata.json" {
		t.Errorf("Paths.MetadataPath = %q, want %q", cfg.Paths.MetadataPath, "/tmp/lazyam3/metadata.json")
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
	if cfg.Paths.FfprobeBinary != "" {
		t.Errorf("Paths.FfprobeBinary = %q, want empty string", cfg.Paths.FfprobeBinary)
	}
}

func TestDefaultConfig_LoggerPath(t *testing.T) {
	cfg := defaultCFG()
	if cfg.Logger.Path != "/path/to/file" {
		t.Errorf("Logger.Path = %q, want %q", cfg.Logger.Path, "/path/to/file")
	}
}
