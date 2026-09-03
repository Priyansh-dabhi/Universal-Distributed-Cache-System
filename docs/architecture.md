# Architecture

## Current Architecture (Phase 8 — Distributed Router with Hash-Based Routing)

At present, the system implements a distributed router that sits between clients and multiple backend cache nodes. Clients communicate directly with the router, which deterministically maps each key to its owning cache node using simple modulo hashing:

```text
                         Clients
                            |
                            v
                     ┌─────────────┐
                     │   Router    │
                     │   (:9000)   │
                     │ hash(key)%N │
                     └──────┬──────┘
                            |
              ┌─────────────┼─────────────┐
              ↓             ↓             ↓
         ┌─────────┐   ┌─────────┐   ┌─────────┐
         │ Node 1  │   │ Node 2  │   │ Node 3  │
         │ :8001   │   │ :8002   │   │ :8003   │
         └────┬────┘   └────┬────┘   └────┬────┘
              ↓             ↓             ↓
           Cache 1       Cache 2       Cache 3
```

### Key Principles of Phase 8

* **Decoupled Responsibilities**: The router is responsible *only* for deterministic key routing and proxying HTTP requests. The backend cache nodes own cache storage, eviction policies, and TTL tracking.
* **Deterministic Modulo Hashing**: Each key is mapped to an owner using:
  $$\text{index} = \text{hash}(\text{key}) \pmod N$$
  where $N$ is the number of configured nodes and $\text{hash}$ is 32-bit FNV-1a (`hash/fnv`).
* **Single Key Owner**: Every key is assigned to exactly one backend node. No broadcast operations are performed.
* **Transparent Proxying**: Request payloads (such as JSON with TTL) and response statuses (`200`, `404`, etc.) are forwarded transparently.
* **Failure Semantics**: If a target node is unreachable, the router immediately returns `502 Bad Gateway`. **Failover is intentionally not implemented in this phase.**
* **Temporary Limitation of Modulo Hashing**: While simple and fast, changing node count ($N$) or node ordering remaps a large fraction of keys. Phase 9 addresses this by replacing modulo hashing with consistent hashing.

---

### End-to-End Request Flow

```text
Client
  │
  │  PUT /cache/user:123
  ▼
Router (:9000)
  │
  ├─► hash("user:123") % 3 = Node 3
  ▼
Cache Node 3 (:8003)
  │
  ├─► Stores "user:123" in its local Cache Engine
  ▼
HTTP 200 OK (propagated back to client via Router)
```

---

## Planned Architecture (Phase 9 — Consistent Hashing)

Phase 9 will introduce a consistent hashing ring with virtual nodes to minimize cache remapping during cluster resizing:

```text
                         Clients
                            |
                            v
                     ┌─────────────┐
                     │   Router    │
                     │             │
                     │ Consistent  │ (Phase 9 — Planned)
                     │  Hash Ring  │
                     └──────┬──────┘
                            |
              ┌─────────────┼─────────────┐
              ↓             ↓             ↓
           Node 1        Node 2        Node 3
```

### Planned Distributed Capabilities
* **Consistent Hash Ring**: Minimize cache disruption when adding or removing nodes.
* **Virtual Nodes**: Balance key distribution evenly across heterogeneous or homogeneous nodes.
* **Health Probing & Dynamic Node Membership**: Detect node liveness automatically.
* **Data Replication & Failover**: Replicate keys across successive ring nodes for high availability.
