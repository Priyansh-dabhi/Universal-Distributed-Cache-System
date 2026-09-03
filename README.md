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
Phase 1 — Basic In-Memory Cache    ✓
Phase 2 — LRU                     ✓
Phase 3 — LFU                     ✓
Phase 4 — 2Q                      ✓
Phase 5 — TTL                     ✓
Phase 6 — HTTP API                ✓
Phase 7 — Multiple Cache Nodes    ✓
Phase 8 — Router                  ✓
Phase 9 — Consistent Hashing      ✓
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
key
 ↓
hash(key)
 ↓
ring position
 ↓
clockwise binary search
 ↓
physical node
```

- **Virtual Nodes**: Each physical node registers 100 virtual positions (configurable via `--replicas`) distributed evenly along the ring to eliminate load skew.
- **Clockwise Binary Search**: Key lookup takes $O(\log R)$ time via `sort.Search` (~121 ns per lookup in benchmarks).
- **Minimal Key Redistribution**: Expanding from 3 to 4 nodes moves only ~15–25% of keys, compared to ~75% under modulo hashing.
- **Order-Independent**: Node IDs are sorted and deterministic linear probing is used for collision handling, guaranteeing identical routing regardless of the node configuration order.

---

## Router HTTP REST API

The router exposes a unified REST interface identical to individual cache nodes, plus node inspection endpoints:

### Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| **`GET`** | `/health` | Router health check returning `{"status": "ok"}` |
| **`GET`** | `/nodes` | List of all registered backend cache nodes and replica count |
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

#### 3. Store Key via Router
```bash
curl -X PUT http://localhost:9000/cache/user:123 \
  -H "Content-Type: application/json" \
  -d '{"value":"Priyansh","ttl":"60s"}'
# Response: {"message":"cache entry stored"}
```

#### 4. Retrieve Key via Router
```bash
curl http://localhost:9000/cache/user:123
# Response: {"key":"user:123","value":"Priyansh"}
```

#### 5. Delete Key via Router
```bash
curl -X DELETE http://localhost:9000/cache/user:123
# Response: {"message":"cache entry deleted"}
```

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
* **Lazy Expiration**: Expired entries are detected and cleaned up lazily when an operation encounters them (`GET`, `DELETE`). This avoids background cleanup goroutines and extra concurrency overhead while guaranteeing expired entries are never returned.
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

### Cache Coordinator & TTL
- **`expiresAt time.Time`**: Attached to each cache node across all eviction policies.
- **Lazy Eviction on Access**: During `Get`, if `!expiresAt.IsZero() && now >= expiresAt`, the entry is unlinked and deleted in $O(1)$ time.

### LRU Architecture
- **HashMap (`map[string]*lruNode`)**: $O(1)$ key lookup.
- **Doubly Linked List**: Maintains recency order between sentinel `head` (MRU) and `tail` (LRU).

### LFU Architecture
- **HashMap (`map[string]*lfuNode`)**: $O(1)$ key-to-node lookup.
- **Frequency Buckets (`map[int]*lfuList`)**: Maps each access frequency to an ordered doubly linked list of items.
  - New/updated items are prepended at the head (MRU within that frequency).
  - The tail (`tail.prev`) holds the least recently accessed item with that frequency (enabling $O(1)$ LRU tie-breaking).
- **`minFreq` Counter**: Tracks the minimum non-empty frequency bucket for $O(1)$ eviction candidate selection without scanning.

### 2Q Architecture
- **HashMap (`map[string]*twoQNode`)**: $O(1)$ key-to-node index mapping.
- **Queue A1 (`twoQList`)**: Doubly linked list managed as a FIFO queue for recent admissions.
- **Queue Am (`twoQList`)**: Doubly linked list managed as an LRU queue for promoted entries.

### Consistent Hash Ring Architecture
- **Sorted Positions (`[]uint32`)**: $O(\log R)$ binary search (`sort.Search`) for clockwise key lookup.
- **Ring Map (`map[uint32]Node`)**: $O(1)$ mapping from ring position to physical node.
- **Virtual Nodes**: Configurable replicas per node (default 100) preventing hotspots.
- **Thread Safety**: Protected by `sync.RWMutex` for concurrent requests.

---

## Time Complexity

| Operation | Expected Complexity | Description |
| :--- | :---: | :--- |
| **GET** | $O(1)$ average | Map lookup + $O(1)$ expiration check + recency/frequency/promotion update |
| **SET** | $O(1)$ average | Map lookup/insert + node insertion + optional eviction |
| **SET with TTL** | $O(1)$ average | Same as SET with timestamp calculation |
| **DELETE** | $O(1)$ average | Map removal + unlinking from linked list / frequency bucket / 2Q queue |
| **Expiration Check** | $O(1)$ | Timestamp comparison (`now >= expiresAt`) |
| **Promotion A1 → Am** | $O(1)$ | Unlink from A1 and prepend to Am head |
| **A1 eviction** | $O(1)$ | Unlink tail of A1 FIFO queue |
| **Am eviction** | $O(1)$ | Unlink tail of Am LRU queue |
| **Size** | $O(1)$ | Querying map length (expired entries cleaned up on access) |
| **Key Routing** | $O(\log R)$ | FNV-1a hash calculation + binary search clockwise on $R$ virtual node positions (~121 ns) |
| **Node Addition** | $O(R \log R)$ | Registering virtual nodes, resolving collisions, and sorting ring |
| **Node Removal** | $O(R \log R)$ | Removing virtual nodes and updating ring |

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
