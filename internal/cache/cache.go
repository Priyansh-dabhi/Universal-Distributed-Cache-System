package cache

import (
	"errors"
	"sync"
)

// ErrInvalidCapacity is returned when attempting to initialize a Cache with capacity <= 0.
var ErrInvalidCapacity = errors.New("capacity must be greater than 0")

// node represents a doubly linked list node storing key and value.
type node struct {
	key   string
	value string
	prev  *node
	next  *node
}

// Cache represents an in-memory thread-safe key-value store with LRU eviction.
type Cache struct {
	mu       sync.RWMutex
	capacity int
	items    map[string]*node
	head     *node // sentinel head: head.next points to most recently used (MRU)
	tail     *node // sentinel tail: tail.prev points to least recently used (LRU)
}

// New initializes and returns a new Cache instance with the specified maximum capacity.
// Returns ErrInvalidCapacity if capacity is less than or equal to zero.
func New(capacity int) (*Cache, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}

	head := &node{}
	tail := &node{}
	head.next = tail
	tail.prev = head

	return &Cache{
		capacity: capacity,
		items:    make(map[string]*node, capacity),
		head:     head,
		tail:     tail,
	}, nil
}

// addToFront places a node right after the sentinel head (making it MRU).
func (c *Cache) addToFront(n *node) {
	n.prev = c.head
	n.next = c.head.next
	c.head.next.prev = n
	c.head.next = n
}

// removeNode unlinks a node from the doubly linked list.
func (c *Cache) removeNode(n *node) {
	n.prev.next = n.next
	n.next.prev = n.prev
	n.prev = nil
	n.next = nil
}

// moveToFront moves an existing node to the MRU position (head.next).
func (c *Cache) moveToFront(n *node) {
	c.removeNode(n)
	c.addToFront(n)
}

// removeLRU unlinks and returns the least recently used node (tail.prev).
// Returns nil if the list is empty.
func (c *Cache) removeLRU() *node {
	if c.tail.prev == c.head {
		return nil
	}
	lru := c.tail.prev
	c.removeNode(lru)
	return lru
}

// Set stores a key-value pair in the cache. If the key already exists, its value is replaced
// and it is moved to the MRU position. If the cache is at capacity, the LRU item is evicted.
func (c *Cache) Set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if n, ok := c.items[key]; ok {
		n.value = value
		c.moveToFront(n)
		return
	}

	if len(c.items) >= c.capacity {
		if lru := c.removeLRU(); lru != nil {
			delete(c.items, lru.key)
		}
	}

	newNode := &node{
		key:   key,
		value: value,
	}
	c.addToFront(newNode)
	c.items[key] = newNode
}

// Get retrieves the value associated with the given key and marks it as MRU.
// The second return value indicates whether the key was present in the cache.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, ok := c.items[key]
	if !ok {
		return "", false
	}

	c.moveToFront(n)
	return n.value, true
}

// Delete removes the key and its value from the cache and recency list.
// It returns true if the key existed and was deleted, or false if it did not exist.
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, ok := c.items[key]
	if !ok {
		return false
	}

	c.removeNode(n)
	delete(c.items, key)
	return true
}

// Size returns the number of entries currently stored in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Capacity returns the maximum capacity configured for the cache.
func (c *Cache) Capacity() int {
	return c.capacity
}
