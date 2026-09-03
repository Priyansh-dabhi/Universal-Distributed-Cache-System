package cache

import (
	"errors"
	"fmt"
	"sync"
)

// EvictionPolicy defines the type of eviction algorithm used by the Cache.
type EvictionPolicy string

const (
	// PolicyLRU uses the Least Recently Used eviction strategy.
	PolicyLRU EvictionPolicy = "LRU"
	// PolicyLFU uses the Least Frequently Used eviction strategy with LRU tie-breaking.
	PolicyLFU EvictionPolicy = "LFU"
)

// ErrInvalidCapacity is returned when attempting to initialize a Cache with capacity <= 0.
var ErrInvalidCapacity = errors.New("capacity must be greater than 0")

// ErrUnsupportedPolicy is returned when an unrecognized EvictionPolicy is requested.
var ErrUnsupportedPolicy = errors.New("unsupported eviction policy")

// storage is the internal interface implemented by cache eviction strategies.
type storage interface {
	get(key string) (string, bool)
	set(key string, value string)
	delete(key string) bool
	size() int
}

// frequencyTracker is implemented by eviction policies that track frequency (such as LFU).
type frequencyTracker interface {
	getFrequency(key string) (int, bool)
}

// Cache represents an in-memory thread-safe key-value store supporting configurable eviction policies.
type Cache struct {
	mu       sync.RWMutex
	capacity int
	policy   EvictionPolicy
	storage  storage
}

// New initializes and returns a new Cache instance defaulting to LRU eviction.
// Returns ErrInvalidCapacity if capacity is less than or equal to zero.
func New(capacity int) (*Cache, error) {
	return NewWithPolicy(capacity, PolicyLRU)
}

// NewLRU initializes and returns a new Cache instance with the LRU eviction policy.
func NewLRU(capacity int) (*Cache, error) {
	return NewWithPolicy(capacity, PolicyLRU)
}

// NewLFU initializes and returns a new Cache instance with the LFU eviction policy.
func NewLFU(capacity int) (*Cache, error) {
	return NewWithPolicy(capacity, PolicyLFU)
}

// NewWithPolicy initializes and returns a new Cache instance with the specified eviction policy.
func NewWithPolicy(capacity int, policy EvictionPolicy) (*Cache, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}

	var s storage
	switch policy {
	case PolicyLRU:
		s = newLRUCache(capacity)
	case PolicyLFU:
		s = newLFUCache(capacity)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPolicy, policy)
	}

	return &Cache{
		capacity: capacity,
		policy:   policy,
		storage:  s,
	}, nil
}

// Set stores a key-value pair in the cache according to the configured eviction policy.
func (c *Cache) Set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.storage.set(key, value)
}

// Get retrieves the value associated with the given key and updates eviction recency/frequency.
// The second return value indicates whether the key was present in the cache.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.storage.get(key)
}

// Delete removes the key and its value from the cache and eviction metadata.
// It returns true if the key existed and was deleted, or false if it did not exist.
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.storage.delete(key)
}

// Size returns the number of entries currently stored in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.storage.size()
}

// Capacity returns the maximum capacity configured for the cache.
func (c *Cache) Capacity() int {
	return c.capacity
}

// Policy returns the eviction policy used by the cache.
func (c *Cache) Policy() EvictionPolicy {
	return c.policy
}

// GetFrequency returns the access frequency for a key if the cache policy tracks frequency (e.g. LFU).
func (c *Cache) GetFrequency(key string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if ft, ok := c.storage.(frequencyTracker); ok {
		return ft.getFrequency(key)
	}
	return 0, false
}

// ContainsInLRUList returns whether a key is present in the LRU doubly linked list (used for test validation).
func (c *Cache) ContainsInLRUList(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if lru, ok := c.storage.(*lruCache); ok {
		return lru.containsInList(key)
	}
	return false
}
