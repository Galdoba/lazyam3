// Package jsonstore is template of a simple key-value store persisted to a JSON file.
// It emulates a database: Get/Set/Delete/Close methods match what a real
// store would have, so swapping in a real database won't affect commands.
package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// Store is a thread-safe in-memory key-value store
// that is flushed to a JSON file on Close/Save.
type Store struct {
	path   string
	mu     sync.RWMutex
	data   map[string]any
	closed bool
}

// New opens (or creates on first Save) the store at the given path.
// If the file exists, data is loaded into memory.
//log=info msg="jsonstore created at: %s", path
func New(path string) (*Store, error) {
	if path == "" {
		//log=error msg="store path is empty"
		return nil, fmt.Errorf("store path is empty")
	}
	s := &Store{
		path: path,
		data: make(map[string]any),
	}
	if err := s.load(); err != nil {
		//log=error msg="failed to load store from %q: %v", path, err
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil // new empty store
	}
	if err != nil {
		//log=error msg="failed to read store file: %v", err
		return fmt.Errorf("failed to read store file: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		//log=error msg="failed to unmarshal store file: %v", err
		return fmt.Errorf("failed to unmarshal store file: %w", err)
	}
	//log=debug msg="store loaded: %d entries", len(s.data)
	return nil
}

// Get returns the value and whether the key exists.
func (s *Store) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// Set writes a value. Changes are in-memory; to persist,
// call Save explicitly or rely on Close during Shutdown.
//log=debug msg="store set key=%s", key
func (s *Store) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		//log=error msg="store is closed, cannot set key=%s", key
		return fmt.Errorf("store is closed")
	}
	s.data[key] = value
	return nil
}

// Delete removes a key. Missing key is not an error.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Keys returns a slice of keys (order is not guaranteed).
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of entries.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Save flushes the current contents to the file.
//log=info msg="store saved: %d entries", len(s.data)
func (s *Store) Save() error {
	s.mu.RLock()
	snapshot := make(map[string]any, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	s.mu.RUnlock()

	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		//log=error msg="failed to marshal store: %v", err
		return fmt.Errorf("failed to marshal store: %w", err)
	}
	if err := os.WriteFile(s.path, raw, 0o644); err != nil {
		//log=error msg="failed to write store file: %v", err
		return fmt.Errorf("failed to write store file: %w", err)
	}
	return nil
}

// Close saves data and marks the store as closed.
// Idempotent.
//log=info msg="store closed and saved"
func (s *Store) Close() error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	return s.Save()
}
