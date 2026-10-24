package config

import (
	"fmt"

	"github.com/Galdoba/appcontext/configmanager"
)

const (
	APPNAME = "lazyam3"
)

type Config struct {
	Paths            Paths            `toml:"paths"`
	Logger           Logging          `toml:"logger"`
	Storage          Storage          `toml:"storage"`
	ScriptGeneration ScriptGeneration `toml:"script_generation"`
}

// ScriptGeneration holds all script-generation settings.
type ScriptGeneration struct {
	Hardsub      Hardsub `toml:"hardsub"`
	FfmpegBinary string  `toml:"ffmpeg_binary,omitempty"`
}

// Hardsub holds hardsub classification rules.
type Hardsub struct {
	Tag string `toml:"tag"` // filename tag marking a sidecar subtitle as hardsub (burn-in)
}

// Storage holds store settings. Currently only the path to the JSON file;
// when replacing jsonstore with a real database, DSN, timeouts, etc. will be added here.
type Storage struct {
	Path string `toml:"path"`
}

// GeneratorPaths holds directory paths used by the generator process.
type GeneratorPaths struct {
	InputDirectory  string `toml:"input_directory"`
	OutputDirectory string `toml:"output_directory"`
	MetadataPath    string `toml:"metadata_path"`
	FfprobeBinary   string `toml:"ffprobe_binary,omitempty"`
}

// Paths holds application directory paths, split into generator and executor
// namespaces. Generator paths are used by lazyam3 CLI; executor paths are
// embedded into generated scripts for the executor to use.
type Paths struct {
	Generator GeneratorPaths    `toml:"generator"`
	Executor  map[string]string `toml:"executor"`
}

// GeneratorInput returns the generator input directory.
func (p Paths) GeneratorInput() string { return p.Generator.InputDirectory }

// GeneratorOutput returns the generator output directory.
func (p Paths) GeneratorOutput() string { return p.Generator.OutputDirectory }

// GeneratorMetadata returns the metadata JSON path.
func (p Paths) GeneratorMetadata() string { return p.Generator.MetadataPath }

// GeneratorFFprobe returns the ffprobe binary path.
func (p Paths) GeneratorFFprobe() string { return p.Generator.FfprobeBinary }

// ExecutorPath returns an executor path by key.
func (p Paths) ExecutorPath(key string) string { return p.Executor[key] }

type Logging struct {
	Level    string `toml:"level"`
	Path     string `toml:"path"`
	Terminal bool   `toml:"terminal"`
	Format   string `toml:"format"` // "json" (default) or "text" — terminal output only
	Color    bool   `toml:"color"`  // colorize level in terminal text output
}

func defaultCFG() Config {
	return Config{
		Paths: Paths{
			Generator: GeneratorPaths{
				InputDirectory:  "/tmp/lazyam3/input",
				OutputDirectory: "/tmp/lazyam3/output",
				MetadataPath:    "/tmp/lazyam3/metadata.json",
				FfprobeBinary:   "ffprobe",
			},
			// Executor defaults mirror buildScriptConfig fallbacks for
			// root = Generator.OutputDirectory — keep them in sync.
			Executor: map[string]string{
				"target":        "/tmp/lazyam3/output/target",
				"buffer":        "/tmp/lazyam3/output",
				"in_progress":   "/tmp/lazyam3/output",
				"done":          "/tmp/lazyam3/output",
				"notifications": "/tmp/lazyam3/output",
				"archive":       "/tmp/lazyam3/output/archive",
			},
		},
		Logger: Logging{
			Level:    "info",
			Path:     "/path/to/file.log",
			Terminal: true,
			Format:   "text",
			Color:    true,
		},
		Storage: Storage{
			Path: "/path/to/file.json",
		},
		ScriptGeneration: ScriptGeneration{
			Hardsub:      Hardsub{Tag: "Forsirovannye"},
			FfmpegBinary: "ffmpeg",
		},
	}
}

func (c Config) Validate() error {
	switch c.Logger.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logger.level: invalid value %q (want debug, info, warn or error)", c.Logger.Level)
	}
	if c.Paths.Generator.InputDirectory == "" {
		return fmt.Errorf("paths.generator.input_directory is required")
	}
	if c.Paths.Generator.OutputDirectory == "" {
		return fmt.Errorf("paths.generator.output_directory is required")
	}
	if c.Paths.Generator.MetadataPath == "" {
		return fmt.Errorf("paths.generator.metadata_path is required")
	}
	return nil
}

func Load(suggestedPath string) (Config, error) {
	cm, err := configmanager.New(APPNAME, defaultCFG(), configmanager.WithFormat(configmanager.TOML))
	if err != nil {
		return Config{}, fmt.Errorf("failed to create configmanager: %w", err)
	}
	switch suggestedPath {
	case "":
		if err := cm.EnsureLoaded(); err != nil {
			return Config{}, fmt.Errorf("failed to load config: %w", err)
		}
	default:
		if err := cm.DiscoverAndLoad(suggestedPath); err != nil {
			return Config{}, fmt.Errorf("failed to load config: %w", err)
		}
	}
	return cm.Config(), nil
}
