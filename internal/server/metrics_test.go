package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"universal-distributed-cache/internal/cache"
)

func TestMetricsEndpoint(t *testing.T) {
	c, _ := cache.NewLRU(10)
	cfg := DefaultConfig(8080)
	cfg.NodeID = "node-test"
	s := New(c, cfg)

	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	client := ts.Client()

	// 1. Initial GET /metrics
	res, err := client.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed GET /metrics: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}

	var mData map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&mData); err != nil {
		t.Fatalf("failed to decode metrics JSON: %v", err)
	}
	_ = res.Body.Close()

	if mData["policy"] != "lru" || mData["capacity"] != float64(10) || mData["size"] != float64(0) {
		t.Fatalf("unexpected initial cache info in metrics: %+v", mData)
	}

	// 2. Perform operations:
	// PUT /cache/k1
	putReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/cache/k1", bytes.NewBufferString(`{"value":"v1"}`))
	putReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(putReq)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT failed: %v", err)
	}
	_ = resp.Body.Close()

	// GET /cache/k1 (HIT)
	resp, err = client.Get(ts.URL + "/cache/k1")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET hit failed: %v", err)
	}
	_ = resp.Body.Close()

	// GET /cache/missing (MISS)
	resp, err = client.Get(ts.URL + "/cache/missing")
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 on missing key, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// PUT with short TTL and wait for expiration
	putTTL, _ := http.NewRequest(http.MethodPut, ts.URL+"/cache/exp-key", bytes.NewBufferString(`{"value":"exp","ttl":"20ms"}`))
	putTTL.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(putTTL)
	_ = resp.Body.Close()

	time.Sleep(30 * time.Millisecond)

	// GET expired key (EXPIRED + MISS)
	resp, _ = client.Get(ts.URL + "/cache/exp-key")
	_ = resp.Body.Close()

	// DELETE /cache/k1
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/cache/k1", nil)
	resp, _ = client.Do(delReq)
	_ = resp.Body.Close()

	// 3. Verify updated /metrics
	res, err = client.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed GET /metrics: %v", err)
	}
	defer res.Body.Close()

	var updated map[string]interface{}
	_ = json.NewDecoder(res.Body).Decode(&updated)

	if updated["hits"] != float64(1) {
		t.Errorf("expected 1 hit, got %v", updated["hits"])
	}
	if updated["misses"] != float64(2) { // 1 missing + 1 expired
		t.Errorf("expected 2 misses, got %v", updated["misses"])
	}
	if updated["sets"] != float64(2) { // k1 + exp-key
		t.Errorf("expected 2 sets, got %v", updated["sets"])
	}
	if updated["deletes"] != float64(1) {
		t.Errorf("expected 1 delete, got %v", updated["deletes"])
	}
	if updated["expired"] != float64(1) {
		t.Errorf("expected 1 expired, got %v", updated["expired"])
	}

	// Hit rate: 1 / (1 + 2) = 1/3 = ~0.3333
	hitRate, ok := updated["hit_rate"].(float64)
	if !ok || hitRate < 0.33 || hitRate > 0.34 {
		t.Errorf("expected hit rate ~0.333, got %v", updated["hit_rate"])
	}

	// 4xx errors should be at least 2 (404 missing key + 404 expired key)
	if errors4xx := updated["errors_4xx"].(float64); errors4xx < 2 {
		t.Errorf("expected at least 2 4xx errors, got %v", errors4xx)
	}

	// Latency should be non-negative
	if avgLat := updated["avg_latency_ms"].(float64); avgLat < 0 {
		t.Errorf("expected non-negative avg latency, got %v", avgLat)
	}
}

func TestConcurrentHTTPMetrics(t *testing.T) {
	c, _ := cache.NewLRU(50)
	s := New(c, DefaultConfig(8080))
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	client := ts.Client()
	var wg sync.WaitGroup
	workers := 20
	iterations := 50

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("k-%d-%d", workerID, i%10)

				// PUT
				putURL := fmt.Sprintf("%s/cache/%s", ts.URL, key)
				req, _ := http.NewRequest(http.MethodPut, putURL, bytes.NewBufferString(`{"value":"concurrent-val"}`))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err == nil {
					_ = resp.Body.Close()
				}

				// GET
				gResp, err := client.Get(putURL)
				if err == nil {
					_ = gResp.Body.Close()
				}

				// Occasional GET /metrics
				if i%10 == 0 {
					mResp, err := client.Get(ts.URL + "/metrics")
					if err == nil {
						_ = mResp.Body.Close()
					}
				}
			}
		}(w)
	}

	wg.Wait()

	// Final verification of /metrics endpoint
	res, err := client.Get(ts.URL + "/metrics")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("final metrics check failed: %v", err)
	}
	defer res.Body.Close()

	var snap map[string]interface{}
	_ = json.NewDecoder(res.Body).Decode(&snap)

	if reqCount := snap["requests"].(float64); reqCount < float64(workers*iterations*2) {
		t.Fatalf("expected at least %d requests, got %f", workers*iterations*2, reqCount)
	}
}
