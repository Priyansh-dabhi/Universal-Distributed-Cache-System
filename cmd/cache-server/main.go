package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"universal-distributed-cache/internal/node"
)

func main() {
	id := flag.String("id", "node-1", "Unique node identifier (default node-1)")
	host := flag.String("host", "127.0.0.1", "Node host or bind address (default 127.0.0.1)")
	port := flag.Int("port", 8080, "HTTP server port (default 8080)")
	capacity := flag.Int("capacity", 100, "Maximum cache capacity (default 100)")
	policyStr := flag.String("policy", "lru", "Eviction policy: lru, lfu, 2q (default lru)")
	flag.Parse()

	cfg := node.Config{
		ID:       *id,
		Host:     *host,
		Port:     *port,
		Capacity: *capacity,
		Policy:   *policyStr,
	}

	n, err := node.New(cfg)
	if err != nil {
		log.Fatalf("node initialization error: %v", err)
	}

	// Channel to listen for OS termination signals
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := n.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-shutdownCh
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := n.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server shutdown error: %v", err)
	}

	fmt.Printf("Cache node %s stopped successfully.\n", n.ID())
}
