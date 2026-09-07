package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"universal-distributed-cache/internal/cache"
	"universal-distributed-cache/internal/server"
)

var (
	// ErrEmptyNodeID is returned when the node ID is empty or only whitespace.
	ErrEmptyNodeID = errors.New("node ID must not be empty")
	// ErrInvalidPort is returned when the port is out of the valid TCP port range (1-65535).
	ErrInvalidPort = errors.New("port must be between 1 and 65535")
	// ErrInvalidCapacity is returned when the cache capacity is less than or equal to 0.
	ErrInvalidCapacity = errors.New("capacity must be greater than 0")
	// ErrInvalidPolicy is returned when the eviction policy is not supported.
	ErrInvalidPolicy = errors.New("invalid eviction policy")
)

// Config encapsulates configuration parameters for an independent cache node.
type Config struct {
	ID       string
	Host     string
	Port     int
	Capacity int
	Policy   string
}

// DefaultConfig returns sensible default settings for a cache node.
func DefaultConfig() Config {
	return Config{
		ID:       "node-1",
		Host:     "127.0.0.1",
		Port:     8080,
		Capacity: 100,
		Policy:   "lru",
	}
}

// Validate checks the node configuration for correctness.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return ErrEmptyNodeID
	}
	if c.Port <= 0 || c.Port > 65535 {
		return ErrInvalidPort
	}
	if c.Capacity <= 0 {
		return ErrInvalidCapacity
	}

	switch strings.ToLower(strings.TrimSpace(c.Policy)) {
	case "lru", "lfu", "2q":
		return nil
	default:
		return fmt.Errorf("%w: %q (supported: lru, lfu, 2q)", ErrInvalidPolicy, c.Policy)
	}
}

// Node represents an independent cache node owning its configuration, cache, and HTTP server.
type Node struct {
	cfg    Config
	cache  *cache.Cache
	server *server.Server
}

// New constructs and initializes a new Node with its own independent cache and HTTP server.
func New(cfg Config) (*Node, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var policy cache.EvictionPolicy
	switch strings.ToLower(strings.TrimSpace(cfg.Policy)) {
	case "lru":
		policy = cache.PolicyLRU
	case "lfu":
		policy = cache.PolicyLFU
	case "2q":
		policy = cache.Policy2Q
	}

	c, err := cache.NewWithPolicy(cfg.Capacity, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize node cache: %w", err)
	}

	srvCfg := server.Config{
		NodeID:       cfg.ID,
		Host:         cfg.Host,
		Port:         cfg.Port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srv := server.New(c, srvCfg)

	return &Node{
		cfg:    cfg,
		cache:  c,
		server: srv,
	}, nil
}

// ID returns the node's unique identifier.
func (n *Node) ID() string {
	return n.cfg.ID
}

// Host returns the node's host address.
func (n *Node) Host() string {
	return n.cfg.Host
}

// Port returns the node's listening port.
func (n *Node) Port() int {
	return n.cfg.Port
}

// Config returns the node configuration.
func (n *Node) Config() Config {
	return n.cfg
}

// Cache returns the node's isolated cache instance.
func (n *Node) Cache() *cache.Cache {
	return n.cache
}

// Server returns the node's HTTP server instance.
func (n *Node) Server() *server.Server {
	return n.server
}

// Start logs startup details and starts the node's HTTP server.
func (n *Node) Start() error {
	log.Printf("Starting cache node\nid=%s\naddress=%s:%d\ncapacity=%d\npolicy=%s",
		n.cfg.ID, n.cfg.Host, n.cfg.Port, n.cfg.Capacity, n.cfg.Policy)
	return n.server.Start()
}

// Serve accepts incoming connections on a custom net.Listener.
func (n *Node) Serve(l net.Listener) error {
	return n.server.Serve(l)
}

// Shutdown gracefully terminates the node's HTTP server.
func (n *Node) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down cache node %s", n.cfg.ID)
	return n.server.Shutdown(ctx)
}
