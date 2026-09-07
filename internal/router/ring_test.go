package router

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

// 1. Basic routing: Add node, GetNode(key) should return a valid registered node.
func TestBasicRouting(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	node := Node{ID: "node-1", Address: "http://localhost:8001"}

	if err := ring.AddNode(node); err != nil {
		t.Fatalf("failed to add node: %v", err)
	}

	got, ok := ring.GetNode("my-test-key")
	if !ok {
		t.Fatalf("expected node to be found, got false")
	}
	if got.ID != "node-1" || got.Address != "http://localhost:8001" {
		t.Fatalf("unexpected node returned: %+v", got)
	}
}

// 2. Determinism: The same key with the same nodes must always return the same node.
func TestDeterminism(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	nodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-2", Address: "http://localhost:8002"},
		{ID: "node-3", Address: "http://localhost:8003"},
	}
	for _, n := range nodes {
		if err := ring.AddNode(n); err != nil {
			t.Fatalf("failed to add node %s: %v", n.ID, err)
		}
	}

	keys := []string{"alpha", "beta", "gamma", "user:42", "session:abc", "long_complex_cache_key_99999"}
	for _, key := range keys {
		firstNode, ok := ring.GetNode(key)
		if !ok {
			t.Fatalf("expected node for key %q", key)
		}

		for i := 0; i < 100; i++ {
			node, ok := ring.GetNode(key)
			if !ok || node.ID != firstNode.ID {
				t.Fatalf("non-deterministic routing for key %q on iteration %d: got %s, want %s",
					key, i, node.ID, firstNode.ID)
			}
		}
	}
}

// 3. Empty key: Ensure an empty string key behaves deterministically.
func TestEmptyKeyRouting(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	_ = ring.AddNode(Node{ID: "node-2", Address: "http://localhost:8002"})

	node1, ok1 := ring.GetNode("")
	node2, ok2 := ring.GetNode("")

	if !ok1 || !ok2 {
		t.Fatalf("expected empty key to route successfully")
	}
	if node1.ID != node2.ID {
		t.Fatalf("empty key routing not deterministic: %s vs %s", node1.ID, node2.ID)
	}
}

// 4. Single node: With one node, any key -> that node.
func TestSingleNode(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	expectedNode := Node{ID: "solo-node", Address: "http://localhost:8001"}
	_ = ring.AddNode(expectedNode)

	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("arbitrary-key-%d", i)
		n, ok := ring.GetNode(key)
		if !ok {
			t.Fatalf("expected node for key %q", key)
		}
		if n.ID != expectedNode.ID {
			t.Fatalf("expected single node %s, got %s", expectedNode.ID, n.ID)
		}
	}
}

// 5. Multiple nodes: Add several nodes and verify keys map across registered nodes.
func TestMultipleNodes(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	nodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-2", Address: "http://localhost:8002"},
		{ID: "node-3", Address: "http://localhost:8003"},
	}
	for _, n := range nodes {
		_ = ring.AddNode(n)
	}

	keyCounts := make(map[string]int)
	for i := 0; i < 600; i++ {
		key := fmt.Sprintf("key-%d", i)
		n, ok := ring.GetNode(key)
		if !ok {
			t.Fatalf("expected node for key %q", key)
		}
		keyCounts[n.ID]++
	}

	// Each node must receive a healthy share of the keys
	if len(keyCounts) != len(nodes) {
		t.Fatalf("expected all %d nodes to receive keys, got counts: %+v", len(nodes), keyCounts)
	}
	for id, count := range keyCounts {
		if count < 50 {
			t.Fatalf("node %s received too few keys (%d), distribution appears imbalanced", id, count)
		}
	}
}

// 6. Wrap-around: Test keys whose hash falls beyond the last ring position and ensure wrap-around to index 0.
func TestWrapAround(t *testing.T) {
	// Custom hasher to place nodes and keys at controlled hash positions
	customHasher := &mockHasher{
		hashes: map[string]uint32{
			"node-A#0": 1000,
			"node-B#0": 5000,
			"key-low":  500,  // <= 1000 -> node-A
			"key-mid":  3000, // <= 5000 -> node-B
			"key-high": 9000, // > 5000 -> wrap-around to node-A (position 1000)
		},
	}

	ring := NewHashRingWithHasher(1, customHasher)
	_ = ring.AddNode(Node{ID: "node-A", Address: "http://localhost:8001"})
	_ = ring.AddNode(Node{ID: "node-B", Address: "http://localhost:8002"})

	nodeLow, ok := ring.GetNode("key-low")
	if !ok || nodeLow.ID != "node-A" {
		t.Fatalf("expected node-A for key-low, got %s", nodeLow.ID)
	}

	nodeMid, ok := ring.GetNode("key-mid")
	if !ok || nodeMid.ID != "node-B" {
		t.Fatalf("expected node-B for key-mid, got %s", nodeMid.ID)
	}

	nodeHigh, ok := ring.GetNode("key-high")
	if !ok || nodeHigh.ID != "node-A" {
		t.Fatalf("expected wrap-around to node-A for key-high, got %s", nodeHigh.ID)
	}
}

type mockHasher struct {
	hashes map[string]uint32
}

func (m *mockHasher) Hash(key string) uint32 {
	if val, ok := m.hashes[key]; ok {
		return val
	}
	return NewFNVHasher().Hash(key)
}

// 7. Node addition: Adding a node must leave most existing keys mapped to their previous nodes.
func TestNodeAdditionRedistribution(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	initialNodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-2", Address: "http://localhost:8002"},
		{ID: "node-3", Address: "http://localhost:8003"},
	}
	for _, n := range initialNodes {
		_ = ring.AddNode(n)
	}

	numKeys := 2000
	initialMappings := make(map[string]string, numKeys)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		n, _ := ring.GetNode(key)
		initialMappings[key] = n.ID
	}

	// Add 4th node
	newNode := Node{ID: "node-4", Address: "http://localhost:8004"}
	if err := ring.AddNode(newNode); err != nil {
		t.Fatalf("failed to add node-4: %v", err)
	}

	movedKeys := 0
	for key, originalNode := range initialMappings {
		newNodeAfter, _ := ring.GetNode(key)
		if newNodeAfter.ID != originalNode {
			movedKeys++
			// Crucial consistent hashing property: keys that moved must ONLY move to the new node!
			if newNodeAfter.ID != "node-4" {
				t.Fatalf("key %q moved from %s to %s instead of new node-4", key, originalNode, newNodeAfter.ID)
			}
		}
	}

	movedPercentage := float64(movedKeys) / float64(numKeys) * 100.0
	t.Logf("Consistent hashing node addition: %d/%d keys moved (%.2f%%)", movedKeys, numKeys, movedPercentage)

	// In consistent hashing with 3 -> 4 nodes, expected movement is ~25%.
	// Assert that redistribution is between 15% and 40% (substantially less than modulo hashing's ~75%).
	if movedPercentage > 40.0 || movedPercentage < 15.0 {
		t.Fatalf("expected moved keys percentage between 15%% and 40%%, got %.2f%%", movedPercentage)
	}
}

// 8. Node removal: Removed node is never returned, affected keys move clockwise to remaining nodes.
func TestNodeRemoval(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	nodes := []Node{
		{ID: "node-1", Address: "http://localhost:8001"},
		{ID: "node-2", Address: "http://localhost:8002"},
		{ID: "node-3", Address: "http://localhost:8003"},
	}
	for _, n := range nodes {
		_ = ring.AddNode(n)
	}

	numKeys := 1000
	initialMappings := make(map[string]string, numKeys)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("removal-key-%d", i)
		n, _ := ring.GetNode(key)
		initialMappings[key] = n.ID
	}

	// Remove node-2
	if err := ring.RemoveNode("node-2"); err != nil {
		t.Fatalf("failed to remove node-2: %v", err)
	}

	for key, prevNode := range initialMappings {
		n, ok := ring.GetNode(key)
		if !ok {
			t.Fatalf("expected node for key %q", key)
		}
		if n.ID == "node-2" {
			t.Fatalf("removed node-2 was returned for key %q", key)
		}

		// If key was not on node-2, it should remain on its original node
		if prevNode != "node-2" && n.ID != prevNode {
			t.Fatalf("key %q unexpectedly moved from %s to %s when unrelated node-2 was removed",
				key, prevNode, n.ID)
		}
	}
}

// 9. Multiple virtual nodes: Verify each physical node creates multiple ring positions.
func TestMultipleVirtualNodes(t *testing.T) {
	replicas := 75
	ring := NewHashRing(replicas)

	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	if ring.VirtualNodeCount() != replicas {
		t.Fatalf("expected %d virtual nodes, got %d", replicas, ring.VirtualNodeCount())
	}

	_ = ring.AddNode(Node{ID: "node-2", Address: "http://localhost:8002"})
	if ring.VirtualNodeCount() != replicas*2 {
		t.Fatalf("expected %d virtual nodes, got %d", replicas*2, ring.VirtualNodeCount())
	}

	_ = ring.RemoveNode("node-1")
	if ring.VirtualNodeCount() != replicas {
		t.Fatalf("expected %d virtual nodes after removal, got %d", replicas, ring.VirtualNodeCount())
	}
}

// 10. Deterministic node ordering: Different insertion order must yield the exact same ring and routing.
func TestDeterministicNodeOrdering(t *testing.T) {
	node1 := Node{ID: "node-1", Address: "http://localhost:8001"}
	node2 := Node{ID: "node-2", Address: "http://localhost:8002"}
	node3 := Node{ID: "node-3", Address: "http://localhost:8003"}

	ring1 := NewHashRing(DefaultReplicas)
	_ = ring1.AddNode(node1)
	_ = ring1.AddNode(node2)
	_ = ring1.AddNode(node3)

	ring2 := NewHashRing(DefaultReplicas)
	_ = ring2.AddNode(node3)
	_ = ring2.AddNode(node1)
	_ = ring2.AddNode(node2)

	ring3 := NewHashRing(DefaultReplicas)
	_ = ring3.AddNode(node2)
	_ = ring3.AddNode(node3)
	_ = ring3.AddNode(node1)

	// Verify all virtual node positions match exactly across all 3 rings
	pos1 := ring1.Positions()
	pos2 := ring2.Positions()
	pos3 := ring3.Positions()

	if len(pos1) != len(pos2) || len(pos1) != len(pos3) {
		t.Fatalf("position lengths do not match: %d vs %d vs %d", len(pos1), len(pos2), len(pos3))
	}
	for i := range pos1 {
		if pos1[i] != pos2[i] || pos1[i] != pos3[i] {
			t.Fatalf("ring position mismatch at index %d: %d vs %d vs %d", i, pos1[i], pos2[i], pos3[i])
		}
	}

	// Verify 1000 keys map to the exact same nodes
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("order-independence-key-%d", i)
		n1, _ := ring1.GetNode(key)
		n2, _ := ring2.GetNode(key)
		n3, _ := ring3.GetNode(key)

		if n1.ID != n2.ID || n1.ID != n3.ID {
			t.Fatalf("routing mismatch for key %q: ring1=%s, ring2=%s, ring3=%s", key, n1.ID, n2.ID, n3.ID)
		}
	}
}

// 11. Duplicate node: Adding the same physical node twice should return an error.
func TestDuplicateNode(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	node := Node{ID: "node-1", Address: "http://localhost:8001"}

	if err := ring.AddNode(node); err != nil {
		t.Fatalf("first AddNode failed: %v", err)
	}

	err := ring.AddNode(node)
	if err == nil || !errors.Is(err, ErrDuplicateNodeID) {
		t.Fatalf("expected ErrDuplicateNodeID, got %v", err)
	}
}

// 12. Remove unknown node: Removing a node that does not exist should return ErrNodeNotFound.
func TestRemoveUnknownNode(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)
	err := ring.RemoveNode("ghost-node")
	if err == nil || !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}
}

// 13. No nodes: GetNode() should safely report that no node is available.
func TestNoNodes(t *testing.T) {
	ring := NewHashRing(DefaultReplicas)

	got, ok := ring.GetNode("some-key")
	if ok || got.ID != "" {
		t.Fatalf("expected (Node{}, false) on empty ring, got %+v, %v", got, ok)
	}

	if ring.NodeCount() != 0 || ring.VirtualNodeCount() != 0 {
		t.Fatalf("expected 0 counts, got nodeCount=%d, vnodeCount=%d", ring.NodeCount(), ring.VirtualNodeCount())
	}
}

// 14. Concurrent access: Ensure thread safety under concurrent reads and writes.
func TestConcurrentRingAccess(t *testing.T) {
	ring := NewHashRing(20)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	_ = ring.AddNode(Node{ID: "node-2", Address: "http://localhost:8002"})

	var readerWg sync.WaitGroup
	var writerWg sync.WaitGroup
	stopCh := make(chan struct{})

	// 100 concurrent readers
	for r := 0; r < 100; r++ {
		readerWg.Add(1)
		go func(readerID int) {
			defer readerWg.Done()
			i := 0
			for {
				select {
				case <-stopCh:
					return
				default:
					key := fmt.Sprintf("reader-%d-key-%d", readerID, i)
					node, ok := ring.GetNode(key)
					if ok && node.ID == "" {
						t.Errorf("empty node ID returned")
					}
					ring.Nodes()
					i++
				}
			}
		}(r)
	}

	// 4 concurrent writers adding/removing dynamic nodes
	for w := 0; w < 4; w++ {
		writerWg.Add(1)
		go func(writerID int) {
			defer writerWg.Done()
			for i := 0; i < 50; i++ {
				dynamicID := fmt.Sprintf("dynamic-node-%d-%d", writerID, i)
				_ = ring.AddNode(Node{ID: dynamicID, Address: "http://localhost:8999"})
				_ = ring.RemoveNode(dynamicID)
			}
		}(w)
	}

	// Wait for writers to finish
	writerWg.Wait()
	// Stop readers and wait for them to finish
	close(stopCh)
	readerWg.Wait()
}

// 15. Comparison Test With Modulo Hashing (Section 19)
// Demonstrate that consistent hashing remaps substantially fewer keys than modulo hashing
// when expanding from 3 to 4 nodes.
func TestModuloVsConsistentHashingComparison(t *testing.T) {
	const totalKeys = 10000
	hasher := NewFNVHasher()

	// -------------------------------------------------------------
	// 1. Modulo Hashing Simulation (Phase 8 strategy)
	// -------------------------------------------------------------
	initialNodeCountModulo := 3
	expandedNodeCountModulo := 4

	moduloMoved := 0
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("comparison-key-%d", i)
		h := hasher.Hash(key)

		oldNode := int(h % uint32(initialNodeCountModulo))
		newNode := int(h % uint32(expandedNodeCountModulo))

		if oldNode != newNode {
			moduloMoved++
		}
	}
	moduloMovedPercent := float64(moduloMoved) / float64(totalKeys) * 100.0

	// -------------------------------------------------------------
	// 2. Consistent Hashing Simulation (Phase 9 strategy)
	// -------------------------------------------------------------
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	_ = ring.AddNode(Node{ID: "node-2", Address: "http://localhost:8002"})
	_ = ring.AddNode(Node{ID: "node-3", Address: "http://localhost:8003"})

	initialRingOwners := make([]string, totalKeys)
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("comparison-key-%d", i)
		n, _ := ring.GetNode(key)
		initialRingOwners[i] = n.ID
	}

	// Add 4th node to ring
	_ = ring.AddNode(Node{ID: "node-4", Address: "http://localhost:8004"})

	consistentMoved := 0
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("comparison-key-%d", i)
		n, _ := ring.GetNode(key)
		if n.ID != initialRingOwners[i] {
			consistentMoved++
		}
	}
	consistentMovedPercent := float64(consistentMoved) / float64(totalKeys) * 100.0

	t.Logf("=== Redistribution Comparison (10,000 keys; 3 -> 4 nodes) ===")
	t.Logf("Modulo Hashing:     %d keys moved (%.2f%%)", moduloMoved, moduloMovedPercent)
	t.Logf("Consistent Hashing: %d keys moved (%.2f%%)", consistentMoved, consistentMovedPercent)

	// Modulo hashing mathematically moves ~75% of keys (3/4).
	if moduloMovedPercent < 65.0 {
		t.Fatalf("expected modulo hashing key movement > 65%%, got %.2f%%", moduloMovedPercent)
	}

	// Consistent hashing ideally moves ~25% of keys (1/4).
	if consistentMovedPercent > 35.0 {
		t.Fatalf("expected consistent hashing key movement < 35%%, got %.2f%%", consistentMovedPercent)
	}

	// Crucial assertion: Consistent hashing must remap far fewer keys than modulo hashing!
	if consistentMoved >= moduloMoved {
		t.Fatalf("consistent hashing (%d) did not remap fewer keys than modulo (%d)",
			consistentMoved, moduloMoved)
	}
}

func BenchmarkHashRingGetNode(b *testing.B) {
	ring := NewHashRing(DefaultReplicas)
	for i := 1; i <= 10; i++ {
		_ = ring.AddNode(Node{
			ID:      fmt.Sprintf("node-%d", i),
			Address: fmt.Sprintf("http://localhost:%d", 8000+i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i%1000)
		_, _ = ring.GetNode(key)
	}
}
