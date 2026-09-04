# Universal Distributed Cache System

A lightweight distributed in-memory caching system built in Go.

## Goals

* In-memory key-value caching
* Configurable eviction policies
* TTL-based expiration
* Multiple cache nodes
* Consistent hashing for key distribution
* Concurrent request handling
* Cache metrics and benchmarking

## Planned Eviction Policies

* LRU (Implemented)
* LFU (Implemented)
* 2Q (Implemented)

## Status

```text
Phase 1 — Basic In-Memory Cache            ✓
Phase 2 — LRU                             ✓
Phase 3 — LFU                             ✓
Phase 4 — 2Q                              ✓
Phase 5 — TTL                             ✓
Phase 6 — HTTP API                        ✓
Phase 7 — Multiple Cache Nodes            ✓
Phase 8 — Router                          ✓
Phase 9 — Consistent Hashing              ✓
Phase 10 — Concurrency & Metrics          ✓
Phase 11 — Benchmarking & Performance     ✓
```

---

## Distributed Cluster Setup

Run multiple independent cache nodes alongside a central routing proxy backed by a consistent hash ring:

```bash
# 1. Start Cache Nodes
go run ./cmd/cache-server --id node-1 --port 8001 --policy lru
go run ./cmd/cache-server --id node-2 --port 8002 --policy lfu
go run ./cmd/cache-server --id node-3 --port 8003 --policy 2q

# 2. Start Distributed Router (with Consistent Hashing)
go run ./cmd/router \
  --port 9000 \
  --nodes node-1=http://localhost:8001,node-2=http://localhost:8002,node-3=http://localhost:8003 \
  --replicas 100
```

Clients interact exclusively with the router at `http://localhost:9000`.

---

## Distributed Request Flow

```text
Client
  │
  ▼
Router (:9000)
  │
  ▼
Consistent Hash Ring (Virtual Nodes)
  │
  ▼
Selected Cache Node (:8001, :8002, or :8003)
  │
  ▼
Cache Engine (LRU / LFU / 2Q + TTL)
  │
  ▼
Atomic Metrics & Telemetry
```

### Consistent Hashing Ring Mechanism

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

- **Virtual Nodes**: Each physical node registers 100 virtual positions (configurable via `--replicas`) distributed evenly along the ring to eliminate load skew.
- **Clockwise Binary Search**: Key lookup takes $O(\log R)$ time via `sort.Search` (~121 ns per lookup in benchmarks).
- **Minimal Key Redistribution**: Expanding from 3 to 4 nodes moves only ~15–25% of keys, compared to ~75% under modulo hashing.
- **Order-Independent**: Node IDs are sorted and deterministic linear probing is used for collision handling, guaranteeing identical routing regardless of the node configuration order.

---

## Router HTTP REST API

The router exposes a unified REST interface identical to individual cache nodes, plus node inspection and router telemetry endpoints:

### Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| **`GET`** | `/health` | Router health check returning `{"status": "ok"}` |
| **`GET`** | `/nodes` | List of all registered backend cache nodes and replica count |
| **`GET`** | `/metrics` | Router-level HTTP metrics (requests, 4xx/5xx errors, latency) |
| **`PUT`** | `/cache/{key}` | Deterministically route and store entry on owning cache node |
| **`GET`** | `/cache/{key}` | Deterministically route and retrieve entry from owning cache node |
| **`DELETE`** | `/cache/{key}` | Deterministically route and remove entry from owning cache node |

### cURL Examples via Router

#### 1. Router Health
```bash
curl http://localhost:9000/health
# Response: {"status":"ok"}
```

#### 2. Configured Nodes & Replicas
```bash
curl http://localhost:9000/nodes
# Response: {"nodes":[{"id":"node-1","address":"http://localhost:8001"},{"id":"node-2","address":"http://localhost:8002"},{"id":"node-3","address":"http://localhost:8003"}],"replicas":100}
```

#### 3. Router Telemetry
```bash
curl http://localhost:9000/metrics
# Response: {"requests":5000,"successes":4965,"errors_4xx":20,"errors_5xx":15,"avg_latency_ms":0.42}
```

#### 4. Store Key via Router
```bash
curl -X PUT http://localhost:9000/cache/user:123 \
  -H "Content-Type: application/json" \
  -d '{"value":"Priyansh","ttl":"60s"}'
# Response: {"message":"cache entry stored"}
```

#### 5. Retrieve Key via Router
```bash
curl http://localhost:9000/cache/user:123
# Response: {"key":"user:123","value":"Priyansh"}
```

#### 6. Delete Key via Router
```bash
curl -X DELETE http://localhost:9000/cache/user:123
# Response: {"message":"cache entry deleted"}
```

---

## Node-Level Observability (`GET /metrics`)

Each individual cache node exposes its own independent telemetry:

```bash
curl http://localhost:8001/metrics
```

Example JSON Response:
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

### Hit Rate Calculation
$$\text{Hit Rate} = \frac{\text{hits}}{\text{hits} + \text{misses}}$$
*(Safely returns `0.0` if total queries are 0).*

---

## TTL / Key Expiration Overview

TTL (Time-To-Live) controls how long an entry remains valid. Eviction policies (LRU, LFU, 2Q) control which entry is removed when the cache reaches its capacity. These are completely independent mechanisms.

```text
TTL controls how long an entry remains valid.

Eviction policies control which entry is removed when
the cache reaches its capacity.

These are independent mechanisms.
```

### Expiration Semantics

* **`Set(key, value)`**: Stores a key with no expiration (`expiresAt` is zero). If the key previously had a TTL, the expiration is removed, converting it into a persistent cache entry.
* **`SetWithTTL(key, value, ttl)`**: Stores a key with an absolute expiration calculated as `time.Now().Add(ttl)`.
* **Updating TTL**: Calling `SetWithTTL` on an existing key updates its value and resets its expiration to `now + new_ttl`.
* **Lazy Expiration**: Expired entries are detected and cleaned up lazily when an operation encounters them (`GET`, `DELETE`). Expired reads are counted as misses and increment the `expired` metric counter.
* **Edge-case Semantics (`TTL <= 0`)**:
  - `TTL = 0` or negative `TTL < 0`: Treated as immediately expired. Any existing entry with that key is removed immediately, and no entry is stored in cache.

---

## Eviction Policies Overview

### LRU (Least Recently Used)
Removes the item that has not been used recently. Every successful `GET` or `SET` moves the item to the MRU (Most Recently Used) position.

### LFU (Least Frequently Used)
Removes the item that has been used the fewest times across its lifetime in cache. Every access (`GET` or `SET` update) increments the item's access counter.

### LFU Tie-Breaking
When multiple entries share the same lowest access frequency, eviction tie-breaking is resolved using **LRU among equally frequent items**—the least recently accessed item within that frequency bucket is evicted first.

### 2Q (Two-Queue)
2Q uses two conceptual queues to eliminate cache pollution caused by one-time or sequential scans:
- **A1**: Recently / newly seen entries (FIFO queue).
- **Am**: Frequently reused entries (LRU queue).

New entries initially enter **A1**. If an entry in A1 is accessed again (via `GET` or updating `SET`), it graduates and is promoted to **Am**. Entries accessed only once are evicted from A1 without polluting Am.

**Default Capacity Ratio**: By default, 2Q allocates **25%** of total capacity to **A1** (FIFO) and **75%** to **Am** (LRU), configurable via `New2QWithCapacities(total, a1Cap, amCap)`.

---

## Major Data Structures

### Cache Coordinator & Locking
- **Locking Strategy**: `Get` operations acquire an exclusive write lock (`Lock()`) because `Get` mutates LRU recency, LFU frequency, 2Q queue promotions, or performs lazy TTL purges.
- **Lock Scopes**: In-memory pointer/map operations only. No locks held across network boundaries.

### Lock-Free Atomic Metrics
- **Atomic Counters (`internal/metrics`)**: Implemented with `sync/atomic` 64-bit unsigned integers (`atomic.Uint64`), eliminating mutex contention across concurrent requests.

### Consistent Hash Ring Architecture
- **Sorted Positions (`[]uint32`)**: $O(\log R)$ binary search (`sort.Search`) for clockwise key lookup.
- **Ring Map (`map[uint32]Node`)**: $O(1)$ mapping from ring position to physical node.
- **Virtual Nodes**: Configurable replicas per node (default 100) preventing hotspots.
- **Thread Safety**: Protected by `sync.RWMutex` for concurrent requests.

---

## Time Complexity & Performance

| Operation | Expected Complexity | Benchmarked Latency | Description |
| :--- | :---: | :---: | :--- |
| **LRU GET** | $O(1)$ average | **~95 ns/op** | Map lookup + recency promotion + lock-free atomic metric increment |
| **LFU GET** | $O(1)$ average | **~105 ns/op** | Map lookup + bucket frequency promotion |
| **2Q GET** | $O(1)$ average | **~86 ns/op** | Map lookup + A1 $\to$ Am queue promotion |
| **LRU SET** | $O(1)$ average | **~184 ns/op** | Map insert + LRU node push |
| **LFU SET** | $O(1)$ average | **~198 ns/op** | Map insert + bucket 1 push |
| **2Q SET** | $O(1)$ average | **~173 ns/op** | Map insert + A1 FIFO push |
| **Key Routing** | $O(\log R)$ | **~123 ns/op** | FNV-1a hash calculation + binary search clockwise on $R$ virtual node positions |

---

## Phase Summary

### Phase 1 — Basic In-Memory Cache
- Core key-value store (`SET`, `GET`, `DELETE`, `Size`)
- Thread safety via internal `sync.RWMutex`
- Comprehensive unit tests

### Phase 2 — LRU Eviction
- Configurable maximum capacity and error validation
- Doubly linked list for $O(1)$ recency promotion and eviction
- Concurrency protection across readers and writers

### Phase 3 — LFU Eviction
- Frequency tracking (initial frequency 1, increment on GET and SET updates)
- Frequency buckets with $O(1)$ promotions
- Deterministic LRU tie-breaking within identical frequency tiers
- Dynamic `minFreq` tracking for $O(1)$ evictions

### Phase 4 — 2Q Eviction
- Dual-queue architecture separating recent admissions (A1 FIFO) from frequent entries (Am LRU)
- Admission and promotion mechanics preventing single-access scan pollution
- Configurable capacity partitioning (default 25% A1 / 75% Am)

### Phase 5 — TTL / Key Expiration
- Optional `time.Duration` expiration on entries via `SetWithTTL`
- Reversion to persistent entries on normal `Set`
- Lazy expiration upon access preventing background cleanup goroutine overhead
- Seamless compatibility with LRU, LFU, and 2Q eviction mechanics

### Phase 6 — HTTP API
- RESTful HTTP transport layer using Go standard library `net/http`
- Decoupled server architecture interacting via `Cache` interface
- JSON request and response payloads with validation and error formatting
- CLI flags (`--port`, `--capacity`, `--policy`) and graceful shutdown on termination signals

### Phase 7 — Multiple Independent Cache Nodes
- Node entity abstraction (`internal/node`) managing identity, config validation, and lifecycle
- Multi-process execution on distinct ports (e.g. `:8001`, `:8002`, `:8003`)
- Strict memory isolation guaranteeing zero accidental state sharing between nodes
- Dedicated `/node` endpoint and `node_id` reporting on `/health`

### Phase 8 — Distributed Router with Hash-Based Routing
- Central HTTP reverse proxy router dispatching client requests
- Deterministic FNV-1a hashing with modulo node index selection: $\text{hash}(\text{key}) \pmod N$
- Node registry with strict URL and ID validation
- End-to-end transparent payload proxying with 502 bad gateway isolation on unreachable nodes

### Phase 9 — Consistent Hashing
- Dedicated, thread-safe `HashRing` component replacing modulo hashing
- Virtual nodes (default 100 replicas) distributing load uniformly across the 32-bit hash space
- Clockwise key routing via binary search (`sort.Search`) in $O(\log R)$ time (~121 ns/op)
- Minimal key churn upon topology changes: only ~15–25% of keys remapped on node addition vs ~75% in modulo hashing
- Deterministic virtual node naming with linear probing collision resolution ensuring input-order independence
- Router HTTP handlers decoupled from hashing implementation details

### Phase 10 — Concurrency, Metrics & Observability
- Concurrency hardening across all cache engines with exclusive locking on state-mutating `Get` operations
- Lock-free atomic metrics package (`internal/metrics`) tracking hits, misses, hit rate, sets, deletes, evictions, expired entries
- Clean separation of metric ownership between Router, Node, and Cache Engine
- HTTP request telemetry tracking total requests, 2xx/4xx/5xx status codes, and average latency
- New `/metrics` JSON endpoints for both cache server nodes and the distributed router
- Context propagation (`http.NewRequestWithContext`) canceling outbound requests upon client disconnect
- Multi-worker concurrent stress tests and Go benchmark suite in preparation for Phase 11

### Phase 11 — Benchmarking & Performance Evaluation
- Comprehensive empirical evaluation of **LRU vs LFU vs 2Q** across operation types, capacities (100, 1,000, 10,000), and synthetic workloads
- 5 deterministic synthetic workloads: Uniform Random, Zipfian / Hot-Key, Sequential Access, Scan / Pollution, and Repeated Hot Set
- Multi-threaded parallel benchmarks (`b.RunParallel`) measuring lock contention and concurrent throughput across 12 worker goroutines
- Empirical topology change experiments: consistent hashing remapped only **13.37%** of keys on node addition vs **75.07%** under modulo hashing
- Full empirical report, layered latency breakdown, charts, and architectural analysis documented in [`docs/benchmark-results.md`](docs/benchmark-results.md)

---

## Benchmarking & Performance Evaluation

The system includes a reproducible benchmarking suite built on Go's standard `testing.B` framework.

### 1. Running Benchmarks

```bash
# Run operation microbenchmarks with memory allocations (Layer 1)
go test -bench="Benchmark(Get|Set|Delete|TTL|Parallel)" -benchmem ./internal/cache

# Run the complete empirical workload evaluation table (LRU vs LFU vs 2Q)
$env:RUN_WORKLOAD_REPORT="1"; go test -v -run="TestWorkloadEvaluationReport" ./internal/cache

# Run consistent hashing topology change experiments (3 -> 4 nodes, node removal)
go test -v -run="TestConsistentHashNode" ./internal/router

# Run consistent hash ring lookup benchmark (Layer 2)
go test -bench="BenchmarkConsistentHashRingLookup" -benchmem ./internal/router

# Run direct HTTP node benchmarks (Layer 3)
go test -bench="BenchmarkHTTPServer" -benchmem ./internal/server

# Run distributed router proxy benchmarks (Layer 4)
go test -bench="BenchmarkRouterProxy" -benchmem ./internal/router
```

### 2. Key Empirical Findings

- **Best Raw Read Latency**: **2Q** achieved the lowest GET latency (**22.10 – 27.60 ns/op**) with zero memory allocations.
- **Best Hit Rate under Hot-Key / Skewed Traffic**: **LFU** achieved the highest hit rate (**77.38%** at capacity 100 and **90.66%** at capacity 1,000) under Zipfian distributions.
- **Best Concurrent Read Scalability**: Under 12 parallel reader goroutines, **2Q** achieved **73.19 ns/op** (1.45x faster than LRU's 106.2 ns/op).
- **Cluster Resiliency**: Adding a 4th node remapped only **13.37%** of keys under Consistent Hashing compared to **75.07%** under Modulo Hashing.
- **Layered Overhead**: Core cache engine lookups require **~25 ns**, while network and HTTP serialization account for **>99.9%** of total request latency (~90 µs direct node, ~207 µs router proxy).

For full benchmark tables, visual comparison charts, and interview-level analyses, see [docs/benchmark-results.md](docs/benchmark-results.md).

