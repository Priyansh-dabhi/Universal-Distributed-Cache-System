package router

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// DefaultReplicas is the default number of virtual node positions per physical node.
const DefaultReplicas = 100

// ErrNodeNotFound is returned when attempting to operate on or remove a node that does not exist.
var ErrNodeNotFound = errors.New("node not found in hash ring")

// HashRing manages a circular consistent hash ring with virtual nodes.
// Read operations (GetNode, Nodes, etc.) execute concurrently under RLock.
// Mutation operations (AddNode, RemoveNode) execute under Lock.
type HashRing struct {
	mu            sync.RWMutex
	replicas      int
	hasher        Hasher
	nodes         map[string]Node
	ringPositions []uint32
	ringMap       map[uint32]Node
}

// NewHashRing constructs a new HashRing using FNV-1a hashing and the given replica count.
// If replicas <= 0, DefaultReplicas (100) is used.
func NewHashRing(replicas int) *HashRing {
	return NewHashRingWithHasher(replicas, NewFNVHasher())
}

// NewHashRingWithHasher constructs a new HashRing with a custom Hasher and replica count.
func NewHashRingWithHasher(replicas int, hasher Hasher) *HashRing {
	if replicas <= 0 {
		replicas = DefaultReplicas
	}
	if hasher == nil {
		hasher = NewFNVHasher()
	}
	return &HashRing{
		replicas:      replicas,
		hasher:        hasher,
		nodes:         make(map[string]Node),
		ringPositions: make([]uint32, 0),
		ringMap:       make(map[uint32]Node),
	}
}

// AddNode adds a physical node and all its virtual node positions to the hash ring.
// Returns an error if the node ID is empty, the address is invalid, or the node already exists.
func (r *HashRing) AddNode(node Node) error {
	id := strings.TrimSpace(node.ID)
	if id == "" {
		return ErrEmptyNodeID
	}

	addr := strings.TrimSpace(node.Address)
	if addr == "" {
		return fmt.Errorf("%w for node %q", ErrEmptyNodeAddress, id)
	}

	// Normalize: strip trailing slashes
	addr = strings.TrimRight(addr, "/")

	u, err := url.ParseRequestURI(addr)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%w %q for node %q", ErrInvalidNodeAddress, addr, id)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[id]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateNodeID, id)
	}

	r.nodes[id] = Node{
		ID:      id,
		Address: addr,
	}

	r.rebuildRingLocked()
	return nil
}

// RemoveNode removes a physical node and all its virtual node positions from the hash ring.
// Future key requests will automatically map clockwise to remaining nodes.
func (r *HashRing) RemoveNode(nodeID string) error {
	id := strings.TrimSpace(nodeID)
	if id == "" {
		return ErrEmptyNodeID
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[id]; !exists {
		return fmt.Errorf("%w: %q", ErrNodeNotFound, id)
	}

	delete(r.nodes, id)
	r.rebuildRingLocked()
	return nil
}

// GetNode routes a cache key to its owning physical node using consistent hashing.
// The key's hash position is located on the circular ring using binary search (O(log R)),
// walking clockwise to the next available virtual node position.
// Returns (node, true) if a node is found, or (Node{}, false) if the ring contains no nodes.
func (r *HashRing) GetNode(key string) (Node, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.ringPositions) == 0 {
		return Node{}, false
	}

	hashVal := r.hasher.Hash(key)

	// Binary search for the first ring position >= hashVal (clockwise lookup)
	idx := sort.Search(len(r.ringPositions), func(i int) bool {
		return r.ringPositions[i] >= hashVal
	})

	// Wrap around to position 0 if beyond the highest hash value on the ring
	if idx == len(r.ringPositions) {
		idx = 0
	}

	targetHash := r.ringPositions[idx]
	targetNode, exists := r.ringMap[targetHash]
	return targetNode, exists
}

// Nodes returns a snapshot copy of all physical nodes in the ring, sorted deterministically by ID.
func (r *HashRing) Nodes() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		result = append(result, n)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// Replicas returns the configured virtual node replica count per physical node.
func (r *HashRing) Replicas() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.replicas
}

// NodeCount returns the number of physical nodes currently registered in the ring.
func (r *HashRing) NodeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// VirtualNodeCount returns the total number of virtual node positions active on the ring.
func (r *HashRing) VirtualNodeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.ringPositions)
}

// Positions returns a copy of all sorted virtual node hash positions on the ring.
func (r *HashRing) Positions() []uint32 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]uint32, len(r.ringPositions))
	copy(res, r.ringPositions)
	return res
}

// Hasher returns the key hashing implementation used by this ring.
func (r *HashRing) Hasher() Hasher {
	return r.hasher
}

// rebuildRingLocked updates the internal ring positions and map.
// Caller must hold r.mu (write lock).
// Sorting node IDs guarantees that node registration order never affects virtual token positions.
// In the rare event of a hash collision between virtual nodes, linear probing (pos++)
// finds the next available unoccupied slot deterministically.
func (r *HashRing) rebuildRingLocked() {
	totalPositions := len(r.nodes) * r.replicas
	r.ringPositions = make([]uint32, 0, totalPositions)
	r.ringMap = make(map[uint32]Node, totalPositions)

	if len(r.nodes) == 0 {
		return
	}

	// Sort node IDs to guarantee order independence
	nodeIDs := make([]string, 0, len(r.nodes))
	for id := range r.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for _, id := range nodeIDs {
		node := r.nodes[id]
		for i := 0; i < r.replicas; i++ {
			vnodeToken := fmt.Sprintf("%s#%d", id, i)
			pos := r.hasher.Hash(vnodeToken)

			// Linear probing collision strategy:
			// If pos is already taken, probe pos+1 until an unoccupied slot is found.
			for {
				if _, exists := r.ringMap[pos]; !exists {
					break
				}
				pos++
			}

			r.ringMap[pos] = node
			r.ringPositions = append(r.ringPositions, pos)
		}
	}

	// Sort ring positions for O(log R) binary search
	sort.Slice(r.ringPositions, func(i, j int) bool {
		return r.ringPositions[i] < r.ringPositions[j]
	})
}
