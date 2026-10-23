package config

import (
	"fmt"

	"github.com/Galdoba/appcontext/configmanager"
)

const (
	APPNAME = "lazyam3"
)

type Config struct {
	Paths   Paths   `toml:"paths"`
	Logger  Logging `toml:"logger"`
	Storage Storage `toml:"storage"`
	Hardsub Hardsub `toml:"hardsub"`
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
}

func defaultCFG() Config {
	return Config{
		Paths: Paths{
			Generator: GeneratorPaths{
				InputDirectory:  "/tmp/lazyam3/input",
				OutputDirectory: "/tmp/lazyam3/output",
				MetadataPath:    "/tmp/lazyam3/metadata.json",
				FfprobeBinary:   "",
			},
			Executor: map[string]string{},
		},
		Logger: Logging{
			Level:    "info",
			Path:     "/path/to/file",
			Terminal: true,
		},
		Storage: Storage{
			Path: "/path/to/file.json",
		},
		Hardsub: Hardsub{
			Tag: "Forsirovannye",
		},
	}
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
