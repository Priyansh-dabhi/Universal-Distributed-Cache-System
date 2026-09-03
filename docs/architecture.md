# Architecture

## Current Architecture (Phase 3 — In-Memory Cache with Configurable LRU & LFU)

At present, the system implements a standalone, thread-safe in-memory key-value cache with pluggable eviction policies (LRU and LFU):

```text
                    Cache
                      |
              Eviction Policy
                 /         \
                /           \
              LRU           LFU
                              |
                     Frequency Buckets
```

### Components

* **Cache Coordinator**: Wraps operations with `sync.RWMutex`, exposes the unified cache API (`Set`, `Get`, `Delete`, `Size`), and delegates storage/eviction to the configured policy.
* **LRU (Least Recently Used)**:
  - Backed by `map[string]*lruNode` and a doubly linked list.
  - Promotes accessed entries to the head (MRU); evicts from the tail (LRU) in $O(1)$ time.
* **LFU (Least Frequently Used)**:
  - Backed by `map[string]*lfuNode` and `freqBuckets map[int]*lfuList`.
  - Tracks `minFreq` for instantaneous $O(1)$ identification of the lowest frequency bucket.
  - Resolves ties via LRU ordering within each frequency bucket's doubly linked list.

> **Note**: 2Q eviction, expiration (TTL), HTTP routing, consistent hashing, and multi-node clustering are planned for subsequent phases and are **not yet implemented**.

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
* **Cache Engine**: The core storage coordinator managing entries and triggering eviction.
* **Eviction Policies**: Configurable eviction strategies (LRU, LFU, 2Q) per cache server.
* **Node Management**: Membership tracking and node health monitoring.
* **Consistent Hashing**: Hash ring to evenly distribute keys across active nodes with minimal rehashing.
* **TTL / Expiration**: Background or on-access expiration for key lifetimes.
* **Metrics**: Cache hit/miss rates, memory usage, and operational latency tracking.
* **Concurrency**: Thread-safe operations across readers and writers.
* **Future Replication / Failover**: Redundant data replication and automatic failover handling.

For the eviction layer, the design will eventually support:

```text
Cache Engine
    |
    +-- LRU (Implemented)
    +-- LFU (Implemented)
    +-- 2Q  (Planned)
```
