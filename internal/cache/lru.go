package cache

import "time"

// lruNode represents a doubly linked list node storing key, value, and optional expiration for LRU.
type lruNode struct {
	key       string
	value     string
	expiresAt time.Time
	prev      *lruNode
	next      *lruNode
}

func (n *lruNode) isExpired(now time.Time) bool {
	return !n.expiresAt.IsZero() && !now.Before(n.expiresAt)
}

// lruCache implements the LRU (Least Recently Used) eviction policy with lazy TTL expiration.
type lruCache struct {
	capacity int
	items    map[string]*lruNode
	head     *lruNode // sentinel head: head.next points to most recently used (MRU)
	tail     *lruNode // sentinel tail: tail.prev points to least recently used (LRU)
}

func newLRUCache(capacity int) *lruCache {
	head := &lruNode{}
	tail := &lruNode{}
	head.next = tail
	tail.prev = head

	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*lruNode, capacity),
		head:     head,
		tail:     tail,
	}
}

func (l *lruCache) addToFront(n *lruNode) {
	n.prev = l.head
	n.next = l.head.next
	l.head.next.prev = n
	l.head.next = n
}

func (l *lruCache) removeNode(n *lruNode) {
	n.prev.next = n.next
	n.next.prev = n.prev
	n.prev = nil
	n.next = nil
}

func (l *lruCache) moveToFront(n *lruNode) {
	l.removeNode(n)
	l.addToFront(n)
}

func (l *lruCache) removeLRU() *lruNode {
	if l.tail.prev == l.head {
		return nil
	}
	lru := l.tail.prev
	l.removeNode(lru)
	return lru
}

func (l *lruCache) set(key string, value string, expiresAt time.Time) bool {
	if n, ok := l.items[key]; ok {
		n.value = value
		n.expiresAt = expiresAt
		l.moveToFront(n)
		return false
	}

	var evicted bool
	if len(l.items) >= l.capacity {
		if lru := l.removeLRU(); lru != nil {
			delete(l.items, lru.key)
			evicted = true
		}
	}

	newNode := &lruNode{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	}
	l.addToFront(newNode)
	l.items[key] = newNode
	return evicted
}

func (l *lruCache) get(key string) (string, bool, bool) {
	n, ok := l.items[key]
	if !ok {
		return "", false, false
	}

	if n.isExpired(time.Now()) {
		l.removeNode(n)
		delete(l.items, key)
		return "", false, true
	}

	l.moveToFront(n)
	return n.value, true, false
}

func (l *lruCache) delete(key string) (bool, bool) {
	n, ok := l.items[key]
	if !ok {
		return false, false
	}

	expired := n.isExpired(time.Now())
	l.removeNode(n)
	delete(l.items, key)
	if expired {
		return false, true
	}
	return true, false
}

func (l *lruCache) size() int {
	return len(l.items)
}

func (l *lruCache) containsInList(key string) bool {
	for curr := l.head.next; curr != l.tail; curr = curr.next {
		if curr.key == key {
			return true
		}
	}
	return false
}
