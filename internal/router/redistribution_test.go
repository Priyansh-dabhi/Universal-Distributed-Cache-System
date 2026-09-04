package router

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
)

// TestConsistentHashNodeAdditionExperiment demonstrates key remapping when scaling 3 -> 4 nodes.
func TestConsistentHashNodeAdditionExperiment(t *testing.T) {
	const totalKeys = 10000

	// 1. Modulo Hashing (N=3 -> N=4)
	moduloHash := func(key string, n int) int {
		h := sha256.Sum256([]byte(key))
		val := binary.BigEndian.Uint32(h[:4])
		return int(val % uint32(n))
	}

	initialModulo := make([]int, totalKeys)
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("exp-key-%d", i)
		initialModulo[i] = moduloHash(key, 3)
	}

	moduloRemapped := 0
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("exp-key-%d", i)
		if moduloHash(key, 4) != initialModulo[i] {
			moduloRemapped++
		}
	}
	moduloPct := float64(moduloRemapped) / float64(totalKeys) * 100.0

	// 2. Consistent Hashing (3 nodes -> 4 nodes)
	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	_ = ring.AddNode(Node{ID: "node-2", Address: "http://localhost:8002"})
	_ = ring.AddNode(Node{ID: "node-3", Address: "http://localhost:8003"})

	initialRing := make([]string, totalKeys)
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("exp-key-%d", i)
		n, ok := ring.GetNode(key)
		if !ok {
			t.Fatalf("unexpected GetNode failure for key: %s", key)
		}
		initialRing[i] = n.ID
	}

	// Add 4th node
	_ = ring.AddNode(Node{ID: "node-4", Address: "http://localhost:8004"})

	consistentRemapped := 0
	node4Count := 0
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("exp-key-%d", i)
		n, _ := ring.GetNode(key)
		if n.ID != initialRing[i] {
			consistentRemapped++
			if n.ID == "node-4" {
				node4Count++
			}
		}
	}
	consistentPct := float64(consistentRemapped) / float64(totalKeys) * 100.0

	t.Logf("=== Scaling Experiment (10,000 Keys; 3 -> 4 Nodes) ===")
	t.Logf("Modulo Hashing Remapped:     %5d / %d (%.2f%%)", moduloRemapped, totalKeys, moduloPct)
	t.Logf("Consistent Hashing Remapped: %5d / %d (%.2f%%)", consistentRemapped, totalKeys, consistentPct)
	t.Logf("Keys Absorbed by Node 4:     %5d / %d (%.2f%%)", node4Count, totalKeys, float64(node4Count)/float64(totalKeys)*100.0)

	// In modulo hashing, ~75% of keys change owner.
	// In consistent hashing, only ~25% of keys (those taken by node-4) change owner.
	if consistentRemapped >= moduloRemapped {
		t.Errorf("consistent hashing (%d) should move far fewer keys than modulo (%d)", consistentRemapped, moduloRemapped)
	}
	if consistentRemapped != node4Count {
		t.Errorf("expected all remapped keys to go to the new node, got %d remapped vs %d on node-4", consistentRemapped, node4Count)
	}
}

// TestConsistentHashNodeRemovalExperiment demonstrates key remapping when removing 1 node (3 -> 2 nodes).
func TestConsistentHashNodeRemovalExperiment(t *testing.T) {
	const totalKeys = 10000

	ring := NewHashRing(DefaultReplicas)
	_ = ring.AddNode(Node{ID: "node-1", Address: "http://localhost:8001"})
	_ = ring.AddNode(Node{ID: "node-2", Address: "http://localhost:8002"})
	_ = ring.AddNode(Node{ID: "node-3", Address: "http://localhost:8003"})

	initialOwners := make([]string, totalKeys)
	node2KeysBefore := 0
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("rem-key-%d", i)
		n, _ := ring.GetNode(key)
		initialOwners[i] = n.ID
		if n.ID == "node-2" {
			node2KeysBefore++
		}
	}

	// Remove node-2
	_ = ring.RemoveNode("node-2")

	keysMoved := 0
	unaffectedMoved := 0
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("rem-key-%d", i)
		n, _ := ring.GetNode(key)
		if n.ID != initialOwners[i] {
			keysMoved++
			if initialOwners[i] != "node-2" {
				unaffectedMoved++
			}
		}
	}

	movedPct := float64(keysMoved) / float64(totalKeys) * 100.0
	t.Logf("=== Node Removal Experiment (10,000 Keys; 3 -> 2 Nodes) ===")
	t.Logf("Keys previously on node-2:   %5d (%.2f%%)", node2KeysBefore, float64(node2KeysBefore)/float64(totalKeys)*100.0)
	t.Logf("Total keys remapped:         %5d (%.2f%%)", keysMoved, movedPct)
	t.Logf("Unaffected node keys moved:  %5d", unaffectedMoved)

	// When node-2 is removed, only node-2's keys should migrate to remaining nodes.
	// Keys on node-1 and node-3 should not be disturbed at all!
	if unaffectedMoved != 0 {
		t.Errorf("keys belonging to unaffected nodes should not move, but %d moved", unaffectedMoved)
	}
	if keysMoved != node2KeysBefore {
		t.Errorf("expected exactly node-2 keys (%d) to move, but %d moved", node2KeysBefore, keysMoved)
	}
}

// BenchmarkConsistentHashRingLookup measures GetNode latency across cluster sizes and replica counts.
func BenchmarkConsistentHashRingLookup(b *testing.B) {
	nodeCounts := []int{3, 10, 50}
	replicas := []int{50, DefaultReplicas, 300}

	for _, n := range nodeCounts {
		for _, r := range replicas {
			name := fmt.Sprintf("nodes=%d/replicas=%d", n, r)
			b.Run(name, func(b *testing.B) {
				ring := NewHashRing(r)
				for i := 1; i <= n; i++ {
					_ = ring.AddNode(Node{
						ID:      fmt.Sprintf("node-%d", i),
						Address: fmt.Sprintf("http://localhost:%d", 8000+i),
					})
				}

				keys := make([]string, 1000)
				for i := 0; i < 1000; i++ {
					keys[i] = fmt.Sprintf("lookup-key-%d", i)
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = ring.GetNode(keys[i%1000])
				}
			})
		}
	}
}
