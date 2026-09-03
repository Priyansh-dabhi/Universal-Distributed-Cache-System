package cache

// lruNode represents a doubly linked list node storing key and value for LRU.
type lruNode struct {
	key   string
	value string
	prev  *lruNode
	next  *lruNode
}

// lruCache implements the LRU (Least Recently Used) eviction policy.
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

func (l *lruCache) set(key string, value string) {
	if n, ok := l.items[key]; ok {
		n.value = value
		l.moveToFront(n)
		return
	}

	if len(l.items) >= l.capacity {
		if lru := l.removeLRU(); lru != nil {
			delete(l.items, lru.key)
		}
	}

	newNode := &lruNode{
		key:   key,
		value: value,
	}
	l.addToFront(newNode)
	l.items[key] = newNode
}

func (l *lruCache) get(key string) (string, bool) {
	n, ok := l.items[key]
	if !ok {
		return "", false
	}
	l.moveToFront(n)
	return n.value, true
}

func (l *lruCache) delete(key string) bool {
	n, ok := l.items[key]
	if !ok {
		return false
	}
	l.removeNode(n)
	delete(l.items, key)
	return true
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
