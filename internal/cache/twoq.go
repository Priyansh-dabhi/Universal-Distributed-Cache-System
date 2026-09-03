package cache

import "time"

type queueType int

const (
	queueA1 queueType = 1
	queueAm queueType = 2
)

// twoQNode represents an entry in the 2Q cache with key, value, queue type, and expiration.
type twoQNode struct {
	key       string
	value     string
	qType     queueType
	expiresAt time.Time
	prev      *twoQNode
	next      *twoQNode
}

func (n *twoQNode) isExpired(now time.Time) bool {
	return !n.expiresAt.IsZero() && !now.Before(n.expiresAt)
}

// twoQList is a doubly linked list of twoQNodes with sentinel head and tail.
type twoQList struct {
	head *twoQNode
	tail *twoQNode
	len  int
}

func newTwoQList() *twoQList {
	head := &twoQNode{}
	tail := &twoQNode{}
	head.next = tail
	tail.prev = head
	return &twoQList{
		head: head,
		tail: tail,
		len:  0,
	}
}

func (l *twoQList) pushFront(n *twoQNode) {
	n.prev = l.head
	n.next = l.head.next
	l.head.next.prev = n
	l.head.next = n
	l.len++
}

func (l *twoQList) remove(n *twoQNode) {
	n.prev.next = n.next
	n.next.prev = n.prev
	n.prev = nil
	n.next = nil
	l.len--
}

func (l *twoQList) moveToFront(n *twoQNode) {
	l.remove(n)
	l.pushFront(n)
}

func (l *twoQList) popTail() *twoQNode {
	if l.len == 0 {
		return nil
	}
	oldest := l.tail.prev
	l.remove(oldest)
	return oldest
}

// twoQCache implements the 2Q (Two-Queue) eviction policy with lazy TTL expiration.
type twoQCache struct {
	capacity int
	a1Cap    int
	amCap    int
	items    map[string]*twoQNode
	a1       *twoQList // FIFO for recent entries
	am       *twoQList // LRU for promoted/frequent entries
}

func newTwoQCache(capacity, a1Cap, amCap int) *twoQCache {
	return &twoQCache{
		capacity: capacity,
		a1Cap:    a1Cap,
		amCap:    amCap,
		items:    make(map[string]*twoQNode, capacity),
		a1:       newTwoQList(),
		am:       newTwoQList(),
	}
}

func (c *twoQCache) set(key string, value string, expiresAt time.Time) {
	if node, ok := c.items[key]; ok {
		node.value = value
		node.expiresAt = expiresAt
		if node.qType == queueA1 {
			// Updating an existing A1 key counts as an access: promote to Am
			c.promoteToAm(node)
		} else {
			// Updating an existing Am key: move to MRU
			c.am.moveToFront(node)
		}
		return
	}

	// New key: insert into A1.
	// If A1 is full, evict oldest from A1
	if c.a1.len >= c.a1Cap {
		c.evictA1()
	}

	// If overall cache capacity is reached, evict to maintain bounds
	if len(c.items) >= c.capacity {
		if c.a1.len > 0 {
			c.evictA1()
		} else if c.am.len > 0 {
			c.evictAm()
		}
	}

	newNode := &twoQNode{
		key:       key,
		value:     value,
		qType:     queueA1,
		expiresAt: expiresAt,
	}
	c.items[key] = newNode
	c.a1.pushFront(newNode)
}

func (c *twoQCache) get(key string) (string, bool) {
	node, ok := c.items[key]
	if !ok {
		return "", false
	}

	if node.isExpired(time.Now()) {
		c.removeNode(node)
		return "", false
	}

	if node.qType == queueA1 {
		// Second access: promote from A1 to Am
		c.promoteToAm(node)
	} else {
		// Already in Am: move to MRU
		c.am.moveToFront(node)
	}

	return node.value, true
}

func (c *twoQCache) promoteToAm(node *twoQNode) {
	c.a1.remove(node)

	// Ensure Am does not exceed its capacity or overall capacity
	if c.am.len >= c.amCap || len(c.items) > c.capacity {
		c.evictAm()
	}

	node.qType = queueAm
	c.am.pushFront(node)
}

func (c *twoQCache) evictA1() {
	if oldest := c.a1.popTail(); oldest != nil {
		delete(c.items, oldest.key)
	}
}

func (c *twoQCache) evictAm() {
	if lru := c.am.popTail(); lru != nil {
		delete(c.items, lru.key)
	}
}

func (c *twoQCache) removeNode(node *twoQNode) {
	delete(c.items, node.key)
	if node.qType == queueA1 {
		c.a1.remove(node)
	} else {
		c.am.remove(node)
	}
}

func (c *twoQCache) delete(key string) bool {
	node, ok := c.items[key]
	if !ok {
		return false
	}

	expired := node.isExpired(time.Now())
	c.removeNode(node)
	if expired {
		return false
	}
	return true
}

func (c *twoQCache) size() int {
	return len(c.items)
}

func (c *twoQCache) isInA1(key string) bool {
	node, ok := c.items[key]
	return ok && node.qType == queueA1 && !node.isExpired(time.Now())
}

func (c *twoQCache) isInAm(key string) bool {
	node, ok := c.items[key]
	return ok && node.qType == queueAm && !node.isExpired(time.Now())
}

func (c *twoQCache) a1Size() int {
	return c.a1.len
}

func (c *twoQCache) amSize() int {
	return c.am.len
}
