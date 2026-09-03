# Architecture

## Current System (Phase 1)

At present, the system implements a standalone, thread-safe in-memory key-value cache:

```text
Application
     |
     v
   Cache
     |
     v
In-Memory Map
```

Eviction policies (LRU, LFU, 2Q) and distributed components (HTTP routing, consistent hashing, clustering, replication) are planned for subsequent phases.

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
    +-- LRU
    +-- LFU
    +-- 2Q
```
