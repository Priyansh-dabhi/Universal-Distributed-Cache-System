package cache

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"universal-distributed-cache/internal/metrics"
)

// EvictionPolicy defines the type of eviction algorithm used by the Cache.
type EvictionPolicy string

const (
	// PolicyLRU uses the Least Recently Used eviction strategy.
	PolicyLRU EvictionPolicy = "LRU"
	// PolicyLFU uses the Least Frequently Used eviction strategy with LRU tie-breaking.
	PolicyLFU EvictionPolicy = "LFU"
	// Policy2Q uses the Two-Queue (2Q) eviction strategy separating new and frequent entries.
	Policy2Q EvictionPolicy = "2Q"
)

// Default2QA1Ratio specifies the default percentage of total capacity allocated to queue A1 in 2Q.
const Default2QA1Ratio = 0.25

// ErrInvalidCapacity is returned when attempting to initialize a Cache with invalid capacity (<= 0).
var ErrInvalidCapacity = errors.New("capacity must be greater than 0")

// ErrUnsupportedPolicy is returned when an unrecognized EvictionPolicy is requested.
var ErrUnsupportedPolicy = errors.New("unsupported eviction policy")

// storage is the internal interface implemented by cache eviction strategies.
type storage interface {
	get(key string) (val string, found bool, expired bool)
	set(key string, value string, expiresAt time.Time) (evicted bool)
	delete(key string) (deleted bool, expired bool)
	size() int
}

// frequencyTracker is implemented by eviction policies that track frequency (such as LFU).
type frequencyTracker interface {
	getFrequency(key string) (int, bool)
}

// twoQInspector is implemented by 2Q eviction policy for queue-specific introspection.
type twoQInspector interface {
	isInA1(key string) bool
	isInAm(key string) bool
	a1Size() int
	amSize() int
}

// Cache represents an in-memory thread-safe key-value store supporting configurable eviction policies,
// TTL expiration, and atomic telemetry metrics.
type Cache struct {
	mu       sync.RWMutex
	capacity int
	policy   EvictionPolicy
	storage  storage
	metrics  *metrics.CacheMetrics
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

// New2Q initializes and returns a new Cache instance with the 2Q eviction policy using default capacity ratio (25% A1, 75% Am).
func New2Q(capacity int) (*Cache, error) {
	return NewWithPolicy(capacity, Policy2Q)
}

// New2QWithCapacities initializes and returns a new Cache instance with 2Q eviction policy and custom A1 and Am queue capacities.
func New2QWithCapacities(capacity, a1Cap, amCap int) (*Cache, error) {
	if capacity <= 0 || a1Cap <= 0 || amCap <= 0 {
		return nil, ErrInvalidCapacity
	}

	return &Cache{
		capacity: capacity,
		policy:   Policy2Q,
		storage:  newTwoQCache(capacity, a1Cap, amCap),
		metrics:  metrics.NewCacheMetrics(),
	}, nil
}

// calculate2QCapacities calculates the default A1 and Am capacities based on total capacity.
func calculate2QCapacities(capacity int) (int, int) {
	if capacity <= 1 {
		return 1, 1
	}
	a1Cap := int(float64(capacity) * Default2QA1Ratio)
	if a1Cap < 1 {
		a1Cap = 1
	}
	amCap := capacity - a1Cap
	if amCap < 1 {
		amCap = 1
	}
	return a1Cap, amCap
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
	case Policy2Q:
		a1Cap, amCap := calculate2QCapacities(capacity)
		s = newTwoQCache(capacity, a1Cap, amCap)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPolicy, policy)
	}

	return &Cache{
		capacity: capacity,
		policy:   policy,
		storage:  s,
		metrics:  metrics.NewCacheMetrics(),
	}, nil
}

// Metrics returns the CacheMetrics tracking instance for this cache.
func (c *Cache) Metrics() *metrics.CacheMetrics {
	return c.metrics
}

// Set stores a key-value pair in the cache without expiration.
// If the key previously had a TTL, the expiration is removed.
func (c *Cache) Set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := c.storage.set(key, value, time.Time{})
	c.metrics.IncSets()
	if evicted {
		c.metrics.IncEvictions()
	}
}

// SetWithTTL stores a key-value pair in the cache with a Time-To-Live (TTL).
// A TTL <= 0 causes immediate expiration (deleting any existing entry and not storing a new one).
func (c *Cache) SetWithTTL(key string, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl <= 0 {
		deleted, expired := c.storage.delete(key)
		if deleted {
			c.metrics.IncDeletes()
		}
		if expired {
			c.metrics.IncExpired()
		}
		return
	}

	evicted := c.storage.set(key, value, time.Now().Add(ttl))
	c.metrics.IncSets()
	if evicted {
		c.metrics.IncEvictions()
	}
}

// Get retrieves the value associated with the given key and updates eviction recency/frequency/queue.
// If the entry has expired, it is lazily removed and ("", false) is returned.
// GET requires an exclusive lock because it mutates eviction recency/frequency/queue state or purges expired items.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	val, found, expired := c.storage.get(key)
	if expired {
		c.metrics.IncExpired()
		c.metrics.IncMisses()
		return "", false
	}
	if !found {
		c.metrics.IncMisses()
		return "", false
	}
	c.metrics.IncHits()
	return val, true
}

// Delete removes the key and its value from the cache and eviction metadata.
// If the entry is already expired, it returns false while cleaning up any stale metadata.
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	deleted, expired := c.storage.delete(key)
	if expired {
		c.metrics.IncExpired()
	}
	if deleted {
		c.metrics.IncDeletes()
	}
	return deleted
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

// IsInA1 returns true if key is currently in queue A1 of a 2Q cache.
func (c *Cache) IsInA1(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if tq, ok := c.storage.(twoQInspector); ok {
		return tq.isInA1(key)
	}
	return false
}

// IsInAm returns true if key is currently in queue Am of a 2Q cache.
func (c *Cache) IsInAm(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if tq, ok := c.storage.(twoQInspector); ok {
		return tq.isInAm(key)
	}
	return false
}

// A1Size returns the number of entries currently stored in queue A1 of a 2Q cache.
func (c *Cache) A1Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if tq, ok := c.storage.(twoQInspector); ok {
		return tq.a1Size()
	}
	return 0
}

// AmSize returns the number of entries currently stored in queue Am of a 2Q cache.
func (c *Cache) AmSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if tq, ok := c.storage.(twoQInspector); ok {
		return tq.amSize()
	}
	return 0
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
