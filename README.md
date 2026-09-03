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

## Planned Architecture

Client → Router → Cache Nodes

## Status

```text
Phase 1 — Basic In-Memory Cache    ✓
Phase 2 — LRU                     ✓
Phase 3 — LFU                     ✓
Phase 4 — 2Q                      ✓
Phase 5 — TTL                     ✓
Phase 6 — HTTP API                ✓
Phase 7 — Multiple Cache Nodes    ✓
Phase 8 — Router                  Planned
```

---

## Running Multiple Nodes

Multiple independent cache nodes can be executed simultaneously as distinct processes with isolated in-memory stores and independent eviction configurations:

```bash
go run ./cmd/cache-server --id node-1 --port 8001 --policy lru
go run ./cmd/cache-server --id node-2 --port 8002 --policy lfu
go run ./cmd/cache-server --id node-3 --port 8003 --policy 2q
```

> **Important**: Each node currently maintains an independent in-memory cache. Requests are not automatically distributed between nodes yet.

---

## HTTP REST API

The cache server exposes a REST API powered by Go's standard library `net/http`.

### Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| **`GET`** | `/health` | Health check endpoint returning status and `node_id` |
| **`GET`** | `/node` | Static node configuration (`id`, `host`, `port`, `capacity`, `policy`) |
| **`GET`** | `/cache` | Current cache metrics (`size`, `capacity`, `policy`) |
| **`PUT`** | `/cache/{key}` | Store a key-value entry (supports optional `ttl` duration) |
| **`GET`** | `/cache/{key}` | Retrieve an entry by key (returns 404 on miss or expiration) |
| **`DELETE`** | `/cache/{key}` | Remove an entry by key (returns 404 if missing or expired) |

### cURL Examples

#### 1. Health Check
```bash
curl http://localhost:8001/health
# Response: {"status":"ok","node_id":"node-1"}
```

#### 2. Node Information
```bash
curl http://localhost:8001/node
# Response: {"id":"node-1","host":"127.0.0.1","port":8001,"capacity":100,"policy":"lru"}
```

#### 3. Set Entry (without TTL)
```bash
curl -X PUT http://localhost:8001/cache/user:123 \
  -H "Content-Type: application/json" \
  -d '{"value":"Priyansh"}'
# Response: {"message":"cache entry stored"}
```

#### 4. Set Entry with TTL
```bash
curl -X PUT http://localhost:8001/cache/session:abc \
  -H "Content-Type: application/json" \
  -d '{"value":"active","ttl":"60s"}'
# Response: {"message":"cache entry stored"}
```

#### 5. Get Entry
```bash
curl http://localhost:8001/cache/user:123
# Response: {"key":"user:123","value":"Priyansh"}
```

#### 6. Cache Information
```bash
curl http://localhost:8001/cache
# Response: {"size":2,"capacity":100,"policy":"lru"}
```

#### 7. Delete Entry
```bash
curl -X DELETE http://localhost:8001/cache/user:123
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
