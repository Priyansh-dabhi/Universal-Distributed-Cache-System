package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestCacheMetrics(t *testing.T) {
	m := NewCacheMetrics()

	// Initial state
	if m.Hits() != 0 || m.Misses() != 0 || m.Sets() != 0 || m.Deletes() != 0 || m.Evictions() != 0 || m.Expired() != 0 {
		t.Fatalf("expected all counters to start at 0")
	}
	if m.HitRate() != 0.0 {
		t.Fatalf("expected hit rate 0.0 with 0 queries, got %f", m.HitRate())
	}

	m.IncHits()
	m.IncHits()
	m.IncHits()
	m.IncHits() // 4 hits

	m.IncMisses() // 1 miss

	if m.Hits() != 4 || m.Misses() != 1 {
		t.Fatalf("expected 4 hits and 1 miss")
	}

	expectedHitRate := 4.0 / 5.0
	if m.HitRate() != expectedHitRate {
		t.Fatalf("expected hit rate %f, got %f", expectedHitRate, m.HitRate())
	}

	m.IncSets()
	m.IncDeletes()
	m.IncEvictions()
	m.IncExpired()

	snap := m.Snapshot()
	if snap.Hits != 4 || snap.Misses != 1 || snap.HitRate != expectedHitRate ||
		snap.Sets != 1 || snap.Deletes != 1 || snap.Evictions != 1 || snap.Expired != 1 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestHTTPMetrics(t *testing.T) {
	m := NewHTTPMetrics()

	if m.Requests() != 0 || m.AvgLatencyMs() != 0.0 {
		t.Fatalf("expected 0 requests and 0.0 avg latency initially")
	}

	m.RecordRequest(200, 2*time.Millisecond)
	m.RecordRequest(201, 4*time.Millisecond)
	m.RecordRequest(404, 3*time.Millisecond)
	m.RecordRequest(500, 7*time.Millisecond)

	if m.Requests() != 4 {
		t.Fatalf("expected 4 requests, got %d", m.Requests())
	}
	if m.Successes() != 2 {
		t.Fatalf("expected 2 successes, got %d", m.Successes())
	}
	if m.Errors4xx() != 1 {
		t.Fatalf("expected 1 4xx error, got %d", m.Errors4xx())
	}
	if m.Errors5xx() != 1 {
		t.Fatalf("expected 1 5xx error, got %d", m.Errors5xx())
	}

	// (2 + 4 + 3 + 7) / 4 = 16 / 4 = 4.0 ms
	if m.AvgLatencyMs() != 4.0 {
		t.Fatalf("expected avg latency 4.0 ms, got %f", m.AvgLatencyMs())
	}

	snap := m.Snapshot()
	if snap.Requests != 4 || snap.Successes != 2 || snap.Errors4xx != 1 || snap.Errors5xx != 1 || snap.AvgLatencyMs != 4.0 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestConcurrentMetrics(t *testing.T) {
	cm := NewCacheMetrics()
	hm := NewHTTPMetrics()

	var wg sync.WaitGroup
	workers := 50
	iterations := 1000

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cm.IncHits()
				cm.IncMisses()
				cm.IncSets()
				cm.IncDeletes()
				cm.IncEvictions()
				cm.IncExpired()

				hm.RecordRequest(200, 1*time.Millisecond)
				hm.RecordRequest(400, 1*time.Millisecond)
				hm.RecordRequest(500, 1*time.Millisecond)
			}
		}()
	}

	wg.Wait()

	expectedCount := uint64(workers * iterations)
	if cm.Hits() != expectedCount || cm.Misses() != expectedCount {
		t.Fatalf("cache metrics concurrency mismatch")
	}
	if hm.Requests() != expectedCount*3 {
		t.Fatalf("http metrics concurrency mismatch")
	}
}
