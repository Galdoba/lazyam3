// Package jsonstore — простое key-value хранилище, персистентное в JSON-файл.
// Эмулирует БД: методы Get/Set/Delete/Close совпадают с тем, что было бы
// у реального хранилища, поэтому замена на настоящую БД не затронет команды.
package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// Store — потокобезопасное key-value хранилище в памяти,
// выгружаемое в JSON-файл по Close/Save.
type Store struct {
	path   string
	mu     sync.RWMutex
	data   map[string]any
	closed bool
}

// New открывает (или создаёт при первом Save) хранилище по указанному пути.
// Если файл существует — данные загружаются в память.
func New(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("store path is empty")
	}
	s := &Store{
		path: path,
		data: make(map[string]any),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil // новый пустой store
	}
	if err != nil {
		return fmt.Errorf("failed to read store file: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return fmt.Errorf("failed to unmarshal store file: %w", err)
	}
	return nil
}

// Get возвращает значение и признак наличия ключа.
func (s *Store) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// Set записывает значение. Изменения в памяти; для персистентности
// вызывайте Save явно или полагайтесь на Close при Shutdown.
func (s *Store) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("store is closed")
	}
	s.data[key] = value
	return nil
}

// Delete удаляет ключ. Отсутствие ключа — не ошибка.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Keys возвращает срез ключей (порядок не определён).
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// Len возвращает число записей.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Save выгружает текущее содержимое в файл.
func (s *Store) Save() error {
	s.mu.RLock()
	snapshot := make(map[string]any, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	s.mu.RUnlock()

	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}
	if err := os.WriteFile(s.path, raw, 0o644); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}
	return nil
}

// Close сохраняет данные и помечает store закрытым.
// Идемпотентен.
func (s *Store) Close() error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	return s.Save()
}
