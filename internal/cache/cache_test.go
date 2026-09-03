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

func mustNewLFU(t *testing.T, capacity int) *Cache {
	t.Helper()
	c, err := NewLFU(capacity)
	if err != nil {
		t.Fatalf("unexpected error creating LFU cache: %v", err)
	}
	return c
}

// -----------------------------------------------------------------------------
// Phase 1 & Phase 2 LRU Tests (Preserved)
// -----------------------------------------------------------------------------

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

	// Verify doubly linked list integrity
	if c.ContainsInLRUList("B") {
		t.Fatalf("found deleted key 'B' still present in linked list")
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

// Test Unsupported Policy
func TestUnsupportedPolicy(t *testing.T) {
	c, err := NewWithPolicy(5, EvictionPolicy("RANDOM"))
	if c != nil {
		t.Fatalf("expected nil cache for unsupported policy")
	}
	if !errors.Is(err, ErrUnsupportedPolicy) {
		t.Fatalf("expected ErrUnsupportedPolicy, got %v", err)
	}
}

// Test Concurrent Access (LRU)
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

	if c.Size() > capacity {
		t.Fatalf("final size %d exceeded capacity %d", c.Size(), capacity)
	}
}

// -----------------------------------------------------------------------------
// Phase 3 LFU Tests
// -----------------------------------------------------------------------------

// LFU Test 1 — Basic LFU eviction
func TestLFUBasicEviction(t *testing.T) {
	c := mustNewLFU(t, 3)

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")

	// Access frequencies:
	// A: 1 (set) + 2 (gets) = 3
	c.Get("A")
	c.Get("A")

	// C: 1 (set) + 1 (get) = 2
	c.Get("C")

	// B has freq 1 (only set)
	// Now SET D should evict B (lowest frequency)
	c.Set("D", "4")

	if _, ok := c.Get("B"); ok {
		t.Fatalf("expected 'B' to be evicted as least frequently used")
	}

	for _, key := range []string{"A", "C", "D"} {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to exist in LFU cache", key)
		}
	}
}

// LFU Test 2 — Frequency increments
func TestLFUFrequencyIncrements(t *testing.T) {
	c := mustNewLFU(t, 3)

	c.Set("A", "100")
	freq, ok := c.GetFrequency("A")
	if !ok || freq != 1 {
		t.Fatalf("expected initial frequency 1, got %d (ok=%v)", freq, ok)
	}

	c.Get("A")
	c.Get("A")
	c.Get("A")

	freq, ok = c.GetFrequency("A")
	if !ok || freq != 4 {
		t.Fatalf("expected frequency 4 after 3 GETs, got %d", freq)
	}
}

// LFU Test 3 — Lowest frequency is evicted
func TestLFULowestFrequencyEvicted(t *testing.T) {
	c := mustNewLFU(t, 4)

	c.Set("W", "1") // freq 1
	c.Set("X", "2") // freq 1 -> 2
	c.Get("X")
	c.Set("Y", "3") // freq 1 -> 3
	c.Get("Y")
	c.Get("Y")
	c.Set("Z", "4") // freq 1 -> 4
	c.Get("Z")
	c.Get("Z")
	c.Get("Z")

	// Current frequencies: W: 1, X: 2, Y: 3, Z: 4.
	// Insert new item: W (freq 1) must be evicted.
	c.Set("NewKey", "5")

	if _, ok := c.Get("W"); ok {
		t.Fatalf("expected lowest frequency key 'W' to be evicted")
	}

	for _, key := range []string{"X", "Y", "Z", "NewKey"} {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to remain in cache", key)
		}
	}
}

// LFU Test 4 — LFU tie breaking (LRU among equal frequencies)
func TestLFUTieBreaking(t *testing.T) {
	c := mustNewLFU(t, 3)

	// Set A, B, C (all initially freq 1)
	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")

	// Increment A and B to freq 2
	c.Get("A") // A freq = 2 (used earlier)
	c.Get("B") // B freq = 2 (used more recently)
	// C remains freq 1

	// Insert D -> C has freq 1, so C is evicted without tie
	c.Set("D", "4")
	if _, ok := c.Get("C"); ok {
		t.Fatalf("expected 'C' to be evicted as lowest frequency")
	}

	// Now D has freq 1. Increment D to freq 2.
	c.Get("D") // D freq = 2

	// Now all keys A, B, D have frequency 2!
	// Recency order inside freq 2 bucket:
	// A was accessed first, then B, then D.
	// So A is the least recently used in freq 2!
	// Inserting E (freq 1) must evict A via LRU tie-breaking!
	c.Set("E", "5")

	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected 'A' to be evicted as LRU among items with equal frequency 2")
	}

	for _, key := range []string{"B", "D", "E"} {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to remain in cache", key)
		}
	}
}

// LFU Test 5 — Updating an existing key
func TestLFUUpdateExistingKey(t *testing.T) {
	c := mustNewLFU(t, 3)

	c.Set("A", "100") // freq 1
	c.Get("A")        // freq 2
	c.Set("A", "200") // freq 3 (update treated as access)

	val, ok := c.Get("A") // freq 4
	if !ok || val != "200" {
		t.Fatalf("expected value '200', got %q", val)
	}

	freq, ok := c.GetFrequency("A")
	if !ok || freq != 4 {
		t.Fatalf("expected frequency 4 after update and GET, got %d", freq)
	}
}

// LFU Test 6 — Delete removes key and metadata
func TestLFUDeleteRemovesMetadata(t *testing.T) {
	c := mustNewLFU(t, 3)

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3")

	deleted := c.Delete("B")
	if !deleted {
		t.Fatalf("expected Delete('B') to return true")
	}

	if _, ok := c.Get("B"); ok {
		t.Fatalf("expected 'B' to be deleted from cache")
	}

	if _, ok := c.GetFrequency("B"); ok {
		t.Fatalf("expected 'B' frequency metadata to be deleted")
	}

	if c.Size() != 2 {
		t.Fatalf("expected size 2 after deleting 'B', got %d", c.Size())
	}

	// Deleting a non-existent key returns false
	if c.Delete("unknown") {
		t.Fatalf("expected Delete('unknown') to return false")
	}
}

// LFU Test 7 — Capacity
func TestLFUCapacity(t *testing.T) {
	capacity := 4
	c := mustNewLFU(t, capacity)

	for i := 0; i < 40; i++ {
		key := fmt.Sprintf("key-%d", i)
		c.Set(key, fmt.Sprintf("val-%d", i))
		if i%3 == 0 {
			c.Get(key) // increase frequency of some keys
		}
		if c.Size() > capacity {
			t.Fatalf("size %d exceeded capacity %d", c.Size(), capacity)
		}
	}

	if c.Size() != capacity {
		t.Fatalf("expected size to equal capacity %d, got %d", capacity, c.Size())
	}
}

// LFU Test 8 — Capacity = 1
func TestLFUCapacityOne(t *testing.T) {
	c := mustNewLFU(t, 1)

	c.Set("A", "1")
	val, ok := c.Get("A")
	if !ok || val != "1" {
		t.Fatalf("expected Get('A') to return '1', true")
	}

	// A has freq 2 now. Set B: A should be evicted.
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

	// Update existing key
	c.Set("B", "20")
	val, ok = c.Get("B")
	if !ok || val != "20" {
		t.Fatalf("expected Get('B') to return '20', true")
	}

	// Delete
	if !c.Delete("B") {
		t.Fatalf("expected Delete('B') to return true")
	}
	if c.Size() != 0 {
		t.Fatalf("expected size 0, got %d", c.Size())
	}
}

// LFU Test 9 — Concurrent access
func TestLFUConcurrentAccess(t *testing.T) {
	capacity := 20
	c := mustNewLFU(t, capacity)
	var wg sync.WaitGroup

	numGoroutines := 50
	operationsPerGoroutine := 200

	// Spawn writers
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

	// Spawn readers
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

	// Spawn deleters
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

	// Spawn size and frequency checkers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				s := c.Size()
				if s > capacity {
					t.Errorf("LFU size %d exceeded capacity %d during concurrency", s, capacity)
				}
				key := fmt.Sprintf("key-%d", j%40)
				_, _ = c.GetFrequency(key)
			}
		}()
	}

	wg.Wait()

	if c.Size() > capacity {
		t.Fatalf("final LFU size %d exceeded capacity %d", c.Size(), capacity)
	}
}

// -----------------------------------------------------------------------------
// Phase 4 2Q Tests
// -----------------------------------------------------------------------------

func mustNew2Q(t *testing.T, capacity int) *Cache {
	t.Helper()
	c, err := New2Q(capacity)
	if err != nil {
		t.Fatalf("unexpected error creating 2Q cache: %v", err)
	}
	return c
}

func mustNew2QWithCapacities(t *testing.T, capacity, a1Cap, amCap int) *Cache {
	t.Helper()
	c, err := New2QWithCapacities(capacity, a1Cap, amCap)
	if err != nil {
		t.Fatalf("unexpected error creating 2Q cache with capacities: %v", err)
	}
	return c
}

// 2Q Test 1 — New entries enter A1
func Test2QNewEntriesEnterA1(t *testing.T) {
	c := mustNew2Q(t, 4)

	c.Set("A", "100")

	if !c.IsInA1("A") {
		t.Fatalf("expected 'A' to be in queue A1")
	}
	if c.IsInAm("A") {
		t.Fatalf("expected 'A' NOT to be in queue Am")
	}
	if c.A1Size() != 1 {
		t.Fatalf("expected A1 size 1, got %d", c.A1Size())
	}
	if c.AmSize() != 0 {
		t.Fatalf("expected Am size 0, got %d", c.AmSize())
	}
}

// 2Q Test 2 — Second access promotes to Am
func Test2QSecondAccessPromotesToAm(t *testing.T) {
	c := mustNew2Q(t, 4)

	c.Set("A", "100")
	val, ok := c.Get("A")
	if !ok || val != "100" {
		t.Fatalf("expected Get('A') to return '100', true")
	}

	if c.IsInA1("A") {
		t.Fatalf("expected 'A' to no longer be in queue A1 after second access")
	}
	if !c.IsInAm("A") {
		t.Fatalf("expected 'A' to be promoted to queue Am")
	}
	if c.A1Size() != 0 {
		t.Fatalf("expected A1 size 0, got %d", c.A1Size())
	}
	if c.AmSize() != 1 {
		t.Fatalf("expected Am size 1, got %d", c.AmSize())
	}
}

// 2Q Test 3 — One-time entries are evicted from A1
func Test2QOneTimeEntriesEvictedFromA1(t *testing.T) {
	// Total capacity = 4, A1 capacity = 2, Am capacity = 2
	c := mustNew2QWithCapacities(t, 4, 2, 2)

	c.Set("A", "1")
	c.Set("B", "2")
	c.Set("C", "3") // A1 capacity is 2, so oldest entry 'A' must be evicted

	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected oldest A1 entry 'A' to be evicted")
	}
	if val, ok := c.Get("B"); !ok || val != "2" {
		t.Fatalf("expected 'B' to remain in cache")
	}
	if val, ok := c.Get("C"); !ok || val != "3" {
		t.Fatalf("expected 'C' to remain in cache")
	}
}

// 2Q Test 4 — Frequently reused item survives
func Test2QFrequentlyReusedItemSurvives(t *testing.T) {
	// Total capacity = 4, A1 capacity = 2, Am capacity = 2
	c := mustNew2QWithCapacities(t, 4, 2, 2)

	c.Set("A", "1")
	c.Get("A") // Promotes A to Am

	if !c.IsInAm("A") {
		t.Fatalf("expected 'A' to be promoted to Am")
	}

	// Insert items into A1 that exceed A1 capacity
	c.Set("B", "2")
	c.Set("C", "3")
	c.Set("D", "4") // Evicts B from A1

	// A in Am must survive!
	val, ok := c.Get("A")
	if !ok || val != "1" {
		t.Fatalf("expected promoted item 'A' in Am to survive A1 churn")
	}
	if !c.IsInAm("A") {
		t.Fatalf("expected 'A' to remain in Am")
	}
}

// 2Q Test 5 — Am uses LRU behavior
func Test2QAmUsesLRUBehavior(t *testing.T) {
	// Total capacity = 3, A1 capacity = 1, Am capacity = 2
	c := mustNew2QWithCapacities(t, 3, 1, 2)

	// Promote A and B to Am
	c.Set("A", "1")
	c.Get("A") // A in Am

	c.Set("B", "2")
	c.Get("B") // B in Am (MRU in Am)

	// Access A again -> A becomes MRU in Am, B becomes LRU in Am
	c.Get("A")

	// Insert C and promote C to Am -> Am is at capacity (2), so LRU (B) must be evicted from Am
	c.Set("C", "3")
	c.Get("C") // Promotes C to Am

	if _, ok := c.Get("B"); ok {
		t.Fatalf("expected 'B' to be evicted from Am as least recently used")
	}

	for _, key := range []string{"A", "C"} {
		if val, ok := c.Get(key); !ok {
			t.Fatalf("expected key %q to survive in Am", key)
		} else if key == "A" && val != "1" {
			t.Fatalf("expected 'A' value '1', got %q", val)
		}
	}
}

// 2Q Test 6 — SET existing A1 entry
func Test2QSetExistingA1Entry(t *testing.T) {
	c := mustNew2Q(t, 4)

	c.Set("A", "old_val")
	if !c.IsInA1("A") {
		t.Fatalf("expected 'A' in A1 initially")
	}

	// SET of existing key counts as access: updates value and promotes to Am
	c.Set("A", "new_val")

	if c.IsInA1("A") {
		t.Fatalf("expected 'A' to be promoted out of A1 after update SET")
	}
	if !c.IsInAm("A") {
		t.Fatalf("expected 'A' to be in Am after update SET")
	}

	val, ok := c.Get("A")
	if !ok || val != "new_val" {
		t.Fatalf("expected value 'new_val', got %q", val)
	}
}

// 2Q Test 7 — SET existing Am entry
func Test2QSetExistingAmEntry(t *testing.T) {
	// Total capacity = 3, A1 = 1, Am = 2
	c := mustNew2QWithCapacities(t, 3, 1, 2)

	c.Set("A", "1")
	c.Get("A") // A in Am

	c.Set("B", "2")
	c.Get("B") // B in Am (MRU), A is LRU in Am

	// Update existing Am entry A: updates value and moves A to MRU
	c.Set("A", "100")

	// Now promote C into Am: should evict B (the LRU item), while A survives
	c.Set("C", "3")
	c.Get("C")

	if _, ok := c.Get("B"); ok {
		t.Fatalf("expected 'B' to be evicted from Am")
	}

	val, ok := c.Get("A")
	if !ok || val != "100" {
		t.Fatalf("expected 'A' to have updated value '100', got %q", val)
	}
}

// 2Q Test 8 — DELETE from A1
func Test2QDeleteFromA1(t *testing.T) {
	c := mustNew2Q(t, 4)

	c.Set("A", "1")
	if !c.IsInA1("A") {
		t.Fatalf("expected 'A' in A1")
	}

	deleted := c.Delete("A")
	if !deleted {
		t.Fatalf("expected Delete('A') to return true")
	}

	if c.IsInA1("A") {
		t.Fatalf("expected 'A' to be removed from A1")
	}
	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected Get('A') to return false")
	}
	if c.Size() != 0 {
		t.Fatalf("expected size 0, got %d", c.Size())
	}
}

// 2Q Test 9 — DELETE from Am
func Test2QDeleteFromAm(t *testing.T) {
	c := mustNew2Q(t, 4)

	c.Set("A", "1")
	c.Get("A") // Promote to Am

	if !c.IsInAm("A") {
		t.Fatalf("expected 'A' in Am")
	}

	deleted := c.Delete("A")
	if !deleted {
		t.Fatalf("expected Delete('A') to return true")
	}

	if c.IsInAm("A") {
		t.Fatalf("expected 'A' to be removed from Am")
	}
	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected Get('A') to return false")
	}
	if c.Size() != 0 {
		t.Fatalf("expected size 0, got %d", c.Size())
	}
}

// 2Q Test 10 — Capacity
func Test2QCapacity(t *testing.T) {
	capacity := 6
	c := mustNew2Q(t, capacity)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key-%d", i)
		c.Set(key, fmt.Sprintf("val-%d", i))
		if i%2 == 0 {
			c.Get(key) // promotes some keys to Am
		}
		if c.A1Size()+c.AmSize() > capacity {
			t.Fatalf("A1 size (%d) + Am size (%d) exceeded capacity (%d)", c.A1Size(), c.AmSize(), capacity)
		}
		if c.Size() > capacity {
			t.Fatalf("Size() %d exceeded capacity %d", c.Size(), capacity)
		}
	}
}

// 2Q Test 11 — Capacity = 1
func Test2QCapacityOne(t *testing.T) {
	c := mustNew2Q(t, 1)

	c.Set("A", "1")
	if c.Size() != 1 {
		t.Fatalf("expected size 1, got %d", c.Size())
	}
	if !c.IsInA1("A") {
		t.Fatalf("expected 'A' in A1")
	}

	// GET promotes A to Am
	val, ok := c.Get("A")
	if !ok || val != "1" {
		t.Fatalf("expected Get('A') to succeed")
	}
	if !c.IsInAm("A") {
		t.Fatalf("expected 'A' promoted to Am")
	}
	if c.Size() != 1 {
		t.Fatalf("expected size 1, got %d", c.Size())
	}

	// SET B: must evict A from Am to respect capacity = 1
	c.Set("B", "2")
	if c.Size() != 1 {
		t.Fatalf("expected size 1 after setting B, got %d", c.Size())
	}
	if _, ok := c.Get("A"); ok {
		t.Fatalf("expected 'A' to be evicted when capacity is 1")
	}
	if !c.IsInA1("B") {
		t.Fatalf("expected 'B' in A1")
	}
}

// 2Q Test 12 — Concurrent access
func Test2QConcurrentAccess(t *testing.T) {
	capacity := 20
	c := mustNew2Q(t, capacity)
	var wg sync.WaitGroup

	numGoroutines := 50
	operationsPerGoroutine := 200

	// Spawn writers
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

	// Spawn readers
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

	// Spawn deleters
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

	// Spawn queue inspectors
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				s := c.Size()
				if s > capacity {
					t.Errorf("2Q size %d exceeded capacity %d during concurrency", s, capacity)
				}
				key := fmt.Sprintf("key-%d", j%40)
				_ = c.IsInA1(key)
				_ = c.IsInAm(key)
				_ = c.A1Size()
				_ = c.AmSize()
			}
		}()
	}

	wg.Wait()

	if c.Size() > capacity {
		t.Fatalf("final 2Q size %d exceeded capacity %d", c.Size(), capacity)
	}
}
