package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Galdoba/lazyam3/internal/infra/config"
	"github.com/Galdoba/lazyam3/internal/infra/logging"
)

func modulesWithLogger(t *testing.T) *AppModules {
	t.Helper()
	dir := t.TempDir()
	logger, err := logging.NewLogger(config.Logging{
		Level: "info",
		Path:  filepath.Join(dir, "lazyam3.log"),
	})
	if err != nil {
		t.Fatalf("logging.NewLogger: %v", err)
	}
	t.Cleanup(func() { logger.Close() })

	return &AppModules{
		Logger: logger,
		Config: config.Config{Storage: config.Storage{
			Path:           filepath.Join(dir, "history.jsonl"),
			ErrorStorePath: filepath.Join(dir, "errors.jsonl"),
			StatusPath:     filepath.Join(dir, "status.json"),
		}},
	}
}

func pathsInMissingDir(m *AppModules) {
	missing := filepath.Join(filepath.Dir(m.Config.Storage.Path), "missing-dir")
	m.Config.Storage.Path = filepath.Join(missing, "history.jsonl")
	m.Config.Storage.ErrorStorePath = filepath.Join(missing, "errors.jsonl")
	m.Config.Storage.StatusPath = filepath.Join(missing, "status.json")
}

func TestAppModules_Store_UnwritablePathFailsFast(t *testing.T) {
	m := modulesWithLogger(t)
	t.Cleanup(func() { m.Shutdown() })
	pathsInMissingDir(m)

	store, err := m.Store()
	if err == nil {
		t.Fatal("Store(): want error for an unwritable path, got nil")
	}
	if store != nil {
		t.Errorf("Store(): want nil store on error, got %v", store)
	}
	if !strings.Contains(err.Error(), "failed to open store file") {
		t.Errorf("Store(): want wrapped open error, got %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Store(): want error to unwrap to os.ErrNotExist, got %v", err)
	}

	retry, err := m.Store()
	if retry != nil || err == nil {
		t.Errorf("Store(): want the same cached failure, got store=%v err=%v", retry, err)
	}
}

func TestAppModules_ErrorStore_UnwritablePathFailsFast(t *testing.T) {
	m := modulesWithLogger(t)
	t.Cleanup(func() { m.Shutdown() })
	pathsInMissingDir(m)

	store, err := m.ErrorStore()
	if err == nil {
		t.Fatal("ErrorStore(): want error for an unwritable path, got nil")
	}
	if store != nil {
		t.Errorf("ErrorStore(): want nil store on error, got %v", store)
	}
	if !strings.Contains(err.Error(), "failed to open store file") {
		t.Errorf("ErrorStore(): want wrapped open error, got %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ErrorStore(): want error to unwrap to os.ErrNotExist, got %v", err)
	}

	retry, err := m.ErrorStore()
	if retry != nil || err == nil {
		t.Errorf("ErrorStore(): want the same cached failure, got store=%v err=%v", retry, err)
	}
}

func TestAppModules_EnsureStatusFile_UnwritablePathFailsFast(t *testing.T) {
	m := modulesWithLogger(t)
	t.Cleanup(func() { m.Shutdown() })
	pathsInMissingDir(m)

	err := m.EnsureStatusFile()
	if err == nil {
		t.Fatal("EnsureStatusFile(): want error for an unwritable path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open status file") {
		t.Errorf("EnsureStatusFile(): want wrapped open error, got %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("EnsureStatusFile(): want error to unwrap to os.ErrNotExist, got %v", err)
	}
	if _, statErr := os.Stat(m.Config.Storage.StatusPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("EnsureStatusFile(): want no status file created, stat = %v", statErr)
	}
}

func TestAppModules_WritablePathsAreCreated(t *testing.T) {
	m := modulesWithLogger(t)
	t.Cleanup(func() { m.Shutdown() })

	store, err := m.Store()
	if err != nil || store == nil {
		t.Fatalf("Store(): want an opened store, got store=%v err=%v", store, err)
	}
	errorStore, err := m.ErrorStore()
	if err != nil || errorStore == nil {
		t.Fatalf("ErrorStore(): want an opened store, got store=%v err=%v", errorStore, err)
	}
	if err := m.EnsureStatusFile(); err != nil {
		t.Fatalf("EnsureStatusFile(): want nil error, got %v", err)
	}

	for _, path := range []string{
		m.Config.Storage.Path,
		m.Config.Storage.ErrorStorePath,
		m.Config.Storage.StatusPath,
	} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("want %s to exist after startup, stat = %v", path, statErr)
		}
	}
}

func TestAppModules_MethodsBeforeInit(t *testing.T) {
	m := &AppModules{}

	if _, err := m.Store(); err == nil || !strings.Contains(err.Error(), "call Init first") {
		t.Errorf("Store(): want 'call Init first', got %v", err)
	}
	if _, err := m.ErrorStore(); err == nil || !strings.Contains(err.Error(), "call Init first") {
		t.Errorf("ErrorStore(): want 'call Init first', got %v", err)
	}
	if err := m.EnsureStatusFile(); err == nil || !strings.Contains(err.Error(), "call Init first") {
		t.Errorf("EnsureStatusFile(): want 'call Init first', got %v", err)
	}
}

func TestAppModules_Init_LoggerPathCannotBeOpened(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAZYAM3_CONFIG", "")
	t.Setenv("LAZYAM3_CONFIG_PATH", "")

	loggerPath := filepath.Join(t.TempDir(), "missing-dir", "lazyam3.log")
	path := writeConfigFile(t, fmt.Sprintf(`[paths.generator]
input_directory = '/in'
output_directory = '/out'
metadata_path = '/meta.json'

[logger]
level = 'info'
path = '%s'
`, loggerPath))

	m := New()
	err := m.Init(path)
	if err == nil {
		t.Fatal("Init(): want error when the log path cannot be opened, got nil")
	}
	if !strings.Contains(err.Error(), "failed to init logger") {
		t.Errorf("Init(): want wrapped logger error, got %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Init(): want error to unwrap to os.ErrNotExist, got %v", err)
	}
	if m.Logger != nil {
		t.Errorf("Init(): want no logger on error, got %v", m.Logger)
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
