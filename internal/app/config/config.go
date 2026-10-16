package config

import (
	"fmt"

	"github.com/Galdoba/appcontext/configmanager"
)

const (
	APPNAME = "apptemplate"
)

type Config struct {
	Process1 Process1 `json:"process_1"`
	Process2 Process2 `json:"process_2"`
	Logger   Logging  `json:"logger"`
	Storage  Storage  `json:"storage"`
}

// Storage — настройки хранилища. Сейчас только путь к JSON-файлу;
// при замене jsonstore на настоящую БД здесь появятся DSN, таймауты и т.п.
type Storage struct {
	Path string `json:"path"`
}

type Process1 struct {
	Name     string   `json:"name"`
	Defaults []string `json:"defaults"`
}
type Process2 struct {
	Name     string `json:"name"`
	Defaults []int  `json:"defaults"`
}

type Logging struct {
	Level    string `json:"level"`
	Path     string `json:"path"`
	Terminal bool   `json:"terminal"`
}

func defaultCFG() Config {
	return Config{
		Process1: Process1{
			Name:     "p1",
			Defaults: []string{"val 1", "val 2"},
		},
		Process2: Process2{
			Name:     "p2",
			Defaults: []int{1, 2},
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
