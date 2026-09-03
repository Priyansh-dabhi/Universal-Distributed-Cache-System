# Architecture

## Current Architecture (Phase 7 — Multiple Independent Cache Nodes)

At present, the system supports running multiple independent cache nodes simultaneously as separate processes. Each node encapsulates its own configuration, HTTP transport layer, and isolated in-memory cache engine:

```text
                         Clients
                    /       |       \
                   /        |        \
                  ↓         ↓         ↓
             ┌────────┐ ┌────────┐ ┌────────┐
             │ Node 1 │ │ Node 2 │ │ Node 3 │
             │ :8001  │ │ :8002  │ │ :8003  │
             └───┬────┘ └───┬────┘ └───┬────┘
                 ↓           ↓           ↓
              Cache 1     Cache 2     Cache 3
```

### Key Principles of Phase 7

* **Process Isolation**: Each node runs as an independent operating system process.
* **Memory Isolation**: Each node constructs and owns its own isolated `cache.Cache` instance in memory. There is **no shared memory** between nodes.
* **Independent Policies**: Each node can be configured with its own eviction policy (LRU, LFU, 2Q) and maximum capacity.
* **Independent TTL**: Key expiration is tracked locally within each node.
* **No Node-to-Node Communication**: Nodes do not communicate with one another; there is no replication, failover, or consensus mechanism in this phase.
* **No Routing Yet**: Clients interact directly with individual nodes via their respective ports. Requests are not automatically distributed.

---

### Internal Node Architecture

```text
                        HTTP Client
                             |
                             v
                     ┌─────────────────┐
                     │   Node Entity   │
                     │  (internal/node)│
                     │                 │
                     │  HTTP Server    │
                     │  GET / SET /    │
                     │  DELETE / node  │
                     │  health         │
                     └────────┬────────┘
                              |
                              v
                         Cache Engine
                      (internal/cache)
                              |
                 ┌────────────┼────────────┐
                 ↓            ↓            ↓
                LRU          LFU           2Q
                              |
                             TTL
```

### Components

* **Node (`internal/node`)**: Coordinates the node identity (`id`, `host`, `port`), startup configuration validation, lifecycle management, and logging.
* **HTTP Server (`internal/server`)**: Transport layer exposing REST endpoints (`/health`, `/node`, `/cache`, `/cache/{key}`) and enforcing graceful shutdown.
* **Cache Engine (`internal/cache`)**: Standalone thread-safe key-value store with pluggable eviction (LRU, LFU, 2Q) and lazy TTL expiration.

---

## Planned Architecture (Phase 8 — Router & Request Distribution)

In Phase 8, an intelligent HTTP router will be placed in front of the independent cache nodes to automatically distribute requests across the cluster:

```text
                         Clients
                            |
                            v
                         Router (Phase 8 — Planned)
                      /     |     \
                     ↓      ↓      ↓
                  Node 1  Node 2  Node 3
                  :8001   :8002   :8003
```

### Planned Components

* **Router**: Entrypoint that hashes request keys and routes them to appropriate cache nodes using consistent hashing.
* **Consistent Hashing**: Hash ring to evenly balance load and minimize cache misses during node additions or removals.
* **Node Membership**: Dynamic discovery or configuration of active backend cache nodes.
* **Metrics & Observability**: Cluster-wide latency, hit/miss ratios, and throughput tracking.
* **Future Replication & Failover**: Data replication across secondary nodes and automated failover handling.
