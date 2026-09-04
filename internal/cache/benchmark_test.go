package cache

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

// Global sinks to prevent compiler optimizations from eliminating benchmarked calls.
var (
	benchValSink   string
	benchFoundSink bool
)

// policyFactory creates a new cache instance with the specified policy and capacity.
type policyFactory struct {
	name string
	fn   func(capacity int) (*Cache, error)
}

var policies = []policyFactory{
	{name: "LRU", fn: NewLRU},
	{name: "LFU", fn: NewLFU},
	{name: "2Q", fn: New2Q},
}

var benchmarkCapacities = []int{100, 1000, 10000}

// ---------------------------------------------------------------------------
// 1. Backward-compatible top-level microbenchmarks
// ---------------------------------------------------------------------------

func BenchmarkLRUGet(b *testing.B) {
	c, _ := NewLRU(1000)
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		benchValSink, benchFoundSink = c.Get(key)
	}
}

func BenchmarkLFUGet(b *testing.B) {
	c, _ := NewLFU(1000)
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		benchValSink, benchFoundSink = c.Get(key)
	}
}

func Benchmark2QGet(b *testing.B) {
	c, _ := New2Q(1000)
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		benchValSink, benchFoundSink = c.Get(key)
	}
}

func BenchmarkLRUSet(b *testing.B) {
	c, _ := NewLRU(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1500)
		c.Set(key, "benchmark-value")
	}
}

func BenchmarkLFUSet(b *testing.B) {
	c, _ := NewLFU(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1500)
		c.Set(key, "benchmark-value")
	}
}

func Benchmark2QSet(b *testing.B) {
	c, _ := New2Q(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1500)
		c.Set(key, "benchmark-value")
	}
}

// ---------------------------------------------------------------------------
// 2. Comprehensive Operation Microbenchmarks Across Capacities
// ---------------------------------------------------------------------------

// BenchmarkGetExisting measures GET latency on keys that are known to exist in cache.
func BenchmarkGetExisting(b *testing.B) {
	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				// Pre-populate keys to capacity
				keys := make([]string, cap)
				for i := 0; i < cap; i++ {
					keys[i] = fmt.Sprintf("key-%d", i)
					c.Set(keys[i], "val")
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					benchValSink, benchFoundSink = c.Get(keys[i%cap])
				}
			})
		}
	}
}

// BenchmarkGetMissing measures GET latency on keys that do not exist.
func BenchmarkGetMissing(b *testing.B) {
	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				// Pre-populate keys to capacity
				for i := 0; i < cap; i++ {
					c.Set(fmt.Sprintf("existing-%d", i), "val")
				}
				// Prepare missing keys
				missingKeys := make([]string, 1000)
				for i := 0; i < 1000; i++ {
					missingKeys[i] = fmt.Sprintf("missing-%d", i)
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					benchValSink, benchFoundSink = c.Get(missingKeys[i%1000])
				}
			})
		}
	}
}

// BenchmarkSetNew measures SET latency when constantly inserting new keys beyond capacity, forcing evictions.
func BenchmarkSetNew(b *testing.B) {
	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				// Pre-populate to capacity
				for i := 0; i < cap; i++ {
					c.Set(fmt.Sprintf("init-%d", i), "val")
				}

				// Pre-generate keys to avoid fmt.Sprintf overhead inside loop
				keyCount := 100000
				keys := make([]string, keyCount)
				for i := 0; i < keyCount; i++ {
					keys[i] = fmt.Sprintf("new-key-%d", i)
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					c.Set(keys[i%keyCount], "val")
				}
			})
		}
	}
}

// BenchmarkSetExisting measures SET latency when updating existing keys without causing eviction.
func BenchmarkSetExisting(b *testing.B) {
	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				keys := make([]string, cap)
				for i := 0; i < cap; i++ {
					keys[i] = fmt.Sprintf("existing-key-%d", i)
					c.Set(keys[i], "val")
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					c.Set(keys[i%cap], "updated-val")
				}
			})
		}
	}
}

// BenchmarkDeleteExisting measures DELETE latency for keys present in the cache.
func BenchmarkDeleteExisting(b *testing.B) {
	for _, p := range policies {
		for _, cap := range benchmarkCapacities {
			name := fmt.Sprintf("%s/cap=%d", p.name, cap)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(cap)
				keys := make([]string, cap)
				for i := 0; i < cap; i++ {
					keys[i] = fmt.Sprintf("del-key-%d", i)
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					idx := i % cap
					if idx == 0 {
						// Re-populate when full cycle finishes
						b.StopTimer()
						for j := 0; j < cap; j++ {
							c.Set(keys[j], "val")
						}
						b.StartTimer()
					}
					c.Delete(keys[idx])
				}
			})
		}
	}
}

// ---------------------------------------------------------------------------
// 3. TTL Microbenchmarks
// ---------------------------------------------------------------------------

// BenchmarkTTLSetWithTTL evaluates insertion overhead when setting a key with an explicit TTL.
func BenchmarkTTLSetWithTTL(b *testing.B) {
	for _, p := range policies {
		b.Run(p.name, func(b *testing.B) {
			c, _ := p.fn(1000)
			keys := make([]string, 1000)
			for i := 0; i < 1000; i++ {
				keys[i] = fmt.Sprintf("ttl-key-%d", i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				c.SetWithTTL(keys[i%1000], "val", 10*time.Minute)
			}
		})
	}
}

// BenchmarkTTLGetBeforeExpiry evaluates GET performance on keys with an active TTL before expiration.
func BenchmarkTTLGetBeforeExpiry(b *testing.B) {
	for _, p := range policies {
		b.Run(p.name, func(b *testing.B) {
			c, _ := p.fn(1000)
			keys := make([]string, 1000)
			for i := 0; i < 1000; i++ {
				keys[i] = fmt.Sprintf("ttl-active-%d", i)
				c.SetWithTTL(keys[i], "val", 1*time.Hour)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchValSink, benchFoundSink = c.Get(keys[i%1000])
			}
		})
	}
}

// BenchmarkTTLGetAfterExpiry evaluates lazy expiration latency when retrieving expired entries.
func BenchmarkTTLGetAfterExpiry(b *testing.B) {
	for _, p := range policies {
		b.Run(p.name, func(b *testing.B) {
			c, _ := p.fn(1000)
			keys := make([]string, 1000)
			for i := 0; i < 1000; i++ {
				keys[i] = fmt.Sprintf("ttl-expired-%d", i)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := i % 1000
				if idx == 0 {
					b.StopTimer()
					// Pre-insert entries with 1 nanosecond TTL so they expire instantly without sleeping
					for j := 0; j < 1000; j++ {
						c.SetWithTTL(keys[j], "val", 1*time.Nanosecond)
					}
					// Micro-yield to ensure monotonic clock has progressed
					time.Sleep(1 * time.Millisecond)
					b.StartTimer()
				}
				benchValSink, benchFoundSink = c.Get(keys[idx])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. Concurrency Parallel Benchmarks (b.RunParallel)
// ---------------------------------------------------------------------------

// BenchmarkParallelGet evaluates concurrent read throughput using multiple goroutines.
func BenchmarkParallelGet(b *testing.B) {
	for _, p := range policies {
		b.Run(p.name, func(b *testing.B) {
			c, _ := p.fn(1000)
			keys := make([]string, 1000)
			for i := 0; i < 1000; i++ {
				keys[i] = fmt.Sprintf("par-get-%d", i)
				c.Set(keys[i], "val")
			}

			var counter atomic.Uint64
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					idx := int(counter.Add(1) % 1000)
					_, _ = c.Get(keys[idx])
				}
			})
		})
	}
}

// BenchmarkParallelSet evaluates concurrent write throughput under contention.
func BenchmarkParallelSet(b *testing.B) {
	for _, p := range policies {
		b.Run(p.name, func(b *testing.B) {
			c, _ := p.fn(1000)
			keys := make([]string, 2000)
			for i := 0; i < 2000; i++ {
				keys[i] = fmt.Sprintf("par-set-%d", i)
			}

			var counter atomic.Uint64
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					idx := int(counter.Add(1) % 2000)
					c.Set(keys[idx], "val")
				}
			})
		})
	}
}

// BenchmarkParallelMixed evaluates mixed concurrent workloads (80% GET / 20% SET, and 95% GET / 5% SET).
func BenchmarkParallelMixed(b *testing.B) {
	ratios := []struct {
		name       string
		readRatio  int // e.g. 80 means 80% GET
		totalRatio int // 100
	}{
		{name: "80Get_20Set", readRatio: 80, totalRatio: 100},
		{name: "95Get_5Set", readRatio: 95, totalRatio: 100},
	}

	for _, p := range policies {
		for _, r := range ratios {
			name := fmt.Sprintf("%s/%s", p.name, r.name)
			b.Run(name, func(b *testing.B) {
				c, _ := p.fn(1000)
				keys := make([]string, 1500)
				for i := 0; i < 1500; i++ {
					keys[i] = fmt.Sprintf("par-mix-%d", i)
					if i < 1000 {
						c.Set(keys[i], "val")
					}
				}

				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
					for pb.Next() {
						idx := rnd.Intn(1500)
						op := rnd.Intn(r.totalRatio)
						if op < r.readRatio {
							_, _ = c.Get(keys[idx])
						} else {
							c.Set(keys[idx], "val")
						}
					}
				})
			})
		}
	}
}
