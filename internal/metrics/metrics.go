package metrics

import (
	"sync/atomic"
	"time"
)

// CacheMetrics tracks cache engine statistics atomically and safely for concurrent access.
type CacheMetrics struct {
	hits      atomic.Uint64
	misses    atomic.Uint64
	sets      atomic.Uint64
	deletes   atomic.Uint64
	evictions atomic.Uint64
	expired   atomic.Uint64
}

// NewCacheMetrics initializes and returns a new CacheMetrics instance.
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{}
}

// IncHits increments the cache hit counter.
func (m *CacheMetrics) IncHits() {
	m.hits.Add(1)
}

// IncMisses increments the cache miss counter.
func (m *CacheMetrics) IncMisses() {
	m.misses.Add(1)
}

// IncSets increments the cache set counter.
func (m *CacheMetrics) IncSets() {
	m.sets.Add(1)
}

// IncDeletes increments the cache delete counter.
func (m *CacheMetrics) IncDeletes() {
	m.deletes.Add(1)
}

// IncEvictions increments the cache eviction counter.
func (m *CacheMetrics) IncEvictions() {
	m.evictions.Add(1)
}

// IncExpired increments the expired entry counter.
func (m *CacheMetrics) IncExpired() {
	m.expired.Add(1)
}

// Reset resets all counters in CacheMetrics to zero.
func (m *CacheMetrics) Reset() {
	m.hits.Store(0)
	m.misses.Store(0)
	m.sets.Store(0)
	m.deletes.Store(0)
	m.evictions.Store(0)
	m.expired.Store(0)
}

// Hits returns the current number of cache hits.
func (m *CacheMetrics) Hits() uint64 {
	return m.hits.Load()
}

// Misses returns the current number of cache misses.
func (m *CacheMetrics) Misses() uint64 {
	return m.misses.Load()
}

// Sets returns the current number of cache sets.
func (m *CacheMetrics) Sets() uint64 {
	return m.sets.Load()
}

// Deletes returns the current number of cache deletes.
func (m *CacheMetrics) Deletes() uint64 {
	return m.deletes.Load()
}

// Evictions returns the current number of evicted entries.
func (m *CacheMetrics) Evictions() uint64 {
	return m.evictions.Load()
}

// Expired returns the current number of expired entries encountered.
func (m *CacheMetrics) Expired() uint64 {
	return m.expired.Load()
}

// HitRate calculates the cache hit rate: hits / (hits + misses).
// Returns 0.0 if total requests (hits + misses) is zero.
func (m *CacheMetrics) HitRate() float64 {
	h := m.hits.Load()
	miss := m.misses.Load()
	total := h + miss
	if total == 0 {
		return 0.0
	}
	return float64(h) / float64(total)
}

// CacheSnapshot represents a consistent point-in-time snapshot of cache engine metrics.
type CacheSnapshot struct {
	Hits      uint64  `json:"hits"`
	Misses    uint64  `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Sets      uint64  `json:"sets"`
	Deletes   uint64  `json:"deletes"`
	Evictions uint64  `json:"evictions"`
	Expired   uint64  `json:"expired"`
}

// Snapshot returns a thread-safe snapshot of cache metrics.
func (m *CacheMetrics) Snapshot() CacheSnapshot {
	return CacheSnapshot{
		Hits:      m.Hits(),
		Misses:    m.Misses(),
		HitRate:   m.HitRate(),
		Sets:      m.Sets(),
		Deletes:   m.Deletes(),
		Evictions: m.Evictions(),
		Expired:   m.Expired(),
	}
}

// HTTPMetrics tracks HTTP-level request telemetry (counts, error classes, and latency).
type HTTPMetrics struct {
	requests       atomic.Uint64
	successes      atomic.Uint64
	errors4xx      atomic.Uint64
	errors5xx      atomic.Uint64
	totalLatencyNs atomic.Uint64
}

// NewHTTPMetrics initializes and returns a new HTTPMetrics instance.
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{}
}

// RecordRequest registers an HTTP response code and elapsed request processing duration.
func (m *HTTPMetrics) RecordRequest(statusCode int, duration time.Duration) {
	m.requests.Add(1)
	m.totalLatencyNs.Add(uint64(duration.Nanoseconds()))

	if statusCode >= 200 && statusCode < 300 {
		m.successes.Add(1)
	} else if statusCode >= 400 && statusCode < 500 {
		m.errors4xx.Add(1)
	} else if statusCode >= 500 {
		m.errors5xx.Add(1)
	}
}

// Reset resets all counters in HTTPMetrics to zero.
func (m *HTTPMetrics) Reset() {
	m.requests.Store(0)
	m.successes.Store(0)
	m.errors4xx.Store(0)
	m.errors5xx.Store(0)
	m.totalLatencyNs.Store(0)
}

// Requests returns total HTTP requests received.
func (m *HTTPMetrics) Requests() uint64 {
	return m.requests.Load()
}

// Successes returns total HTTP 2xx responses.
func (m *HTTPMetrics) Successes() uint64 {
	return m.successes.Load()
}

// Errors4xx returns total HTTP 4xx responses.
func (m *HTTPMetrics) Errors4xx() uint64 {
	return m.errors4xx.Load()
}

// Errors5xx returns total HTTP 5xx responses.
func (m *HTTPMetrics) Errors5xx() uint64 {
	return m.errors5xx.Load()
}

// AvgLatencyMs returns average request latency in milliseconds.
// Returns 0.0 if no requests have been recorded.
func (m *HTTPMetrics) AvgLatencyMs() float64 {
	reqs := m.requests.Load()
	if reqs == 0 {
		return 0.0
	}
	totalNs := m.totalLatencyNs.Load()
	return float64(totalNs) / float64(reqs) / 1e6
}

// HTTPSnapshot represents a consistent snapshot of HTTP request telemetry.
type HTTPSnapshot struct {
	Requests     uint64  `json:"requests"`
	Successes    uint64  `json:"successes"`
	Errors4xx    uint64  `json:"errors_4xx"`
	Errors5xx    uint64  `json:"errors_5xx"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// Snapshot returns a thread-safe snapshot of HTTP metrics.
func (m *HTTPMetrics) Snapshot() HTTPSnapshot {
	return HTTPSnapshot{
		Requests:     m.Requests(),
		Successes:    m.Successes(),
		Errors4xx:    m.Errors4xx(),
		Errors5xx:    m.Errors5xx(),
		AvgLatencyMs: m.AvgLatencyMs(),
	}
}
