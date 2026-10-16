// Package cache — потокобезопасный in-memory key-value кэш.
// Живёт в пределах процесса, не персистентен.
package cache

import "sync"

// Cache — простая обёртка над map[string]any с мьютексом.
type Cache struct {
	mu   sync.RWMutex
	data map[string]any
}

// New создаёт пустой кэш.
func New() *Cache {
	return &Cache{
		data: make(map[string]any),
	}
}

// Get возвращает значение и признак наличия ключа.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

// Set записывает значение.
func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Delete удаляет ключ.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear очищает кэш.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]any)
}

// Len возвращает число записей.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Close — no-op: in-memory кэш не требует освобождения ресурсов.
// Метод оставлен для единообразия с другими модулями.
func (c *Cache) Close() error {
	return nil
}