package cache

import "time"

// lfuNode represents an item stored in the LFU cache with key, value, frequency, and expiration.
type lfuNode struct {
	key       string
	value     string
	freq      int
	expiresAt time.Time
	prev      *lfuNode
	next      *lfuNode
}

func (n *lfuNode) isExpired(now time.Time) bool {
	return !n.expiresAt.IsZero() && !now.Before(n.expiresAt)
}

// lfuList is a doubly linked list of lfuNodes sharing the same access frequency.
// The list maintains recency: head.next is MRU, tail.prev is LRU for tie-breaking.
type lfuList struct {
	head *lfuNode
	tail *lfuNode
	len  int
}

func newLFUList() *lfuList {
	head := &lfuNode{}
	tail := &lfuNode{}
	head.next = tail
	tail.prev = head
	return &lfuList{
		head: head,
		tail: tail,
		len:  0,
	}
}

func (l *lfuList) pushFront(n *lfuNode) {
	n.prev = l.head
	n.next = l.head.next
	l.head.next.prev = n
	l.head.next = n
	l.len++
}

func (l *lfuList) remove(n *lfuNode) {
	n.prev.next = n.next
	n.next.prev = n.prev
	n.prev = nil
	n.next = nil
	l.len--
}

func (l *lfuList) popTail() *lfuNode {
	if l.len == 0 {
		return nil
	}
	lru := l.tail.prev
	l.remove(lru)
	return lru
}

func (l *lfuList) isEmpty() bool {
	return l.len == 0
}

// lfuCache implements the LFU (Least Frequently Used) eviction policy with LRU tie-breaking and lazy TTL expiration.
type lfuCache struct {
	capacity    int
	items       map[string]*lfuNode
	freqBuckets map[int]*lfuList
	minFreq     int
}

func newLFUCache(capacity int) *lfuCache {
	return &lfuCache{
		capacity:    capacity,
		items:       make(map[string]*lfuNode, capacity),
		freqBuckets: make(map[int]*lfuList),
		minFreq:     0,
	}
}

func (c *lfuCache) incrementFreq(n *lfuNode) {
	oldFreq := n.freq
	oldBucket := c.freqBuckets[oldFreq]
	oldBucket.remove(n)
	if oldBucket.isEmpty() {
		delete(c.freqBuckets, oldFreq)
		if c.minFreq == oldFreq {
			c.minFreq = oldFreq + 1
		}
	}

	n.freq++
	newFreq := n.freq
	newBucket, ok := c.freqBuckets[newFreq]
	if !ok {
		newBucket = newLFUList()
		c.freqBuckets[newFreq] = newBucket
	}
	newBucket.pushFront(n)
}

func (c *lfuCache) set(key string, value string, expiresAt time.Time) {
	if n, ok := c.items[key]; ok {
		n.value = value
		n.expiresAt = expiresAt
		c.incrementFreq(n)
		return
	}

	if len(c.items) >= c.capacity {
		c.evict()
	}

	newNode := &lfuNode{
		key:       key,
		value:     value,
		freq:      1,
		expiresAt: expiresAt,
	}
	c.items[key] = newNode
	b1, ok := c.freqBuckets[1]
	if !ok {
		b1 = newLFUList()
		c.freqBuckets[1] = b1
	}
	b1.pushFront(newNode)
	c.minFreq = 1
}

func (c *lfuCache) evict() {
	bucket, ok := c.freqBuckets[c.minFreq]
	if !ok || bucket.isEmpty() {
		return
	}
	evicted := bucket.popTail()
	if evicted != nil {
		delete(c.items, evicted.key)
	}
	if bucket.isEmpty() {
		delete(c.freqBuckets, c.minFreq)
	}
}

func (c *lfuCache) removeNode(n *lfuNode) {
	delete(c.items, n.key)
	bucket := c.freqBuckets[n.freq]
	if bucket != nil {
		bucket.remove(n)
		if bucket.isEmpty() {
			delete(c.freqBuckets, n.freq)
			if c.minFreq == n.freq {
				if len(c.items) == 0 {
					c.minFreq = 0
				} else {
					min := 0
					for f := range c.freqBuckets {
						if min == 0 || f < min {
							min = f
						}
					}
					c.minFreq = min
				}
			}
		}
	}
}

func (c *lfuCache) get(key string) (string, bool) {
	n, ok := c.items[key]
	if !ok {
		return "", false
	}

	if n.isExpired(time.Now()) {
		c.removeNode(n)
		return "", false
	}

	c.incrementFreq(n)
	return n.value, true
}

func (c *lfuCache) delete(key string) bool {
	n, ok := c.items[key]
	if !ok {
		return false
	}

	expired := n.isExpired(time.Now())
	c.removeNode(n)
	if expired {
		return false
	}
	return true
}

func (c *lfuCache) size() int {
	return len(c.items)
}

func (c *lfuCache) getFrequency(key string) (int, bool) {
	n, ok := c.items[key]
	if !ok || n.isExpired(time.Now()) {
		return 0, false
	}
	return n.freq, true
}
