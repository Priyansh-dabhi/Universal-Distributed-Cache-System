package router

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

var (
	// ErrNoNodes is returned when no cache nodes are configured for the router.
	ErrNoNodes = errors.New("at least one cache node must be configured")
	// ErrEmptyNodeID is returned when a node ID is empty or whitespace.
	ErrEmptyNodeID = errors.New("node ID must not be empty")
	// ErrEmptyNodeAddress is returned when a node address is empty.
	ErrEmptyNodeAddress = errors.New("node address must not be empty")
	// ErrInvalidNodeAddress is returned when a node address is not a valid HTTP/HTTPS URL.
	ErrInvalidNodeAddress = errors.New("node address must be a valid HTTP or HTTPS URL")
	// ErrDuplicateNodeID is returned when multiple nodes share the same ID.
	ErrDuplicateNodeID = errors.New("duplicate node ID detected")
)

// Node represents a backend cache node known to the router.
type Node struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// Registry maintains an ordered, thread-safe collection of backend cache nodes.
type Registry struct {
	mu    sync.RWMutex
	nodes []Node
}

// ParseNodeString parses a comma-separated list of "id=address" tokens into a slice of Nodes.
// Example: "node-1=http://localhost:8001,node-2=http://localhost:8002"
func ParseNodeString(nodesStr string) ([]Node, error) {
	trimmed := strings.TrimSpace(nodesStr)
	if trimmed == "" {
		return nil, ErrNoNodes
	}

	parts := strings.Split(trimmed, ",")
	nodes := make([]Node, 0, len(parts))

	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		subParts := strings.SplitN(token, "=", 2)
		if len(subParts) != 2 {
			return nil, fmt.Errorf("invalid node format %q, expected 'id=address'", token)
		}

		id := strings.TrimSpace(subParts[0])
		addr := strings.TrimSpace(subParts[1])

		nodes = append(nodes, Node{
			ID:      id,
			Address: addr,
		})
	}

	if len(nodes) == 0 {
		return nil, ErrNoNodes
	}

	return nodes, nil
}

// NewRegistry constructs, validates, and initializes a new Registry.
// Node ordering is preserved deterministically for consistent modulo hashing.
func NewRegistry(nodes []Node) (*Registry, error) {
	if len(nodes) == 0 {
		return nil, ErrNoNodes
	}

	seenIDs := make(map[string]bool, len(nodes))
	validatedNodes := make([]Node, 0, len(nodes))

	for _, n := range nodes {
		id := strings.TrimSpace(n.ID)
		if id == "" {
			return nil, ErrEmptyNodeID
		}

		if seenIDs[id] {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateNodeID, id)
		}
		seenIDs[id] = true

		addr := strings.TrimSpace(n.Address)
		if addr == "" {
			return nil, fmt.Errorf("%w for node %q", ErrEmptyNodeAddress, id)
		}

		// Normalize: strip trailing slashes
		addr = strings.TrimRight(addr, "/")

		u, err := url.ParseRequestURI(addr)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, fmt.Errorf("%w %q for node %q", ErrInvalidNodeAddress, addr, id)
		}

		validatedNodes = append(validatedNodes, Node{
			ID:      id,
			Address: addr,
		})
	}

	return &Registry{
		nodes: validatedNodes,
	}, nil
}

// Nodes returns a copy of all registered nodes in deterministic order.
func (r *Registry) Nodes() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Node, len(r.nodes))
	copy(result, r.nodes)
	return result
}

// Count returns the total number of registered nodes.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.nodes)
}

// GetNode returns the node at the specified index, or an error if out of range.
func (r *Registry) GetNode(index int) (Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if index < 0 || index >= len(r.nodes) {
		return Node{}, fmt.Errorf("node index %d out of bounds (count: %d)", index, len(r.nodes))
	}
	return r.nodes[index], nil
}
