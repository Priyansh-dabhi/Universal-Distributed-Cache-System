package cache

import (
	"fmt"
	"sync"
	"testing"
)

// Test 1 — Set and Get
func TestSetAndGet(t *testing.T) {
	c := New()

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
	c := New()

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
	c := New()

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
	c := New()

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
	c := New()

	deleted := c.Delete("non_existent")
	if deleted {
		t.Fatalf("expected Delete('non_existent') to return false")
	}
}

// Test 6 — Size
func TestSize(t *testing.T) {
	c := New()

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
	c := New()

	c.Set("key", "")
	val, ok := c.Get("key")
	if !ok {
		t.Fatalf("expected key with empty string value to return ok=true")
	}
	if val != "" {
		t.Fatalf("expected val to be empty string, got %q", val)
	}
}

// Test 8 — Concurrent access
func TestConcurrentAccess(t *testing.T) {
	c := New()
	var wg sync.WaitGroup

	numGoroutines := 50
	operationsPerGoroutine := 100

	// Spawn writer goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", j%10)
				val := fmt.Sprintf("val-%d-%d", workerID, j)
				c.Set(key, val)
			}
		}(i)
	}

	// Spawn reader goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", j%10)
				_, _ = c.Get(key)
			}
		}()
	}

	// Spawn deleter goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d", j%10)
				_ = c.Delete(key)
			}
		}()
	}

	// Spawn size checker goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				_ = c.Size()
			}
		}()
	}

	wg.Wait()
}
