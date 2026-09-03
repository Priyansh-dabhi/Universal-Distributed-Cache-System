package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// TestMetricsCorrectness verifies that cache operations increment the appropriate metrics counters.
func TestMetricsCorrectness(t *testing.T) {
	c, err := NewLRU(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	m := c.Metrics()

	// 1. Initial State
	if m.Hits() != 0 || m.Misses() != 0 || m.Sets() != 0 || m.Deletes() != 0 || m.Evictions() != 0 || m.Expired() != 0 {
		t.Fatalf("expected all metric counters to start at 0")
	}

	// 2. Miss on non-existent key
	_, ok := c.Get("nonexistent")
	if ok || m.Misses() != 1 || m.Hits() != 0 {
		t.Fatalf("expected 1 miss on nonexistent key")
	}

	// 3. Set keys
	c.Set("k1", "v1")
	c.Set("k2", "v2")
	if m.Sets() != 2 || m.Evictions() != 0 {
		t.Fatalf("expected 2 sets and 0 evictions")
	}

	// 4. Hits
	val, ok := c.Get("k1")
	if !ok || val != "v1" || m.Hits() != 1 || m.Misses() != 1 {
		t.Fatalf("expected 1 hit on k1")
	}

	// Hit rate: 1 hit / (1 hit + 1 miss) = 0.5
	if m.HitRate() != 0.5 {
		t.Fatalf("expected hit rate 0.5, got %f", m.HitRate())
	}

	// 5. Eviction: capacity is 2; adding 3rd item evicts k2 (k1 was accessed recently)
	c.Set("k3", "v3")
	if m.Sets() != 3 || m.Evictions() != 1 {
		t.Fatalf("expected 3 sets and 1 eviction, got sets=%d evictions=%d", m.Sets(), m.Evictions())
	}

	// Verify k2 was evicted
	_, ok = c.Get("k2")
	if ok || m.Misses() != 2 {
		t.Fatalf("expected k2 to be evicted and recorded as a miss")
	}

	// 6. Delete
	deleted := c.Delete("k3")
	if !deleted || m.Deletes() != 1 {
		t.Fatalf("expected delete to succeed and increment deletes counter")
	}

	// 7. Expired key metric
	c.SetWithTTL("expiring-key", "value", 20*time.Millisecond)
	time.Sleep(30 * time.Millisecond)

	_, ok = c.Get("expiring-key")
	if ok || m.Expired() != 1 || m.Misses() != 3 {
		t.Fatalf("expected expired entry to increment expired (%d) and miss (%d)", m.Expired(), m.Misses())
	}
}

// runConcurrentWorkload executes a randomized concurrent workload across multiple workers.
func runConcurrentWorkload(t *testing.T, c *Cache, workers, iterations int) {
	var wg sync.WaitGroup
	keys := []string{"keyA", "keyB", "keyC", "keyD", "keyE", "keyF", "keyG", "keyH"}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for i := 0; i < iterations; i++ {
				key := keys[r.Intn(len(keys))]
				op := r.Intn(4)

				switch op {
				case 0: // GET
					c.Get(key)
				case 1: // SET
					c.Set(key, fmt.Sprintf("val-%d-%d", workerID, i))
				case 2: // SET with TTL
					c.SetWithTTL(key, fmt.Sprintf("val-ttl-%d-%d", workerID, i), time.Duration(r.Intn(50)+1)*time.Millisecond)
				case 3: // DELETE
					c.Delete(key)
				}
			}
		}(w)
	}

	wg.Wait()

	// Verify cache internal invariants
	if c.Size() > c.Capacity() {
		t.Fatalf("size %d exceeds capacity %d after concurrent execution", c.Size(), c.Capacity())
	}

	m := c.Metrics()
	totalOps := m.Hits() + m.Misses() + m.Sets() + m.Deletes()
	if totalOps == 0 {
		t.Fatalf("expected metrics to record operations during concurrent run")
	}
}

func TestCacheConcurrencyLRU(t *testing.T) {
	c, err := NewLRU(5)
	if err != nil {
		t.Fatalf("failed to create LRU cache: %v", err)
	}
	runConcurrentWorkload(t, c, 50, 200)
}

func TestCacheConcurrencyLFU(t *testing.T) {
	c, err := NewLFU(5)
	if err != nil {
		t.Fatalf("failed to create LFU cache: %v", err)
	}
	runConcurrentWorkload(t, c, 50, 200)
}

func TestCacheConcurrency2Q(t *testing.T) {
	c, err := New2Q(6)
	if err != nil {
		t.Fatalf("failed to create 2Q cache: %v", err)
	}
	runConcurrentWorkload(t, c, 50, 200)
}

func TestCacheConcurrencyTTL(t *testing.T) {
	c, err := NewLRU(10)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	var wg sync.WaitGroup
	// Writers adding expiring keys
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := fmt.Sprintf("ttl-key-%d", j%5)
				c.SetWithTTL(key, "val", 10*time.Millisecond)
				time.Sleep(2 * time.Millisecond)
			}
		}(i)
	}

	// Readers reading concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := fmt.Sprintf("ttl-key-%d", j%5)
				c.Get(key)
				time.Sleep(2 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
}
