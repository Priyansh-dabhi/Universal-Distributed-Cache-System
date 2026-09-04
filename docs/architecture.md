# Architecture

## Current Architecture (Phase 11 — Benchmarking & Performance Evaluation)

The system implements a distributed cache cluster where client requests are routed through a central HTTP reverse proxy router backed by a **Consistent Hash Ring** with virtual nodes, hardened for high concurrent throughput, and instrumented with lock-free atomic telemetry:

```text
                    Client
                       │
                       ▼
                 ┌───────────┐
                 │  Router   │
                 │  (:9000)  │
                 └─────┬─────┘
                       │
                Consistent Hash
                     Ring
                       │
              ┌────────┼────────┐
              │        │        │
              ▼        ▼        ▼
            Node 1   Node 2   Node 3
            :8001    :8002    :8003
              │        │        │
              ▼        ▼        ▼
            Cache    Cache    Cache
            Engine   Engine   Engine
              │        │        │
              └────────┼────────┘
                       │
                   Telemetry
                   (Metrics)
```

---

## 1. Concurrency & Locking Strategy

### Cache Engine Locking
- **Exclusive Locking on `Get`**: A `Get` in this cache system is **not** a read-only operation:
  - In **LRU**: `Get` moves the entry to the MRU position.
  - In **LFU**: `Get` increments the entry's access frequency and promotes it to a higher bucket.
  - In **2Q**: `Get` can promote an entry from queue A1 to queue Am.
  - In **TTL**: `Get` detects expired items and lazily purges them.
  Therefore, `Cache.Get` acquires an exclusive lock (`c.mu.Lock()`).
- **Shared Locking on Introspection**: Read-only queries such as `Size()`, `Capacity()`, `Policy()`, and queue inspector helpers acquire a shared read lock (`c.mu.RLock()`).
- **Minimal Lock Scopes**: Lock scopes are restricted strictly to in-memory pointer manipulation and map lookups. Locks are released immediately before any subsequent processing.

### Router & Network I/O Locking
- **Locking Completely Decoupled from Network I/O**: The router **never** holds a lock during outbound network communication:
  1. `Route(key)` queries `HashRing.GetNode(key)` under a brief read lock (`r.ring.mu.RLock()`) to find the target node.
  2. The read lock is released immediately.
  3. Outbound proxy HTTP requests (`r.client.Do(req)`) run completely outside of any lock.
- **Context Propagation**: The router forwards the incoming request context (`req.Context()`) to the backend request (`http.NewRequestWithContext(req.Context(), ...)`). If a client disconnects or aborts, the backend request is automatically canceled.
- **Connection Reuse**: The router uses a shared, long-lived `http.Client` with connection pooling and configurable request timeouts (`DefaultConfig.ClientTimeout = 5s`).

---

## 2. Metrics & Observability

### Why Metrics Exist
> **Correctness** tells us whether the cache works.  
> **Metrics** tell us how the cache behaves.

Metrics allow operators to monitor hit ratios, capacity pressure, latency trends, and node health without changing cache semantics or adding runtime performance penalties.

### Lock-Free Atomic Metrics
Metrics are implemented in `internal/metrics` using `sync/atomic` 64-bit unsigned integer counters (`atomic.Uint64`). Operations use fast hardware atomic instructions (`Add(1)`, `Load()`), avoiding mutex contention even under high concurrent loads.

### Metric Ownership Separation

```text
Router metrics ≠ Node metrics ≠ Cache engine metrics
```

Because requests pass through multiple layers, metric ownership is strictly delineated:

| Metric | Measured At | Layer | Description |
| :--- | :---: | :---: | :--- |
| **`hits`** | Cache Engine | Cache | Key found and valid (not expired) |
| **`misses`** | Cache Engine | Cache | Key not found OR found but expired |
| **`hit_rate`** | Cache Engine | Cache | Calculated: $\frac{\text{hits}}{\text{hits} + \text{misses}}$ (0.0 if total is 0) |
| **`sets`** | Cache Engine | Cache | Number of `Set` and `SetWithTTL` operations |
| **`deletes`** | Cache Engine | Cache | Number of entries removed via `Delete` |
| **`evictions`** | Cache Engine | Cache | Number of valid entries evicted due to capacity limits |
| **`expired`** | Cache Engine | Cache | Number of expired entries purged upon access |
| **`requests`** | Node & Router | HTTP | Total HTTP requests handled at that specific layer |
| **`errors_4xx`** | Node & Router | HTTP | Total client-side errors (400 Bad Request, 404 Not Found, 405 Method Not Allowed) |
| **`errors_5xx`** | Node & Router | HTTP | Total server-side errors (500 Internal Error, 502 Bad Gateway) |
| **`avg_latency_ms`** | Node & Router | HTTP | Average request duration: $\frac{\text{totalLatencyNs}}{\text{requests} \times 10^6}$ |

---

## 3. Observability Endpoints

### Cache Server Node (`GET /metrics`)
Returns JSON containing both node-level HTTP metrics and underlying cache engine statistics:

```json
{
  "hits": 1200,
  "misses": 300,
  "hit_rate": 0.8,
  "sets": 1500,
  "deletes": 100,
  "evictions": 50,
  "expired": 25,
  "requests": 1600,
  "errors_4xx": 10,
  "errors_5xx": 2,
  "avg_latency_ms": 1.42,
  "policy": "lru",
  "capacity": 1000,
  "size": 742
}
```

*(Note: Cache servers also preserve `GET /cache` for simple status: `{"size": 742, "capacity": 1000, "policy": "lru"}`).*

### Router (`GET /metrics`)
Returns JSON containing router-specific reverse proxy metrics:

```json
{
  "requests": 5000,
  "successes": 4965,
  "errors_4xx": 20,
  "errors_5xx": 15,
  "avg_latency_ms": 2.1
}
```

---

## 4. Consistent Hashing & Ring Mechanics

### Hash Ring Structure
The 32-bit FNV-1a hash space ($[0, 2^{32}-1]$) forms a continuous circular ring:

```text
                 Node A
                   |
             +-----+-----+
            /             \
        Node C             Node B
            \             /
             +-----------+
                   |
                hash ring
```

```text
key ──► hash(key) ──► ring position ──► clockwise binary search ──► selected physical node
```

- **Virtual Nodes**: Each physical node registers 100 virtual positions (configurable via `--replicas`) distributed evenly along the ring to eliminate hot spots.
- **Clockwise Binary Search**: Key lookup takes $O(\log R)$ time via `sort.Search` (~121 ns per lookup in benchmarks).
- **Minimal Key Redistribution**: Expanding from 3 to 4 nodes moves only ~15–25% of keys, compared to ~75% under modulo hashing.
- **Order-Independent**: Node IDs are sorted alphabetically and deterministic linear probing resolves collisions, guaranteeing identical routing regardless of the node configuration order.

---

## 5. Clean Layer Separation

```text
                Client
                   │
                   ▼
                 Router
                   │
                   ▼
          ┌────────────────┐
          │   Hash Ring    │
          └────────────────┘
                   │
             selected Node
                   │
                   ▼
          Cache HTTP Server
                   │
                   ▼
            Cache Engine
           /      |       \
         LRU     LFU      2Q
                   │
                  TTL
```

- **Router**: HTTP proxying, connection pooling, context propagation, client timeouts, routing delegation.
- **HashRing**: Hash calculation, virtual node replica management, collision resolution, key-to-node clockwise mapping, thread safety.
- **Cache Server**: Local HTTP API handlers (`PUT`, `GET`, `DELETE`, `/health`, `/node`, `/cache`, `/metrics`), status recording.
- **Cache Engine**: Eviction policies (LRU, LFU, 2Q), lazy TTL expiration, exclusive concurrency safety, cache telemetry.

---

## 6. Benchmarking & Performance Evaluation Framework

Phase 11 introduces a multi-tiered benchmarking architecture designed to evaluate the cache engine, hashing ring, and transport stack with rigorous scientific controls:

### Layered Benchmark Isolation
To determine exactly where latency originates, the benchmarking system isolates components into 4 distinct layers:

```text
Layer 1: Raw Cache Engine (In-Memory Microbenchmarks & Synthetic Workloads)
           ├── Microbenchmarks: GET (existing/missing), SET (new/existing), DELETE
           ├── TTL benchmarks: SetWithTTL, GetBeforeExpiry, GetAfterExpiry
           └── Synthetic Workloads: Uniform, Zipfian, Sequential, Scan/Pollution, Hot Set
                               │
Layer 2: Hash Ring Routing Decision (HashRing.GetNode)
           └── Binary search latency across 3–50 nodes and 50–300 virtual replicas
                               │
Layer 3: Cache Node HTTP Server (Direct Node Endpoints)
           └── Single-hop HTTP serialization, context creation, and handler latency
                               │
Layer 4: Distributed Router HTTP Proxy (End-to-End Client Routing)
           └── Router HTTP proxying, HashRing selection, and backend TCP communication
```

### Deterministic Workload Generators
All synthetic workloads enforce determinism (`seed = 42`) to guarantee cross-policy fairness:
1. **Uniform Random**: Flat key distribution testing non-local memory patterns.
2. **Zipfian / Hot-Key**: Power-law distribution ($s=1.25, v=1.0$) testing frequency exploitation.
3. **Sequential Scan**: Streaming sequential accesses testing zero-locality cache behavior.
4. **Scan / Pollution**: Hot-set establishment followed by a 5,000-key one-time scan to test 2Q pollution resistance.
5. **Repeated Hot Set**: Bounded 50-key working set tested against under-sized ($C=20$) and fitting ($C=100$) caches.

### Empirical Topology Analysis
Topological scaling tests quantify data migration:
- **Consistent Hashing**: $\approx \frac{K}{N+1}$ keys moved on node addition (**13.37%** in empirical tests), with $0\%$ disruption to unaffected nodes.
- **Modulo Hashing**: $\approx \frac{N}{N+1}$ keys moved on node addition (**75.07%** in empirical tests), causing widespread cache stampedes.

For detailed numbers, graphs, and hardware environment details, refer to [`docs/benchmark-results.md`](benchmark-results.md).

