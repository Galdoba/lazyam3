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

// Paths holds application directory paths.
type Paths struct {
	InputDirectory   string `toml:"input_directory"`
	OutputDirectory  string `toml:"output_directory"`
	MetadataPath     string `toml:"metadata_path"`
	FfprobeBinary    string `toml:"ffprobe_binary,omitempty"`
	SourceRootIn     string `json:"source_root_in"`     //where source files are located
	TargetRootDir    string `json:"target_root_dir"`    //where script is sending results to
	Buffer           string `json:"buffer"`             //where scripts and preapared sources land
	ProcessingDir    string `json:"processing_dir"`     //where script is working with files
	DoneDir          string `json:"done_dir"`           //where script is placing processed sources
	ScriptArchiveDir string `json:"script_archive_dir"` //where script is sending it's self after processing
}

type Logging struct {
	Level    string `toml:"level"`
	Path     string `toml:"path"`
	Terminal bool   `toml:"terminal"`
}

func defaultCFG() Config {
	return Config{
		Paths: Paths{
			InputDirectory:  "/tmp/lazyam3/input",
			OutputDirectory: "/tmp/lazyam3/output",
			MetadataPath:    "/tmp/lazyam3/metadata.json",
			FfprobeBinary:   "",
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
