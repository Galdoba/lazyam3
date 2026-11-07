// Package jsonstore is an append-only JSONL store. Each Set call appends a
// single JSON line to the file. Reading is handled externally by consumers.
package jsonstore

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Galdoba/lazyam3/internal/infra/logging"
)

// Store is a thread-safe append-only JSONL store.
// Each Set call appends a single JSON line to the file.
type Store struct {
	path   string
	mu     sync.Mutex
	file   *os.File
	logger *logging.Logger
}

// New opens (or creates) the store at the given path in append mode.
func New(path string, logger *logging.Logger) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("store path is empty")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open store file: %w", err)
	}
	s := &Store{
		path:   path,
		file:   file,
		logger: logger,
	}
	return s, nil
}

// Set appends a value as a single JSON line.
// The key parameter is retained for signature compatibility but not used.
func (s *Store) Set(_ string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return fmt.Errorf("store is closed")
	}
	enc := json.NewEncoder(s.file)
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("failed to encode store entry: %w", err)
	}
	return nil
}

// Close syncs and closes the file. Idempotent.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	if s.file == nil {
		return nil
	}
	if s.logger != nil {
		s.logger.Debug("store closed and synced")
	}
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync store file: %w", err)
	}
	if err := s.file.Close(); err != nil {
		return fmt.Errorf("failed to close store file: %w", err)
	}
	s.file = nil
	return nil
}
