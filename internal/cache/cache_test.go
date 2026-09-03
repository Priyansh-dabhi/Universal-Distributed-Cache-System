package cache

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func mustNew(t *testing.T, capacity int) *Cache {
	t.Helper()
	c, err := New(capacity)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %v", err)
	}
	return c
}

// Test 1 — Set and Get
func TestSetAndGet(t *testing.T) {
	c := mustNew(t, 10)

	c.Set("A", "100")
	val, ok := c.Get("A")
	if !ok {
		t.Fatalf("expected key 'A' to exist")
	}
	if val != "100" {
		t.Fatalf("expected value '100', got %q", val)
	}
}

// Test 2 — Get missing key
func TestGetMissingKey(t *testing.T) {
	c := mustNew(t, 10)

	val, ok := c.Get("missing")
	if ok {
		t.Fatalf("expected missing key to return ok=false, got val=%q", val)
	}
	if val != "" {
		t.Fatalf("expected missing key to return empty string, got %q", val)
	}
}

// Test 3 — Update existing key
func TestUpdateExistingKey(t *testing.T) {
	c := mustNew(t, 10)

	c.Set("A", "100")
	c.Set("A", "200")

	val, ok := c.Get("A")
	if !ok {
		t.Fatalf("expected key 'A' to exist after update")
	}
	if val != "200" {
		t.Fatalf("expected updated value '200', got %q", val)
	}
}

// Test 4 — Delete existing key
func TestDeleteExistingKey(t *testing.T) {
	c := mustNew(t, 10)

	c.Set("A", "100")
	deleted := c.Delete("A")
	if !deleted {
		t.Fatalf("expected Delete('A') to return true")
	}

	val, ok := c.Get("A")
	if ok {
		t.Fatalf("expected key 'A' to be deleted, but found val=%q", val)
	}
}

// Test 5 — Delete missing key
func TestDeleteMissingKey(t *testing.T) {
	c := mustNew(t, 10)

	deleted := c.Delete("non_existent")
	if deleted {
		t.Fatalf("expected Delete('non_existent') to return false")
	}
}

// Test 6 — Size
func TestSize(t *testing.T) {
	c := mustNew(t, 10)

	if size := c.Size(); size != 0 {
		t.Fatalf("expected initial size 0, got %d", size)
	}

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")

	if size := c.Size(); size != 3 {
		t.Fatalf("expected size 3 after setting 3 keys, got %d", size)
	}

	// Updating an existing key should not change size
	c.Set("A", "10")
	if size := c.Size(); size != 3 {
		t.Fatalf("expected size 3 after updating key 'A', got %d", size)
	}

	deleted := c.Delete("B")
	if !deleted {
		t.Fatalf("expected Delete('B') to succeed")
	}

	if size := c.Size(); size != 2 {
		t.Fatalf("expected size 2 after deleting key 'B', got %d", size)
	}

	// Deleting a non-existent key should not change size
	c.Delete("unknown")
	if size := c.Size(); size != 2 {
		t.Fatalf("expected size 2 after deleting unknown key, got %d", size)
	}
}

// Test 7 — Empty string value
func TestEmptyStringValue(t *testing.T) {
	c := mustNew(t, 10)

	c.Set("key", "")
	val, ok := c.Get("key")
	if !ok {
		t.Fatalf("expected key with empty string value to return ok=true")
	}
	if val != "" {
		t.Fatalf("expected val to be empty string, got %q", val)
	}
}

// LRU Test 1 — Evicts least recently used
func TestLRUEviction(t *testing.T) {
	c := mustNew(t, 3)

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")
	c.Set("D", "4")

	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected 'A' to be evicted as least recently used")
	}

	for _, key := range []string{"B", "C", "D"} {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to exist in cache", key)
		}
	}
}

// LRU Test 2 — GET changes recency
func TestLRUGetChangesRecency(t *testing.T) {
	c := mustNew(t, 3)

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")

	// Access A, making it MRU (recency order: A -> C -> B)
	val, ok := c.Get("A")
	if !ok || val != "1" {
		t.Fatalf("expected Get('A') to return '1', true")
	}

	// Setting D should now evict B (since B is now LRU)
	c.Set("D", "4")

	if _, ok := c.Get("B"); ok {
		t.Fatalf("expected 'B' to be evicted as LRU after 'A' was accessed")
	}

	for _, key := range []string{"A", "C", "D"} {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to exist in cache", key)
		}
	}
}

// LRU Test 3 — Updating a key changes recency
func TestLRUUpdateChangesRecency(t *testing.T) {
	c := mustNew(t, 3)

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")

	// Update A with a new value (recency order: A -> C -> B)
	c.Set("A", "100")

	// Setting D should now evict B
	c.Set("D", "4")

	if _, ok := c.Get("B"); ok {
		t.Fatalf("expected 'B' to be evicted as LRU after 'A' was updated")
	}

	for _, key := range []string{"A", "C", "D"} {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to exist in cache", key)
		}
	}
}

// LRU Test 4 — GET returns updated value
func TestLRUGetReturnsUpdatedValue(t *testing.T) {
	c := mustNew(t, 3)

	c.Set("A", "old_value")
	c.Set("A", "new_value")

	val, ok := c.Get("A")
	if !ok {
		t.Fatalf("expected key 'A' to exist")
	}
	if val != "new_value" {
		t.Fatalf("expected updated value 'new_value', got %q", val)
	}
}

// LRU Test 5 — DELETE removes key and LRU metadata
func TestLRUDeleteRemovesMetadata(t *testing.T) {
	c := mustNew(t, 3)

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")

	deleted := c.Delete("B")
	if !deleted {
		t.Fatalf("expected Delete('B') to return true")
	}

	if _, ok := c.Get("B"); ok {
		t.Fatalf("expected 'B' to be deleted from map")
	}

	if c.Size() != 2 {
		t.Fatalf("expected size 2 after deleting 'B', got %d", c.Size())
	}

	// Verify doubly linked list integrity by traversing head to tail
	for curr := c.head.next; curr != c.tail; curr = curr.next {
		if curr.key == "B" {
			t.Fatalf("found deleted key 'B' still present in linked list")
		}
	}

	// Add new elements to trigger eviction and ensure no panic or corruption
	c.Set("D", "4")
	c.Set("E", "5") // Should evict A (order was C -> A, added D -> D -> C -> A, added E -> evicts A)

	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected 'A' to be evicted after adding D and E")
	}
}

// LRU Test 6 — Size never exceeds capacity
func TestLRUSizeNeverExceedsCapacity(t *testing.T) {
	capacity := 5
	c := mustNew(t, capacity)

	for i := 0; i < 50; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i))
		if c.Size() > capacity {
			t.Fatalf("cache size %d exceeded capacity %d", c.Size(), capacity)
		}
	}

	if c.Size() != capacity {
		t.Fatalf("expected cache size to be exactly capacity %d, got %d", capacity, c.Size())
	}
}

// LRU Test 7 — Capacity of one
func TestLRUCapacityOne(t *testing.T) {
	c := mustNew(t, 1)

	c.Set("A", "1")
	val, ok := c.Get("A")
	if !ok || val != "1" {
		t.Fatalf("expected Get('A') to return '1', true")
	}
	if c.Size() != 1 {
		t.Fatalf("expected size 1, got %d", c.Size())
	}

	c.Set("B", "2")
	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected 'A' to be evicted with capacity 1")
	}
	val, ok = c.Get("B")
	if !ok || val != "2" {
		t.Fatalf("expected Get('B') to return '2', true")
	}
	if c.Size() != 1 {
		t.Fatalf("expected size 1, got %d", c.Size())
	}

	// Update existing key on capacity 1
	c.Set("B", "20")
	val, ok = c.Get("B")
	if !ok || val != "20" {
		t.Fatalf("expected Get('B') to return '20', true")
	}
	if c.Size() != 1 {
		t.Fatalf("expected size 1 after update, got %d", c.Size())
	}

	// Delete on capacity 1
	deleted := c.Delete("B")
	if !deleted {
		t.Fatalf("expected Delete('B') to return true")
	}
	if c.Size() != 0 {
		t.Fatalf("expected size 0 after deletion, got %d", c.Size())
	}
}

// Test Invalid Capacity
func TestInvalidCapacity(t *testing.T) {
	for _, cap := range []int{0, -1, -100} {
		c, err := New(cap)
		if c != nil {
			t.Fatalf("expected nil cache for invalid capacity %d", cap)
		}
		if !errors.Is(err, ErrInvalidCapacity) {
			t.Fatalf("expected ErrInvalidCapacity for capacity %d, got %v", cap, err)
		}
	}
}

// Test 8 — Concurrent access with LRU operations
func TestConcurrentAccess(t *testing.T) {
	capacity := 20
	c := mustNew(t, capacity)
	var wg sync.WaitGroup

	numGoroutines := 50
	operationsPerGoroutine := 200

	// Spawn writer goroutines (SET new & existing keys)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", j%40)
				val := fmt.Sprintf("val-%d-%d", workerID, j)
				c.Set(key, val)
			}
		}(i)
	}

	// Spawn reader goroutines (GET modifies recency)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", j%40)
				_, _ = c.Get(key)
			}
		}()
	}

	// Spawn deleter goroutines (DELETE keys and nodes)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", j%40)
				_ = c.Delete(key)
			}
		}()
	}

	// Spawn size checker goroutines (concurrent reads)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				s := c.Size()
				if s > capacity {
					t.Errorf("size %d exceeded capacity %d during concurrent access", s, capacity)
				}
			}
		}()
	}

	wg.Wait()

	// Final sanity check: size must never exceed capacity
	if c.Size() > capacity {
		t.Fatalf("final size %d exceeded capacity %d", c.Size(), capacity)
	}
}
