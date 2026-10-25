// Package cache is a template of thread-safe in-memory key-value cache.
// Lives within the process, not persistent.
package cache

import "sync"

// Cache is a simple wrapper over map[string]any with a mutex.
type Cache struct {
	mu   sync.RWMutex
	data map[string]any
}

// New creates an empty cache.
//log=debug msg="cache created"
func New() *Cache {
	return &Cache{
		data: make(map[string]any),
	}
}

// Get returns the value and whether the key exists.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[key]
	return v, ok
}

// Set writes a value.
//log=debug msg="cache set key=%s", key
func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Delete removes a key.
//log=debug msg="cache delete key=%s", key
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear clears the cache.
//log=debug msg="cache cleared"
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]any)
}

// Len returns the number of entries.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Close is a no-op: in-memory cache doesn't require resource release.
// Method kept for consistency with other modules.
func (c *Cache) Close() error {
	return nil
}
