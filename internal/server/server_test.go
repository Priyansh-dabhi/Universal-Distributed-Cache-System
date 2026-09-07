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

func newTestServer(t *testing.T, capacity int, policy cache.EvictionPolicy) (*Server, *cache.Cache) {
	t.Helper()
	c, err := cache.NewWithPolicy(capacity, policy)
	if err != nil {
		t.Fatalf("failed to create cache for test: %v", err)
	}
	srv := New(c, DefaultConfig(8080))
	return srv, c
}

// Test 1 — Health
func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res healthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", res.Status)
	}
}

// Test 2 — SET
func TestSet(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	body := `{"value":"100"}`
	req := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res setResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Message != "cache entry stored" {
		t.Fatalf("expected 'cache entry stored', got %q", res.Message)
	}
}

// Test 3 — GET
func TestGet(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	// Set A
	setReq := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(`{"value":"100"}`))
	setW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(setW, setReq)
	if setW.Code != http.StatusOK {
		t.Fatalf("failed to set key: %d", setW.Code)
	}

	// Get A
	getReq := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getW.Code)
	}

	var res getResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Key != "A" || res.Value != "100" {
		t.Fatalf("expected key 'A' and value '100', got key %q and value %q", res.Key, res.Value)
	}
}

// Test 4 — GET missing key
func TestGetMissingKey(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	req := httptest.NewRequest(http.MethodGet, "/cache/missing", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var res errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Error != "cache miss" {
		t.Fatalf("expected error 'cache miss', got %q", res.Error)
	}
}

// Test 5 — DELETE
func TestDelete(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	// Set A
	setReq := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(`{"value":"100"}`))
	srv.Handler().ServeHTTP(httptest.NewRecorder(), setReq)

	// Delete A
	delReq := httptest.NewRequest(http.MethodDelete, "/cache/A", nil)
	delW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delW, delReq)

	if delW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", delW.Code)
	}

	var res deleteResponse
	if err := json.Unmarshal(delW.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Message != "cache entry deleted" {
		t.Fatalf("expected 'cache entry deleted', got %q", res.Message)
	}

	// Verify GET A returns 404
	getReq := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 after delete, got %d", getW.Code)
	}
}

// Test 6 — SET with TTL
func TestSetWithTTL(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	// Set A with 50ms TTL
	body := `{"value":"Priyansh","ttl":"50ms"}`
	setReq := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(body))
	setW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(setW, setReq)

	if setW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", setW.Code)
	}

	// Immediately GET: should exist
	getReq := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	getW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected status 200 immediately, got %d", getW.Code)
	}

	// Wait for TTL to expire
	time.Sleep(70 * time.Millisecond)

	// GET after expiration: should return 404
	getExpReq := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	getExpW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getExpW, getExpReq)

	if getExpW.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 after expiration, got %d", getExpW.Code)
	}
}

// Test 7 — Invalid JSON
func TestInvalidJSON(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	malformed := `{"value":`
	req := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(malformed))
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for malformed JSON, got %d", w.Code)
	}

	var res errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Error == "" {
		t.Fatalf("expected an error description in response")
	}
}

// Test 8 — Missing value
func TestMissingValue(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	body := `{}`
	req := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing value, got %d", w.Code)
	}

	var res errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Error != "value is required" {
		t.Fatalf("expected 'value is required', got %q", res.Error)
	}
}

// Test 9 — Cache information
func TestCacheInfo(t *testing.T) {
	srv, c := newTestServer(t, 50, cache.PolicyLRU)

	c.Set("k1", "v1")
	c.Set("k2", "v2")

	req := httptest.NewRequest(http.MethodGet, "/cache", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res cacheInfoResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res.Size != 2 {
		t.Fatalf("expected size 2, got %d", res.Size)
	}
	if res.Capacity != 50 {
		t.Fatalf("expected capacity 50, got %d", res.Capacity)
	}
	if res.Policy != "lru" {
		t.Fatalf("expected policy 'lru', got %q", res.Policy)
	}
}

// Test 10 — URL encoded keys
func TestURLEncodedKeys(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	keys := []string{
		"user:123",
		"product:456",
		"session:abc",
		"foo bar",
		"hello/world",
	}

	for _, k := range keys {
		escapedKey := k
		// For spaces and slashes, use encoded URL path
		urlPath := "/cache/" + k
		if k == "foo bar" {
			urlPath = "/cache/foo%20bar"
		} else if k == "hello/world" {
			urlPath = "/cache/hello%2Fworld"
		}

		setReq := httptest.NewRequest(http.MethodPut, urlPath, bytes.NewBufferString(`{"value":"val"}`))
		setW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(setW, setReq)
		if setW.Code != http.StatusOK {
			t.Fatalf("failed to set key %q: %d", escapedKey, setW.Code)
		}

		getReq := httptest.NewRequest(http.MethodGet, urlPath, nil)
		getW := httptest.NewRecorder()
		srv.Handler().ServeHTTP(getW, getReq)
		if getW.Code != http.StatusOK {
			t.Fatalf("failed to get key %q: %d", escapedKey, getW.Code)
		}

		var res getResponse
		_ = json.Unmarshal(getW.Body.Bytes(), &res)
		if res.Key != k || res.Value != "val" {
			t.Fatalf("key mismatch: expected %q, got %q", k, res.Key)
		}
	}
}

// Test 11 — Unsupported method
func TestUnsupportedMethod(t *testing.T) {
	srv, _ := newTestServer(t, 10, cache.PolicyLRU)

	// POST /cache/A -> 405
	req1 := httptest.NewRequest(http.MethodPost, "/cache/A", nil)
	w1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w1, req1)
	if w1.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /cache/A, got %d", w1.Code)
	}

	// POST /health -> 405
	req2 := httptest.NewRequest(http.MethodPost, "/health", nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /health, got %d", w2.Code)
	}

	// DELETE /cache -> 405
	req3 := httptest.NewRequest(http.MethodDelete, "/cache", nil)
	w3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w3, req3)
	if w3.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for DELETE /cache, got %d", w3.Code)
	}
}

// Test 12 — Concurrent HTTP requests
func TestConcurrentHTTPRequests(t *testing.T) {
	srv, c := newTestServer(t, 20, cache.PolicyLRU)
	server := httptest.NewServer(srv.Handler())
	defer server.Close()

	client := server.Client()
	var wg sync.WaitGroup

	numGoroutines := 30
	opsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("k-%d", j%15)
				val := fmt.Sprintf("val-%d-%d", workerID, j)

				// PUT
				putBody, _ := json.Marshal(map[string]string{"value": val})
				putReq, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/cache/%s", server.URL, key), bytes.NewBuffer(putBody))
				putResp, err := client.Do(putReq)
				if err == nil {
					_ = putResp.Body.Close()
				}

				// GET
				getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/cache/%s", server.URL, key), nil)
				getResp, err := client.Do(getReq)
				if err == nil {
					_ = getResp.Body.Close()
				}

				// DELETE
				if j%5 == 0 {
					delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/cache/%s", server.URL, key), nil)
					delResp, err := client.Do(delReq)
					if err == nil {
						_ = delResp.Body.Close()
					}
				}
			}
		}(i)
	}

	wg.Wait()

	if c.Size() > 20 {
		t.Fatalf("cache size %d exceeded capacity 20 after concurrent access", c.Size())
	}
}

// Test with LFU and 2Q policies
func TestServerWithDifferentPolicies(t *testing.T) {
	policies := []cache.EvictionPolicy{cache.PolicyLRU, cache.PolicyLFU, cache.Policy2Q}

	for _, p := range policies {
		t.Run(string(p), func(t *testing.T) {
			srv, _ := newTestServer(t, 5, p)

			// PUT
			setReq := httptest.NewRequest(http.MethodPut, "/cache/test", bytes.NewBufferString(`{"value":"hello"}`))
			setW := httptest.NewRecorder()
			srv.Handler().ServeHTTP(setW, setReq)
			if setW.Code != http.StatusOK {
				t.Fatalf("[%s] expected 200 for SET, got %d", p, setW.Code)
			}

			// GET
			getReq := httptest.NewRequest(http.MethodGet, "/cache/test", nil)
			getW := httptest.NewRecorder()
			srv.Handler().ServeHTTP(getW, getReq)
			if getW.Code != http.StatusOK {
				t.Fatalf("[%s] expected 200 for GET, got %d", p, getW.Code)
			}

			// Info
			infoReq := httptest.NewRequest(http.MethodGet, "/cache", nil)
			infoW := httptest.NewRecorder()
			srv.Handler().ServeHTTP(infoW, infoReq)
			if infoW.Code != http.StatusOK {
				t.Fatalf("[%s] expected 200 for /cache, got %d", p, infoW.Code)
			}
		})
	}
}
