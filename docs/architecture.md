# Architecture

## Current Architecture (Phase 9 — Consistent Hashing)

The system implements a distributed cache cluster where client requests are routed through a central HTTP reverse proxy router backed by a **Consistent Hash Ring** with virtual nodes:

```text
                 Client
                    │
                    ▼
                 Router (:9000)
                    │
                    ▼
           ┌────────────────┐
           │   Hash Ring    │
           │ (Virtual Nodes)│
           └────────────────┘
                    │
                    ▼
              selected Node
                    │
       ┌────────────┼────────────┐
       ↓            ↓            ↓
  ┌─────────┐  ┌─────────┐  ┌─────────┐
  │ Node 1  │  │ Node 2  │  │ Node 3  │
  │ :8001   │  │ :8002   │  │ :8003   │
  └────┬────┘  └────┬────┘  └────┬────┘
       ↓            ↓            ↓
    Cache 1      Cache 2      Cache 3
       │            │            │
  (LRU/LFU/2Q) (LRU/LFU/2Q) (LRU/LFU/2Q)
     + TTL        + TTL        + TTL
```

---

## 1. Why Modulo Hashing Was Used in Phase 8

In Phase 8, the router used simple modulo hashing:
$$\text{index} = \text{hash}(\text{key}) \pmod N$$
where $N$ is the number of active cache nodes.

Modulo hashing was chosen in Phase 8 as a baseline because it is straightforward to implement, fast ($O(1)$ arithmetic), and produces a deterministic mapping when the cluster topology ($N$) never changes.

---

## 2. The Problem With Modulo Hashing When Node Count Changes

While simple, modulo hashing suffers from extreme cache disruption when nodes are added or removed ($N \to N \pm 1$). Because the divisor changes:
$$\text{hash}(\text{key}) \pmod 3 \neq \text{hash}(\text{key}) \pmod 4$$

When moving from 3 nodes to 4 nodes, mathematically **75% of all keys change owners**.
In general, when moving from $N$ to $N+1$ nodes under modulo hashing, the fraction of keys that must be remapped is:
$$1 - \frac{1}{N+1} = \frac{N}{N+1}$$

For large clusters, adding a single node remaps almost 100% of all keys, causing massive cache misses, stampedes on backend databases, and catastrophic performance degradation. Furthermore, modulo hashing is strictly order-dependent: changing the order of nodes in the configuration remaps almost all keys.

---

## 3. What Consistent Hashing Is

Consistent hashing is a distributed hashing technique where both keys and nodes are mapped onto a circular numerical space (the **hash ring**).

When the node set changes (nodes added or removed), consistent hashing ensures that only:
$$\approx \frac{K}{N}$$
keys need to be remapped on average (where $K$ is the total number of keys and $N$ is the number of nodes), rather than all keys.

---

## 4. What a Hash Ring Is

The 32-bit hash space ($[0, 2^{32}-1]$) is treated as a continuous circular ring:

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

- Position `0` connects seamlessly to position `2^32 - 1`.
- Physical nodes and their virtual replicas are assigned fixed positions along the perimeter of this ring based on the hash of their virtual node tokens.

---

## 5. How Virtual Nodes Work

If only physical nodes were placed on the ring (e.g. 3 points), large gaps could form between nodes, leading to severe load imbalance (hot spots).

To solve this, each physical node is assigned multiple **virtual nodes** (replicas) across the ring:

```text
node-1#0  --> hash("node-1#0")  --> position 124,912
node-1#1  --> hash("node-1#1")  --> position 892,104,112
...
node-1#99 --> hash("node-1#99") --> position 3,991,421,009

node-2#0  --> hash("node-2#0")  --> position 512,881
node-2#1  --> hash("node-2#1")  --> position 1,120,491,220
...
node-2#99 --> hash("node-2#99") --> position 4,102,881,993
```

By default, the system assigns **100 virtual nodes (replicas)** per physical node (configurable via `--replicas` CLI flag or `NewHashRing(replicas)`). This intersperses positions evenly across the ring, guaranteeing uniform load distribution.

### Collision Resolution Strategy
In the rare event that two virtual node tokens produce the exact same 32-bit hash value, **linear probing** (`pos++`) is employed. In addition, physical node IDs are sorted alphabetically before generating virtual node tokens, guaranteeing that ring construction and virtual node placement are 100% deterministic and independent of node input order.

---

## 6. How Key Routing Works

To route a key:

```text
key
 ↓
hash(key)
 ↓
ring position
 ↓
clockwise search
 ↓
physical node
```

1. Compute the 32-bit FNV-1a checksum: `hashVal = hash(key)`.
2. Locate the first virtual node position on the ring that is greater than or equal to `hashVal` (moving clockwise).
3. If `hashVal` is greater than the highest position on the ring, wrap around clockwise to the ring's first position (`ringPositions[0]`).
4. Look up the physical node associated with that virtual node position in `ringMap`.
5. Forward the request to that selected physical node.

---

## 7. What Happens When a Node Is Added

When a new node (e.g., Node D) is added:
1. Node D's virtual positions (`node-D#0` ... `node-D#99`) are inserted into the ring.
2. Only the keys whose hash values fall immediately before Node D's new virtual positions will be captured by Node D.
3. All other keys continue to map to their existing nodes without disruption.
4. When moving from 3 to 4 nodes, only **~15–25% of keys move** (compared to ~75% in modulo hashing).

---

## 8. What Happens When a Node Is Removed

When a node (e.g., Node B) is removed:
1. All virtual positions belonging to Node B are removed from the ring.
2. Keys previously owned by Node B fall through to the next available virtual node positions clockwise on the ring.
3. Keys belonging to remaining nodes (Node A, Node C) remain undisturbed on their respective nodes.
4. No data migration is performed; future requests automatically resolve to the new clockwise owners.

---

## 9. Why Binary Search Is Used

Scanning an unsorted array of virtual node positions sequentially requires $O(R)$ time, where $R$ is the total number of virtual nodes (e.g., $10 \text{ nodes} \times 100 = 1,000 \text{ positions}$). Doing this on every cache request adds unnecessary latency.

Instead, all virtual node hash positions are maintained in a sorted slice (`[]uint32`). When routing a key, the router uses standard library binary search:
```go
idx := sort.Search(len(ringPositions), func(i int) bool {
    return ringPositions[i] >= hashVal
})
```
This reduces lookup time to **$O(\log R)$**. In benchmarks, this executes in **~121 nanoseconds** per lookup (> 8.2 million lookups/second).

---

## 10. Complexity Analysis

| Operation | Complexity | Description |
| :--- | :---: | :--- |
| **`GetNode(key)`** | $O(\log R)$ | 32-bit FNV-1a hash calculation + binary search over $R$ virtual node positions |
| **`AddNode(node)`** | $O(R \log R)$ | Generate $R_{\text{node}}$ virtual tokens, resolve collisions, sort ring positions |
| **`RemoveNode(nodeID)`**| $O(R \log R)$ | Rebuild/filter remaining virtual positions and sort |
| **`Nodes()`** | $O(N \log N)$ | Sort $N$ physical nodes by ID for deterministic ordering |

*(where $N$ is the number of physical nodes, and $R = N \times \text{replicas}$ is the total number of virtual nodes on the ring).*

---

## 11. Difference Between Phase 8 and Phase 9

| Property | Phase 8 (Modulo Hashing) | Phase 9 (Consistent Hashing) |
| :--- | :--- | :--- |
| **Algorithm** | `hash(key) % N` | Hash Ring + Clockwise Binary Search |
| **Node Addition Churn** | $\approx 75\%$ of keys remapped (3 $\to$ 4 nodes) | $\approx 15–25\%$ of keys remapped (3 $\to$ 4 nodes) |
| **Node Ordering** | Order-sensitive; permuting nodes alters mappings | Order-independent; nodes normalized by ID |
| **Hotspot Mitigation** | Relies solely on uniform hash distribution | Replicas/Virtual nodes smooth out variance across ring |
| **Routing Complexity** | $O(1)$ arithmetic | $O(\log R)$ binary search (~121 ns) |
| **Component Architecture**| Hashing coupled within router route logic | Reusable, independently testable `HashRing` component |

---

## Clean Separation of Responsibilities

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

- **Router**: HTTP proxying, connection pooling, client timeouts, routing delegation.
- **HashRing**: Hash calculation, virtual node replica management, collision resolution, key-to-node clockwise mapping, thread safety.
- **Cache Server**: Local HTTP API handlers (`PUT`, `GET`, `DELETE`, `/health`, `/node`), JSON validation, status codes.
- **Cache Engine**: Eviction policies (LRU, LFU, 2Q), lazy TTL expiration, concurrency safety.
