package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Galdoba/appcontext/configmanager"
)

const (
	APPNAME = "lazyam3"
)

// ErrDefaultConfig is returned when the effective configuration equals the built-in default.
var ErrDefaultConfig = errors.New("default configuration in use")

// requiredConfigFields lists the config fields the user must fill in before processing can run.
var requiredConfigFields = []string{
	"paths.generator.input_directory",
	"paths.generator.output_directory",
	"paths.generator.metadata_path",
}

// DefaultConfigError reports that the effective configuration equals the built-in default and
// the user must edit the pointed config file to fill in the required fields.
type DefaultConfigError struct {
	ConfigPath string
}

func (e *DefaultConfigError) Error() string {
	return fmt.Sprintf("default configuration in use: edit the config file %q and fill in the required fields (%s) to enable processing", e.ConfigPath, strings.Join(requiredConfigFields, ", "))
}

func (e *DefaultConfigError) Unwrap() error { return ErrDefaultConfig }

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
		// All paths are obvious placeholders — the user must edit the config
		// file before the effective configuration equals the built-in default.
		Paths: Paths{
			Generator: GeneratorPaths{
				InputDirectory:  "/path/to/lazyam3/input",
				OutputDirectory: "/path/to/lazyam3/output",
				MetadataPath:    "/path/to/lazyam3/metadata.json",
				FfprobeBinary:   "ffprobe",
			},
			// Executor defaults mirror buildScriptConfig fallbacks for
			// root = Generator.OutputDirectory — keep them in sync.
			Executor: map[string]string{
				"target":        "/path/to/lazyam3/output/target",
				"buffer":        "/path/to/lazyam3/output",
				"in_progress":   "/path/to/lazyam3/output",
				"done":          "/path/to/lazyam3/output",
				"notifications": "/path/to/lazyam3/output",
				"archive":       "/path/to/lazyam3/output/archive",
			},
		},
		Logger: Logging{
			Level:    "info",
			Path:     "/path/to/lazyam3/lazyam3.log",
			Terminal: true,
			Format:   "text",
			Color:    true,
		},
		Storage: Storage{
			Path: "/path/to/lazyam3/storage.json",
		},
		ScriptGeneration: ScriptGeneration{
			Hardsub:      Hardsub{Tag: "Forsirovannye"},
			FfmpegBinary: "ffmpeg",
		},
	}
}

// IsDefault reports whether the effective configuration equals the built-in default.
func (c Config) IsDefault() bool {
	return reflect.DeepEqual(c, defaultCFG())
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
	var configPath string
	switch suggestedPath {
	case "":
		if err := cm.EnsureLoaded(); err != nil {
			return Config{}, fmt.Errorf("failed to load config: %w", err)
		}
		configPath = cm.Path()
	default:
		if err := cm.DiscoverAndLoad(suggestedPath); err != nil {
			return Config{}, fmt.Errorf("failed to load config: %w", err)
		}
		configPath = suggestedPath
	}
	cfg := cm.Config()
	if cfg.IsDefault() {
		return Config{}, &DefaultConfigError{ConfigPath: configPath}
	}
	return cfg, nil
}
