package cache

import "sync"

// Cache represents an in-memory thread-safe key-value store.
type Cache struct {
	mu    sync.RWMutex
	items map[string]string
}

// New initializes and returns a new Cache instance.
func New() *Cache {
	return &Cache{
		items: make(map[string]string),
	}
}

// Set stores a key-value pair in the cache. If the key already exists, its value is replaced.
func (c *Cache) Set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = value
}

// Get retrieves the value associated with the given key.
// The second return value indicates whether the key was present in the cache.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.items[key]
	return val, ok
}

// Delete removes the key and its value from the cache.
// It returns true if the key existed and was deleted, or false if it did not exist.
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.items[key]; ok {
		delete(c.items, key)
		return true
	}
	return false
}

// Size returns the number of entries currently stored in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}
