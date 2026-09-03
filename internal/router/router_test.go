package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"universal-distributed-cache/internal/node"
)

// -----------------------------------------------------------------------------
// Section 21: Hash Routing Tests
// -----------------------------------------------------------------------------

// Test 1 — Deterministic routing
func TestDeterministicRouting(t *testing.T) {
	nodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-2", Address: "http://localhost:8002"},
		{ID: "node-3", Address: "http://localhost:8003"},
	}
	ring := NewHashRing(DefaultReplicas)
	for _, n := range nodes {
		if err := ring.AddNode(n); err != nil {
			t.Fatalf("failed to add node: %v", err)
		}
	}

	r := New(ring, DefaultConfig(9000))

	keys := []string{"user:123", "session:abc", "product:999", "alpha", "beta"}

	for _, k := range keys {
		firstNode, err := r.Route(k)
		if err != nil {
			t.Fatalf("unexpected route error for key %q: %v", k, err)
		}

		// Re-routing 50 times must always yield the exact same node
		for i := 0; i < 50; i++ {
			node, err := r.Route(k)
			if err != nil {
				t.Fatalf("unexpected route error: %v", err)
			}
			if node.ID != firstNode.ID {
				t.Fatalf("non-deterministic routing for key %q: got %s, expected %s", k, node.ID, firstNode.ID)
			}
		}
	}
}

// Test 2 — Different keys
func TestDifferentKeys(t *testing.T) {
	nodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-2", Address: "http://localhost:8002"},
		{ID: "node-3", Address: "http://localhost:8003"},
	}
	ring := NewHashRing(DefaultReplicas)
	for _, n := range nodes {
		_ = ring.AddNode(n)
	}
	r := New(ring, DefaultConfig(9000))

	seenNodes := make(map[string]bool)
	keys := []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9", "k10", "k11", "k12"}

	for _, k := range keys {
		n, err := r.Route(k)
		if err != nil {
			t.Fatalf("route error for key %q: %v", k, err)
		}
		seenNodes[n.ID] = true
	}

	// Ensure hashing maps across multiple nodes
	if len(seenNodes) <= 1 {
		t.Fatalf("expected keys to map across multiple nodes, only saw: %+v", seenNodes)
	}
}

// Test 3 — Empty key
func TestEmptyKey(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	r := New(ring, DefaultConfig(9000))

	_, err := r.Route("")
	if err != ErrEmptyKey {
		t.Fatalf("expected ErrEmptyKey, got %v", err)
	}
}

// Test 4 — Node count routing across variable node counts
func TestNodeCountRouting(t *testing.T) {
	for nodeCount := 1; nodeCount <= 10; nodeCount++ {
		ring := NewHashRing(DefaultReplicas)
		nodes := make([]Node, nodeCount)
		for i := 0; i < nodeCount; i++ {
			nodes[i] = Node{
				ID:      fmt.Sprintf("node-%d", i+1),
				Address: fmt.Sprintf("http://localhost:%d", 8000+i),
			}
			if err := ring.AddNode(nodes[i]); err != nil {
				t.Fatalf("failed to add node to ring: %v", err)
			}
		}

		r := New(ring, DefaultConfig(9000))

		for j := 0; j < 100; j++ {
			key := fmt.Sprintf("key-%d-%d", nodeCount, j)
			node, err := r.Route(key)
			if err != nil {
				t.Fatalf("route error: %v", err)
			}

			// Verify node belongs to the configured list
			found := false
			for _, n := range nodes {
				if n.ID == node.ID {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("routed node %s not in configured nodes for count %d", node.ID, nodeCount)
			}
		}
	}
}

// TestNewWithRegistry verifies constructor compatibility with Registry
func TestNewWithRegistry(t *testing.T) {
	nodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-2", Address: "http://localhost:8002"},
	}
	reg, err := NewRegistry(nodes)
	if err != nil {
		t.Fatalf("unexpected registry error: %v", err)
	}

	r := NewWithRegistry(reg, NewFNVHasher(), DefaultConfig(9000))
	if len(r.Nodes()) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(r.Nodes()))
	}
}

// -----------------------------------------------------------------------------
// Section 22: Node Registry Tests
// -----------------------------------------------------------------------------

func TestNodeRegistryValidation(t *testing.T) {
	// Zero nodes
	if _, err := NewRegistry([]Node{}); err != ErrNoNodes {
		t.Fatalf("expected ErrNoNodes, got %v", err)
	}

	// Empty ID
	if _, err := NewRegistry([]Node{{ID: "", Address: "http://localhost:8001"}}); err != ErrEmptyNodeID {
		t.Fatalf("expected ErrEmptyNodeID, got %v", err)
	}

	// Duplicate ID
	duplicateNodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-1", Address: "http://localhost:8002"},
	}
	if _, err := NewRegistry(duplicateNodes); err == nil {
		t.Fatalf("expected duplicate node ID error, got nil")
	}

	// Invalid URL
	invalidAddrNodes := []Node{
		{ID: "node-1", Address: "not_a_valid_url"},
	}
	if _, err := NewRegistry(invalidAddrNodes); err == nil {
		t.Fatalf("expected invalid URL error, got nil")
	}

	// Ordering preservation
	orderedNodes := []Node{
		{ID: "node-z", Address: "http://localhost:8003"},
		{ID: "node-a", Address: "http://localhost:8001"},
		{ID: "node-m", Address: "http://localhost:8002"},
	}
	reg, err := NewRegistry(orderedNodes)
	if err != nil {
		t.Fatalf("unexpected error creating registry: %v", err)
	}

	if reg.Count() != 3 {
		t.Fatalf("expected count 3, got %d", reg.Count())
	}

	for i, expected := range orderedNodes {
		got, err := reg.GetNode(i)
		if err != nil || got.ID != expected.ID {
			t.Fatalf("node at index %d mismatch: got %v, expected %v", i, got, expected)
		}
	}
}

func TestParseNodeString(t *testing.T) {
	str := "node-1=http://localhost:8001, node-2=http://localhost:8002, node-3=http://localhost:8003"
	nodes, err := ParseNodeString(str)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[0].ID != "node-1" || nodes[0].Address != "http://localhost:8001" {
		t.Fatalf("node 0 mismatch: %+v", nodes[0])
	}

	// Empty string
	if _, err := ParseNodeString(""); err != ErrNoNodes {
		t.Fatalf("expected ErrNoNodes for empty string, got %v", err)
	}

	// Invalid format
	if _, err := ParseNodeString("node-1-without-equal-sign"); err == nil {
		t.Fatalf("expected format error, got nil")
	}
}

// -----------------------------------------------------------------------------
// Section 23: HTTP Router Tests
// -----------------------------------------------------------------------------

func TestRouterHealth(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	r := New(ring, DefaultConfig(9000))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", res.Status)
	}
}

func TestRouterNodesEndpoint(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	_ = ring.AddNode(Node{ID: "node-2", Address: "http://localhost:8002"})
	r := New(ring, DefaultConfig(9000))

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res struct {
		Nodes    []Node `json:"nodes"`
		Replicas int    `json:"replicas"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if len(res.Nodes) != 2 || res.Nodes[0].ID != "node-1" || res.Nodes[1].ID != "node-2" {
		t.Fatalf("nodes mismatch: %+v", res.Nodes)
	}
	if res.Replicas != DefaultReplicas {
		t.Fatalf("expected replicas %d, got %d", DefaultReplicas, res.Replicas)
	}
}

func TestRouterNodeUnavailable(t *testing.T) {
	// Point to an address where no server is listening
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "dead-node", Address: "http://127.0.0.1:59999"})
	r := New(ring, DefaultConfig(9000))

	req := httptest.NewRequest(http.MethodGet, "/cache/test-key", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	// Must return 502 Bad Gateway without panic or failover
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway for unavailable node, got %d", w.Code)
	}

	var res struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Error != "cache node unavailable" {
		t.Fatalf("expected 'cache node unavailable', got %q", res.Error)
	}
}

func TestRouterUnsupportedMethod(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	r := New(ring, DefaultConfig(9000))

	req := httptest.NewRequest(http.MethodPost, "/cache/key", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", w.Code)
	}
}

// -----------------------------------------------------------------------------
// Section 24 & 25: Critical Multi-Node Routing Integration Tests
// -----------------------------------------------------------------------------

// TestCriticalThreeNodeRouting verifies end-to-end routing across three active cache nodes.
func TestCriticalThreeNodeRouting(t *testing.T) {
	// Start 3 independent cache nodes using httptest
	n1, _ := node.New(node.Config{ID: "node-1", Host: "127.0.0.1", Port: 18001, Capacity: 10, Policy: "lru"})
	n2, _ := node.New(node.Config{ID: "node-2", Host: "127.0.0.1", Port: 18002, Capacity: 10, Policy: "lru"})
	n3, _ := node.New(node.Config{ID: "node-3", Host: "127.0.0.1", Port: 18003, Capacity: 10, Policy: "lru"})

	s1 := httptest.NewServer(n1.Server().Handler())
	defer s1.Close()

	s2 := httptest.NewServer(n2.Server().Handler())
	defer s2.Close()

	s3 := httptest.NewServer(n3.Server().Handler())
	defer s3.Close()

	backendNodes := []Node{
		{ID: "node-1", Address: s1.URL},
		{ID: "node-2", Address: s2.URL},
		{ID: "node-3", Address: s3.URL},
	}

	ring := NewHashRing(DefaultReplicas)
	for _, bn := range backendNodes {
		if err := ring.AddNode(bn); err != nil {
			t.Fatalf("failed to add node to ring: %v", err)
		}
	}

	r := New(ring, DefaultConfig(9000))
	routerServer := httptest.NewServer(r.Handler())
	defer routerServer.Close()

	client := routerServer.Client()

	// 1. Route key "A" through the router
	targetNode, err := r.Route("A")
	if err != nil {
		t.Fatalf("failed to route key 'A': %v", err)
	}

	// 2. PUT /cache/A via Router
	putURL := fmt.Sprintf("%s/cache/A", routerServer.URL)
	putReq, _ := http.NewRequest(http.MethodPut, putURL, bytes.NewBufferString(`{"value":"valA"}`))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := client.Do(putReq)
	if err != nil {
		t.Fatalf("failed to PUT via router: %v", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		t.Fatalf("expected 200 on PUT via router, got %d. Body: %s", putResp.StatusCode, string(body))
	}

	// 3. GET /cache/A via Router -> HIT
	getURL := fmt.Sprintf("%s/cache/A", routerServer.URL)
	getResp, err := client.Get(getURL)
	if err != nil {
		t.Fatalf("failed to GET via router: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on GET via router, got %d", getResp.StatusCode)
	}

	var getResult struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	_ = json.NewDecoder(getResp.Body).Decode(&getResult)
	if getResult.Key != "A" || getResult.Value != "valA" {
		t.Fatalf("expected key 'A' and value 'valA', got %+v", getResult)
	}

	// 4. Query each node directly: verify key exists ONLY on targetNode
	for _, n := range backendNodes {
		directURL := fmt.Sprintf("%s/cache/A", n.Address)
		resp, err := http.Get(directURL)
		if err != nil {
			t.Fatalf("failed direct GET to node %s: %v", n.ID, err)
		}
		_ = resp.Body.Close()

		if n.ID == targetNode.ID {
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected direct HIT on target node %s, got %d", n.ID, resp.StatusCode)
			}
		} else {
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected direct MISS (404) on non-target node %s, got %d", n.ID, resp.StatusCode)
			}
		}
	}

	// 5. DELETE /cache/A via Router
	delReq, _ := http.NewRequest(http.MethodDelete, getURL, nil)
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("failed to DELETE via router: %v", err)
	}
	_ = delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on DELETE via router, got %d", delResp.StatusCode)
	}

	// 6. Verify 404 propagation on subsequent GET
	subsequentGet, err := client.Get(getURL)
	if err != nil {
		t.Fatalf("failed subsequent GET via router: %v", err)
	}
	_ = subsequentGet.Body.Close()
	if subsequentGet.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 propagation after delete, got %d", subsequentGet.StatusCode)
	}
}

// TestMultipleKeysIntegration tests keys A through H across nodes with consistent hashing.
func TestMultipleKeysIntegration(t *testing.T) {
	n1, _ := node.New(node.Config{ID: "node-1", Host: "127.0.0.1", Port: 18001, Capacity: 20, Policy: "lru"})
	n2, _ := node.New(node.Config{ID: "node-2", Host: "127.0.0.1", Port: 18002, Capacity: 20, Policy: "lru"})
	n3, _ := node.New(node.Config{ID: "node-3", Host: "127.0.0.1", Port: 18003, Capacity: 20, Policy: "lru"})

	s1 := httptest.NewServer(n1.Server().Handler())
	defer s1.Close()
	s2 := httptest.NewServer(n2.Server().Handler())
	defer s2.Close()
	s3 := httptest.NewServer(n3.Server().Handler())
	defer s3.Close()

	backendNodes := []Node{
		{ID: "node-1", Address: s1.URL},
		{ID: "node-2", Address: s2.URL},
		{ID: "node-3", Address: s3.URL},
	}

	ring := NewHashRing(DefaultReplicas)
	for _, bn := range backendNodes {
		_ = ring.AddNode(bn)
	}

	r := New(ring, DefaultConfig(9000))
	routerServer := httptest.NewServer(r.Handler())
	defer routerServer.Close()

	client := routerServer.Client()
	keys := []string{"A", "B", "C", "D", "E", "F", "G", "H"}

	for _, k := range keys {
		val := "value-" + k
		expectedNode, _ := r.Route(k)

		// PUT via router
		putReq, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/cache/%s", routerServer.URL, k), bytes.NewBufferString(fmt.Sprintf(`{"value":%q}`, val)))
		putReq.Header.Set("Content-Type", "application/json")
		putResp, err := client.Do(putReq)
		if err != nil || putResp.StatusCode != http.StatusOK {
			t.Fatalf("failed PUT for key %q: %v", k, err)
		}
		_ = putResp.Body.Close()

		// GET via router
		getResp, err := client.Get(fmt.Sprintf("%s/cache/%s", routerServer.URL, k))
		if err != nil || getResp.StatusCode != http.StatusOK {
			t.Fatalf("failed GET for key %q: %v", k, err)
		}
		_ = getResp.Body.Close()

		// Verify key exists on expected node and NOT on others
		for _, bn := range backendNodes {
			directResp, err := http.Get(fmt.Sprintf("%s/cache/%s", bn.Address, k))
			if err != nil {
				t.Fatalf("failed direct GET: %v", err)
			}
			_ = directResp.Body.Close()

			if bn.ID == expectedNode.ID {
				if directResp.StatusCode != http.StatusOK {
					t.Fatalf("key %q expected on node %s, got status %d", k, bn.ID, directResp.StatusCode)
				}
			} else {
				if directResp.StatusCode != http.StatusNotFound {
					t.Fatalf("key %q NOT expected on node %s, got status %d", k, bn.ID, directResp.StatusCode)
				}
			}
		}
	}
}
