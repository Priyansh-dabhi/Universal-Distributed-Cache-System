package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrEmptyKey is returned when an empty key is provided for routing.
var ErrEmptyKey = errors.New("key must not be empty")

// Config holds configuration parameters for the HTTP Router.
type Config struct {
	Host          string
	Port          int
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
	ClientTimeout time.Duration
}

// DefaultConfig returns reasonable default timeouts for the Router.
func DefaultConfig(port int) Config {
	return Config{
		Host:          "127.0.0.1",
		Port:          port,
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  10 * time.Second,
		IdleTimeout:   60 * time.Second,
		ClientTimeout: 5 * time.Second,
	}
}

// Router dispatches client requests to appropriate backend cache nodes using consistent hashing.
type Router struct {
	cfg        Config
	ring       *HashRing
	client     *http.Client
	httpServer *http.Server
	handler    http.Handler
}

// New creates and initializes a new Router with the provided consistent HashRing and configuration.
func New(ring *HashRing, cfg Config) *Router {
	r := &Router{
		cfg:  cfg,
		ring: ring,
		client: &http.Client{
			Timeout: cfg.ClientTimeout,
		},
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	if cfg.Host != "" {
		addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	}

	r.handler = r.routes()
	r.httpServer = &http.Server{
		Addr:         addr,
		Handler:      r.handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return r
}

// NewWithRegistry initializes a Router by populating a new HashRing from a Registry.
func NewWithRegistry(registry *Registry, hasher Hasher, cfg Config) *Router {
	ring := NewHashRingWithHasher(DefaultReplicas, hasher)
	if registry != nil {
		for _, n := range registry.Nodes() {
			_ = ring.AddNode(n)
		}
	}
	return New(ring, cfg)
}

// Route computes the target backend node for a given key by querying the consistent HashRing.
func (r *Router) Route(key string) (Node, error) {
	if key == "" {
		return Node{}, ErrEmptyKey
	}

	targetNode, ok := r.ring.GetNode(key)
	if !ok {
		return Node{}, ErrNoNodes
	}

	return targetNode, nil
}

// Ring returns the router's underlying HashRing.
func (r *Router) Ring() *HashRing {
	return r.ring
}

// Nodes returns a snapshot of physical nodes registered with the router.
func (r *Router) Nodes() []Node {
	return r.ring.Nodes()
}

// Handler returns the HTTP handler for testing and routing.
func (r *Router) Handler() http.Handler {
	return r.handler
}

// Start begins listening and serving HTTP requests.
func (r *Router) Start() error {
	log.Printf("Starting distributed cache router on %s:%d (nodes: %d, replicas: %d)...",
		r.cfg.Host, r.cfg.Port, r.ring.NodeCount(), r.ring.Replicas())
	return r.httpServer.ListenAndServe()
}

// Serve accepts incoming connections on a custom listener.
func (r *Router) Serve(l net.Listener) error {
	return r.httpServer.Serve(l)
}

// Shutdown gracefully terminates the router's HTTP server.
func (r *Router) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down cache router...")
	return r.httpServer.Shutdown(ctx)
}

func (r *Router) routes() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/health" {
			r.handleHealth(w, req)
			return
		}

		if req.URL.Path == "/nodes" {
			r.handleNodes(w, req)
			return
		}

		if strings.HasPrefix(req.URL.Path, "/cache/") {
			r.handleCacheProxy(w, req)
			return
		}

		writeRouterError(w, http.StatusNotFound, "not found")
	})
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeRouterError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeRouterJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Router) handleNodes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeRouterError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeRouterJSON(w, http.StatusOK, map[string]interface{}{
		"nodes":    r.ring.Nodes(),
		"replicas": r.ring.Replicas(),
	})
}

func (r *Router) handleCacheProxy(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodPut && req.Method != http.MethodDelete {
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeRouterError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var rawKey string
	if req.URL.RawPath != "" {
		rawKey = strings.TrimPrefix(req.URL.RawPath, "/cache/")
	} else {
		rawKey = strings.TrimPrefix(req.URL.Path, "/cache/")
	}

	key, err := url.PathUnescape(rawKey)
	if err != nil {
		writeRouterError(w, http.StatusBadRequest, "invalid URL encoding in key")
		return
	}

	if key == "" {
		writeRouterError(w, http.StatusBadRequest, "key is required")
		return
	}

	targetNode, err := r.Route(key)
	if err != nil {
		if errors.Is(err, ErrNoNodes) {
			writeRouterError(w, http.StatusServiceUnavailable, "no cache nodes available")
			return
		}
		writeRouterError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Construct target destination URL: {node.Address}/cache/{escapedKey}
	targetURL := fmt.Sprintf("%s/cache/%s", targetNode.Address, url.PathEscape(key))

	outboundReq, err := http.NewRequestWithContext(req.Context(), req.Method, targetURL, req.Body)
	if err != nil {
		writeRouterError(w, http.StatusInternalServerError, "failed to create proxy request")
		return
	}

	// Copy incoming request headers (such as Content-Type)
	for headerKey, headerVals := range req.Header {
		for _, val := range headerVals {
			outboundReq.Header.Add(headerKey, val)
		}
	}

	resp, err := r.client.Do(outboundReq)
	if err != nil {
		// Node is unavailable or timed out; do NOT failover in Phase 9
		writeRouterError(w, http.StatusBadGateway, "cache node unavailable")
		return
	}
	defer resp.Body.Close()

	// Mirror backend node response headers
	for headerKey, headerVals := range resp.Header {
		for _, val := range headerVals {
			w.Header().Add(headerKey, val)
		}
	}

	// Mirror backend node status code
	w.WriteHeader(resp.StatusCode)

	// Stream backend response body to the client
	_, _ = io.Copy(w, resp.Body)
}

func writeRouterJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeRouterError(w http.ResponseWriter, status int, message string) {
	writeRouterJSON(w, status, map[string]string{"error": message})
}
