package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"universal-distributed-cache/internal/cache"
)

// Test 1 — Node configuration
func TestNodeConfiguration(t *testing.T) {
	cfg := Config{
		ID:       "node-1",
		Host:     "localhost",
		Port:     8001,
		Capacity: 100,
		Policy:   "lru",
	}

	n, err := New(cfg)
	if err != nil {
		t.Fatalf("expected node creation to succeed, got error: %v", err)
	}

	if n.ID() != "node-1" {
		t.Fatalf("expected ID 'node-1', got %q", n.ID())
	}
	if n.Host() != "localhost" {
		t.Fatalf("expected Host 'localhost', got %q", n.Host())
	}
	if n.Port() != 8001 {
		t.Fatalf("expected Port 8001, got %d", n.Port())
	}
	if n.Cache().Capacity() != 100 {
		t.Fatalf("expected Capacity 100, got %d", n.Cache().Capacity())
	}
	if n.Cache().Policy() != cache.PolicyLRU {
		t.Fatalf("expected Policy LRU, got %s", n.Cache().Policy())
	}
}

// Test 2 — Invalid node ID
func TestInvalidNodeID(t *testing.T) {
	cfg := Config{
		ID:       "",
		Host:     "127.0.0.1",
		Port:     8001,
		Capacity: 100,
		Policy:   "lru",
	}

	if _, err := New(cfg); err != ErrEmptyNodeID {
		t.Fatalf("expected ErrEmptyNodeID, got %v", err)
	}

	cfg.ID = "   "
	if _, err := New(cfg); err != ErrEmptyNodeID {
		t.Fatalf("expected ErrEmptyNodeID for whitespace ID, got %v", err)
	}
}

// Test 3 — Invalid port
func TestInvalidPort(t *testing.T) {
	for _, port := range []int{0, -1, 70000} {
		cfg := Config{
			ID:       "node-1",
			Host:     "127.0.0.1",
			Port:     port,
			Capacity: 100,
			Policy:   "lru",
		}
		if _, err := New(cfg); err != ErrInvalidPort {
			t.Fatalf("expected ErrInvalidPort for port %d, got %v", port, err)
		}
	}
}

// Test 4 — Invalid capacity
func TestInvalidCapacity(t *testing.T) {
	for _, cap := range []int{0, -5} {
		cfg := Config{
			ID:       "node-1",
			Host:     "127.0.0.1",
			Port:     8001,
			Capacity: cap,
			Policy:   "lru",
		}
		if _, err := New(cfg); err != ErrInvalidCapacity {
			t.Fatalf("expected ErrInvalidCapacity for capacity %d, got %v", cap, err)
		}
	}
}

// Test 5 — Invalid policy
func TestInvalidPolicy(t *testing.T) {
	cfg := Config{
		ID:       "node-1",
		Host:     "127.0.0.1",
		Port:     8001,
		Capacity: 100,
		Policy:   "unknown",
	}

	if _, err := New(cfg); err == nil {
		t.Fatalf("expected error for invalid policy 'unknown', got nil")
	}
}

// Test 6 — Node health
func TestNodeHealth(t *testing.T) {
	n, err := New(Config{
		ID:       "node-1",
		Host:     "localhost",
		Port:     8001,
		Capacity: 100,
		Policy:   "lru",
	})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	n.Server().Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res struct {
		Status string `json:"status"`
		NodeID string `json:"node_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	if res.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", res.Status)
	}
	if res.NodeID != "node-1" {
		t.Fatalf("expected node_id 'node-1', got %q", res.NodeID)
	}
}

// Test 7 — Node information
func TestNodeInformation(t *testing.T) {
	n, err := New(Config{
		ID:       "node-1",
		Host:     "localhost",
		Port:     8001,
		Capacity: 100,
		Policy:   "lru",
	})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/node", nil)
	w := httptest.NewRecorder()
	n.Server().Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res struct {
		ID       string `json:"id"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Capacity int    `json:"capacity"`
		Policy   string `json:"policy"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal node response: %v", err)
	}

	if res.ID != "node-1" || res.Host != "localhost" || res.Port != 8001 || res.Capacity != 100 || res.Policy != "lru" {
		t.Fatalf("node info mismatch: %+v", res)
	}
}

// Test 8 — Cache isolation
func TestCacheIsolation(t *testing.T) {
	node1, _ := New(Config{ID: "node-1", Host: "localhost", Port: 8001, Capacity: 100, Policy: "lru"})
	node2, _ := New(Config{ID: "node-2", Host: "localhost", Port: 8002, Capacity: 100, Policy: "lru"})

	// Store A -> 100 on Node 1
	setReq := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(`{"value":"100"}`))
	setW := httptest.NewRecorder()
	node1.Server().Handler().ServeHTTP(setW, setReq)
	if setW.Code != http.StatusOK {
		t.Fatalf("failed to set key on node 1: %d", setW.Code)
	}

	// Verify Node 1 returns 100
	getReq1 := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	getW1 := httptest.NewRecorder()
	node1.Server().Handler().ServeHTTP(getW1, getReq1)
	if getW1.Code != http.StatusOK {
		t.Fatalf("expected 200 on node 1, got %d", getW1.Code)
	}

	// Verify Node 2 returns 404 Not Found (isolated memory)
	getReq2 := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	getW2 := httptest.NewRecorder()
	node2.Server().Handler().ServeHTTP(getW2, getReq2)
	if getW2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on node 2, got %d", getW2.Code)
	}
}

// Test 9 — Independent policies
func TestIndependentPolicies(t *testing.T) {
	node1, _ := New(Config{ID: "node-1", Host: "localhost", Port: 8001, Capacity: 100, Policy: "lru"})
	node2, _ := New(Config{ID: "node-2", Host: "localhost", Port: 8002, Capacity: 100, Policy: "lfu"})
	node3, _ := New(Config{ID: "node-3", Host: "localhost", Port: 8003, Capacity: 100, Policy: "2q"})

	if node1.Cache().Policy() != cache.PolicyLRU {
		t.Fatalf("expected node 1 policy LRU, got %s", node1.Cache().Policy())
	}
	if node2.Cache().Policy() != cache.PolicyLFU {
		t.Fatalf("expected node 2 policy LFU, got %s", node2.Cache().Policy())
	}
	if node3.Cache().Policy() != cache.Policy2Q {
		t.Fatalf("expected node 3 policy 2Q, got %s", node3.Cache().Policy())
	}
}

// Test 10 — Independent TTL
func TestIndependentTTL(t *testing.T) {
	node1, _ := New(Config{ID: "node-1", Host: "localhost", Port: 8001, Capacity: 100, Policy: "lru"})
	node2, _ := New(Config{ID: "node-2", Host: "localhost", Port: 8002, Capacity: 100, Policy: "lru"})

	// Node 1: A with TTL 50ms
	req1 := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(`{"value":"100","ttl":"50ms"}`))
	node1.Server().Handler().ServeHTTP(httptest.NewRecorder(), req1)

	// Node 2: A with TTL 500ms
	req2 := httptest.NewRequest(http.MethodPut, "/cache/A", bytes.NewBufferString(`{"value":"100","ttl":"500ms"}`))
	node2.Server().Handler().ServeHTTP(httptest.NewRecorder(), req2)

	// Wait 75ms (Node 1 expires, Node 2 remains valid)
	time.Sleep(75 * time.Millisecond)

	// Node 1: should be 404 (expired)
	getReq1 := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	w1 := httptest.NewRecorder()
	node1.Server().Handler().ServeHTTP(w1, getReq1)
	if w1.Code != http.StatusNotFound {
		t.Fatalf("expected node 1 to expire key 'A', got status %d", w1.Code)
	}

	// Node 2: should be 200 (valid)
	getReq2 := httptest.NewRequest(http.MethodGet, "/cache/A", nil)
	w2 := httptest.NewRecorder()
	node2.Server().Handler().ServeHTTP(w2, getReq2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected node 2 key 'A' to remain valid, got status %d", w2.Code)
	}
}

// Test 11 (Integration) — Multiple HTTP servers integration
func TestMultipleNodesIntegration(t *testing.T) {
	node1, _ := New(Config{ID: "node-1", Host: "127.0.0.1", Port: 18001, Capacity: 10, Policy: "lru"})
	node2, _ := New(Config{ID: "node-2", Host: "127.0.0.1", Port: 18002, Capacity: 10, Policy: "lfu"})
	node3, _ := New(Config{ID: "node-3", Host: "127.0.0.1", Port: 18003, Capacity: 10, Policy: "2q"})

	s1 := httptest.NewServer(node1.Server().Handler())
	defer s1.Close()

	s2 := httptest.NewServer(node2.Server().Handler())
	defer s2.Close()

	s3 := httptest.NewServer(node3.Server().Handler())
	defer s3.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// PUT A on Node 1
	putReq, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/cache/A", s1.URL), bytes.NewBufferString(`{"value":"stored_on_node1"}`))
	putReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(putReq)
	if err != nil {
		t.Fatalf("failed to PUT on node 1: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on PUT node 1, got %d", resp.StatusCode)
	}

	// GET A from Node 1 -> HIT (200)
	resp1, err := client.Get(fmt.Sprintf("%s/cache/A", s1.URL))
	if err != nil {
		t.Fatalf("failed to GET from node 1: %v", err)
	}
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on node 1, got %d", resp1.StatusCode)
	}

	// GET A from Node 2 -> MISS (404)
	resp2, err := client.Get(fmt.Sprintf("%s/cache/A", s2.URL))
	if err != nil {
		t.Fatalf("failed to GET from node 2: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 on node 2, got %d", resp2.StatusCode)
	}

	// GET A from Node 3 -> MISS (404)
	resp3, err := client.Get(fmt.Sprintf("%s/cache/A", s3.URL))
	if err != nil {
		t.Fatalf("failed to GET from node 3: %v", err)
	}
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 on node 3, got %d", resp3.StatusCode)
	}
}
