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

	"universal-distributed-cache/internal/router"
)

func main() {
	host := flag.String("host", "127.0.0.1", "Router host or bind address (default 127.0.0.1)")
	port := flag.Int("port", 9000, "Router HTTP server port (default 9000)")
	nodesStr := flag.String("nodes", "", "Comma-separated cache nodes (e.g. node-1=http://localhost:8001,node-2=http://localhost:8002)")
	flag.Parse()

	if *nodesStr == "" {
		log.Fatalf("router configuration error: --nodes must be provided (e.g. --nodes node-1=http://localhost:8001,node-2=http://localhost:8002)")
	}

	nodes, err := router.ParseNodeString(*nodesStr)
	if err != nil {
		log.Fatalf("invalid node configuration: %v", err)
	}

	registry, err := router.NewRegistry(nodes)
	if err != nil {
		log.Fatalf("failed to initialize node registry: %v", err)
	}

	hasher := router.NewFNVHasher()
	cfg := router.DefaultConfig(*port)
	cfg.Host = *host

	r := router.New(registry, hasher, cfg)

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := r.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("router HTTP server error: %v", err)
		}
	}()

	<-shutdownCh
	log.Println("Received termination signal, shutting down router gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.Shutdown(ctx); err != nil {
		log.Fatalf("router shutdown error: %v", err)
	}

	fmt.Println("Distributed cache router stopped successfully.")
}
