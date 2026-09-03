package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"universal-distributed-cache/internal/cache"
	"universal-distributed-cache/internal/server"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port (default 8080)")
	capacity := flag.Int("capacity", 100, "Maximum cache capacity (default 100)")
	policyStr := flag.String("policy", "lru", "Eviction policy: lru, lfu, 2q (default lru)")
	flag.Parse()

	if *capacity <= 0 {
		log.Fatalf("invalid capacity: %d (must be greater than 0)", *capacity)
	}

	var policy cache.EvictionPolicy
	switch strings.ToLower(strings.TrimSpace(*policyStr)) {
	case "lru":
		policy = cache.PolicyLRU
	case "lfu":
		policy = cache.PolicyLFU
	case "2q":
		policy = cache.Policy2Q
	default:
		log.Fatalf("invalid eviction policy: %q (supported: lru, lfu, 2q)", *policyStr)
	}

	c, err := cache.NewWithPolicy(*capacity, policy)
	if err != nil {
		log.Fatalf("failed to initialize cache: %v", err)
	}

	cfg := server.DefaultConfig(*port)
	srv := server.New(c, cfg)

	// Channel to listen for OS termination signals
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting Universal Distributed Cache Server on port %d [capacity: %d, policy: %s]...", *port, *capacity, policy)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-shutdownCh
	log.Println("Received termination signal, shutting down cache server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server shutdown error: %v", err)
	}

	fmt.Println("Cache server stopped successfully.")
}
