# Architecture

## Current Architecture (Phase 5 — In-Memory Cache with Pluggable Eviction and TTL)

At present, the system implements a standalone, thread-safe in-memory key-value cache supporting three configurable eviction policies (LRU, LFU, and 2Q) alongside an independent TTL expiration mechanism:

```text
                         Cache
                           |
              ┌────────────┴────────────┐
              ↓                         ↓
       Eviction Policy                TTL
        /     |     \                   |
       LRU    LFU    2Q            expiresAt
                     |
            ┌────────┴────────┐
            ↓                 ↓
           A1                 Am
          FIFO               LRU
```

### Components

* **Cache Coordinator**: Exposes the unified public API (`Set`, `SetWithTTL`, `Get`, `Delete`, `Size`, `Capacity`, `Policy`) and guards all operations with `sync.RWMutex`.
* **TTL (Time To Live)**:
  - Backed by an absolute timestamp `expiresAt time.Time` attached to each cache entry across all storage engines.
  - Implements **lazy expiration**: when an entry is accessed via `Get` or `Delete`, expiration is checked in $O(1)$ time. If expired, it is removed immediately.
  - Reversion to persistent entries occurs seamlessly upon a normal `Set`.
* **LRU (Least Recently Used)**:
  - Backed by `map[string]*lruNode` and a doubly linked list.
  - Promotes accessed entries to the head (MRU); evicts from the tail (LRU) in $O(1)$ time.
* **LFU (Least Frequently Used)**:
  - Backed by `map[string]*lfuNode` and `freqBuckets map[int]*lfuList`.
  - Tracks `minFreq` for instantaneous $O(1)$ identification of the lowest-frequency bucket.
  - Resolves ties via LRU ordering within each frequency bucket's doubly linked list.
* **2Q (Two-Queue)**:
  - Backed by `map[string]*twoQNode` and two distinct doubly linked lists (`a1` and `am`).
  - **A1 Queue (FIFO)**: Buffers newly admitted entries, evicting the oldest when A1 capacity is reached to mitigate scan pollution.
  - **Am Queue (LRU)**: Stores entries that have proven utility through repeat access. Promoted entries are managed with strict LRU semantics.

> **Note**: HTTP routing, consistent hashing, and multi-node clustering are planned for subsequent phases and are **not yet implemented**.

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
* **Cache Engine**: The core storage coordinator managing entries, TTL expiration, and eviction.
* **Eviction Policies**: Configurable eviction strategies (LRU, LFU, 2Q) per cache server.
* **Node Management**: Membership tracking and node health monitoring.
* **Consistent Hashing**: Hash ring to evenly distribute keys across active nodes with minimal rehashing.
* **Metrics**: Cache hit/miss rates, memory usage, and operational latency tracking.
* **Concurrency**: Thread-safe operations across readers and writers.
* **Future Replication / Failover**: Redundant data replication and automatic failover handling.
