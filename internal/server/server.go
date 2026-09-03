package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"universal-distributed-cache/internal/cache"
	"universal-distributed-cache/internal/metrics"
)

// Cache defines the interface required by the HTTP server from the underlying cache engine.
type Cache interface {
	Set(key, value string)
	SetWithTTL(key, value string, ttl time.Duration)
	Get(key string) (string, bool)
	Delete(key string) bool
	Size() int
	Capacity() int
	Policy() cache.EvictionPolicy
	Metrics() *metrics.CacheMetrics
}

// Config contains configuration options for the HTTP server and node identity.
type Config struct {
	NodeID       string
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig returns reasonable default timeouts and configuration for the cache HTTP server.
func DefaultConfig(port int) Config {
	return Config{
		NodeID:       "node-1",
		Host:         "127.0.0.1",
		Port:         port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Server wraps an HTTP server and delegates cache operations to the Cache interface.
type Server struct {
	cache       Cache
	cfg         Config
	httpMetrics *metrics.HTTPMetrics
	httpServer  *http.Server
	handler     http.Handler
}

// New creates and initializes a new HTTP Server.
func New(c Cache, cfg Config) *Server {
	s := &Server{
		cache:       c,
		cfg:         cfg,
		httpMetrics: metrics.NewHTTPMetrics(),
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	if cfg.Host != "" {
		addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	}

	s.handler = s.routes()
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// Start starts listening and serving HTTP requests.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Serve starts the server on a provided net.Listener (useful for testing and custom listeners).
func (s *Server) Serve(l net.Listener) error {
	return s.httpServer.Serve(l)
}

// Shutdown gracefully stops the HTTP server with the provided context.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Handler returns the server's http.Handler for testing purposes.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Config returns the server's configuration.
func (s *Server) Config() Config {
	return s.cfg
}

// Metrics returns the server's HTTP telemetry metrics.
func (s *Server) Metrics() *metrics.HTTPMetrics {
	return s.httpMetrics
}
