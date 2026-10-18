package config

import (
	"fmt"

	"github.com/Galdoba/appcontext/configmanager"
)

const (
	APPNAME = "lazyam3"
)

type Config struct {
	Paths   Paths   `json:"paths"`
	Logger  Logging `json:"logger"`
	Storage Storage `json:"storage"`
}

// Storage holds store settings. Currently only the path to the JSON file;
// when replacing jsonstore with a real database, DSN, timeouts, etc. will be added here.
type Storage struct {
	Path string `json:"path"`
}

// Paths holds application directory paths.
type Paths struct {
	InputDirectory  string `json:"input_directory"`
	OutputDirectory string `json:"output_directory"`
	MetadataPath    string `json:"metadata_path"`
}

type Logging struct {
	Level    string `json:"level"`
	Path     string `json:"path"`
	Terminal bool   `json:"terminal"`
}

func defaultCFG() Config {
	return Config{
		Paths: Paths{
			InputDirectory:  "/tmp/lazyam3/input",
			OutputDirectory: "/tmp/lazyam3/output",
			MetadataPath:    "/tmp/lazyam3/metadata.json",
		},
		Logger: Logging{
			Level:    "info",
			Path:     "/path/to/file",
			Terminal: true,
		},
		Storage: Storage{
			Path: "/path/to/file.json",
		},
	}
}

func Load(suggestedPath string) (Config, error) {
	cm, err := configmanager.New[Config](APPNAME, defaultCFG(), configmanager.WithFormat(configmanager.JSON))
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
