package cache

import (
	"fmt"
	"testing"
)

func BenchmarkLRUGet(b *testing.B) {
	c, _ := NewLRU(1000)
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		_, _ = c.Get(key)
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
		_, _ = c.Get(key)
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
		_, _ = c.Get(key)
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
