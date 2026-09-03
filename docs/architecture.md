# Architecture

## Current Architecture (Phase 6 — HTTP API for Single-Node Cache Server)

At present, the system exposes the standalone, thread-safe in-memory key-value cache through a clean HTTP REST API layer built on Go's standard library `net/http`:

```text
                         HTTP Client
                              |
                              v
                     ┌─────────────────┐
                     │  HTTP Server    │
                     │                 │
                     │ GET / SET /     │
                     │ DELETE / health │
                     └────────┬────────┘
                              |
                              v
                         Cache Engine
                              |
                 ┌────────────┼────────────┐
                 ↓            ↓            ↓
                LRU          LFU           2Q
                              |
                             TTL
```

> **Important**: The current system is still a single cache node. Distribution, consistent hashing, and clustering will be introduced in Phase 7+.

### Components

* **HTTP Server (`internal/server`)**:
  - Transport layer providing JSON REST endpoints (`PUT /cache/{key}`, `GET /cache/{key}`, `DELETE /cache/{key}`, `GET /cache`, `GET /health`).
  - Strict decoupling: does not contain any eviction or cache logic, delegating strictly to the `Cache` interface.
  - Handles URL unescaping, malformed JSON detection, and standard HTTP error codes (200, 400, 404, 405, 500).
  - Supports configurable timeouts and graceful shutdown on `SIGINT` / `SIGTERM`.
* **Cache Coordinator (`internal/cache`)**:
  - Exposes the unified public API (`Set`, `SetWithTTL`, `Get`, `Delete`, `Size`, `Capacity`, `Policy`) and guards all operations with `sync.RWMutex`.
* **TTL (Time To Live)**:
  - Backed by an absolute timestamp `expiresAt time.Time` attached to each cache entry across all storage engines.
  - Implements **lazy expiration**: when an entry is accessed via `Get` or `Delete`, expiration is checked in $O(1)$ time. If expired, it is removed immediately.
  - Reversion to persistent entries occurs seamlessly upon a normal `Set`.
* **Eviction Policies**:
  - **LRU (Least Recently Used)**: Promotes accessed entries to MRU head; evicts from LRU tail in $O(1)$ time.
  - **LFU (Least Frequently Used)**: Tracks frequencies using frequency buckets and `minFreq` with deterministic LRU tie-breaking.
  - **2Q (Two-Queue)**: Dual-queue architecture separating recent admissions (A1 FIFO) from frequent entries (Am LRU).

---

## Planned Architecture (Future Phases)

The distributed system will eventually consist of a routing layer distributing requests across multiple cache nodes:

```text
                    Client
                       |
                       v
                  HTTP Router
                       |
            +----------+----------+
            |          |          |
            v          v          v
         Node 1      Node 2      Node 3
          Cache       Cache       Cache
            |          |          |
            +----------+----------+
                 In Memory
```

### Planned Components

* **Router**: Routes client requests to the appropriate cache node using consistent hashing.
* **Cache Server**: Node-level network service handling requests and serving the local cache engine.
* **Node Management**: Membership tracking and node health monitoring.
* **Consistent Hashing**: Hash ring to evenly distribute keys across active nodes with minimal rehashing.
* **Metrics**: Cache hit/miss rates, memory usage, and operational latency tracking.
* **Future Replication / Failover**: Redundant data replication and automatic failover handling.
