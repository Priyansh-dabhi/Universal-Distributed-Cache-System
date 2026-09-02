# Architecture

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

## Planned Components

* Router
* Cache Server
* Cache Engine
* Eviction Policies
* Node Management
* Consistent Hashing
* TTL / Expiration
* Metrics
* Concurrency
* Future Replication / Failover

For the eviction layer, the design will eventually support:

```text
Cache Engine
    |
    +-- LRU
    +-- LFU
    +-- 2Q
```
