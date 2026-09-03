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

// Router dispatches client requests to appropriate backend cache nodes using hash-based routing.
type Router struct {
	cfg        Config
	registry   *Registry
	hasher     Hasher
	client     *http.Client
	httpServer *http.Server
	handler    http.Handler
}

// New creates and initializes a new Router with the provided registry, hasher, and configuration.
func New(registry *Registry, hasher Hasher, cfg Config) *Router {
	r := &Router{
		cfg:      cfg,
		registry: registry,
		hasher:   hasher,
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

// Route computes the deterministic target node for a given key using modulo hashing: hash(key) % N.
func (r *Router) Route(key string) (Node, error) {
	if key == "" {
		return Node{}, ErrEmptyKey
	}

	count := r.registry.Count()
	if count == 0 {
		return Node{}, ErrNoNodes
	}

	hashVal := r.hasher.Hash(key)
	index := int(hashVal % uint32(count))

	return r.registry.GetNode(index)
}

// Registry returns the router's active node registry.
func (r *Router) Registry() *Registry {
	return r.registry
}

// Hasher returns the router's key hashing mechanism.
func (r *Router) Hasher() Hasher {
	return r.hasher
}

// Handler returns the HTTP handler for testing and routing.
func (r *Router) Handler() http.Handler {
	return r.handler
}

// Start begins listening and serving HTTP requests.
func (r *Router) Start() error {
	log.Printf("Starting distributed cache router on %s:%d (nodes: %d)...",
		r.cfg.Host, r.cfg.Port, r.registry.Count())
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
	writeRouterJSON(w, http.StatusOK, map[string][]Node{"nodes": r.registry.Nodes()})
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
		// Node is unavailable or timed out; do NOT failover in Phase 8
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
