# Architecture Concepts & System Guide

This document is a beginner-to-advanced reference manual explaining every architectural topic, design pattern, and component implemented in the **Universal Distributed Cache System**. 

It is structured progressively from first principles—starting from basic in-memory storage, moving through eviction algorithms and HTTP networking, up to consistent hashing, distributed routing, atomic telemetry, and performance benchmarking.

---

## Table of Contents

1. [Tier 1: In-Memory Caching & Concurrency Fundamentals](#1-tier-1-in-memory-caching--concurrency-fundamentals)
   - [1.1 In-Memory Cache](#11-in-memory-cache)
   - [1.2 Key-Value Pair](#12-key-value-pair)
   - [1.3 Thread Safety & Mutex Locking](#13-thread-safety--mutex-locking)
2. [Tier 2: Cache Eviction Policies](#2-tier-2-cache-eviction-policies)
   - [2.1 Cache Eviction](#21-cache-eviction)
   - [2.2 LRU (Least Recently Used)](#22-lru-least-recently-used)
   - [2.3 LFU (Least Frequently Used)](#23-lfu-least-frequently-used)
   - [2.4 2Q (Two-Queue / Scan-Resistant)](#24-2q-two-queue--scan-resistant)
   - [2.5 Cache Pollution & Scan Resistance](#25-cache-pollution--scan-resistance)
3. [Tier 3: Time-To-Live (TTL) & Expiration](#3-tier-3-time-to-live-ttl--expiration)
   - [3.1 TTL (Time-To-Live)](#31-ttl-time-to-live)
   - [3.2 Lazy Expiration vs Active Expiration](#32-lazy-expiration-vs-active-expiration)
4. [Tier 4: Cache Server Node & HTTP REST API](#4-tier-4-cache-server-node--http-rest-api)
   - [4.1 Cache Node](#41-cache-node)
   - [4.2 HTTP REST API & Handlers](#42-http-rest-api--handlers)
   - [4.3 Health Check (`/health`)](#43-health-check-health)
5. [Tier 5: Multi-Node Clustering](#5-tier-5-multi-node-clustering)
   - [5.1 Horizontal Scaling vs Vertical Scaling](#51-horizontal-scaling-vs-vertical-scaling)
   - [5.2 Port & Memory Isolation](#52-port--memory-isolation)
6. [Tier 6: Distributed Router (Reverse Proxy)](#6-tier-6-distributed-router-reverse-proxy)
   - [6.1 Distributed Router](#61-distributed-router)
   - [6.2 Reverse Proxying & Request Forwarding](#62-reverse-proxying--request-forwarding)
   - [6.3 Node Registry](#63-node-registry)
7. [Tier 7: Consistent Hashing & The Hash Ring](#7-tier-7-consistent-hashing--the-hash-ring)
   - [7.1 Modulo Hashing & Why It Fails](#71-modulo-hashing--why-it-fails)
   - [7.2 The Hash Ring](#72-the-hash-ring)
   - [7.3 Hash Function (FNV-1a / SHA-256)](#73-hash-function-fnv-1a--sha-256)
   - [7.4 Clockwise Node Lookup ($O(\log N)$)](#74-clockwise-node-lookup-olog-n)
   - [7.5 Virtual Nodes (Replicas) & Load Distribution](#75-virtual-nodes-replicas--load-distribution)
   - [7.6 Minimal Key Redistribution](#76-minimal-key-redistribution)
8. [Tier 8: Telemetry, Observability & Metrics](#8-tier-8-telemetry-observability--metrics)
   - [8.1 Telemetry & Observability](#81-telemetry--observability)
   - [8.2 Lock-Free Atomic Metrics (`sync/atomic`)](#82-lock-free-atomic-metrics-syncatomic)
   - [8.3 Metric Ownership Separation](#83-metric-ownership-separation)
   - [8.4 Hit Rate](#84-hit-rate)
   - [8.5 Observability Endpoints (`GET /metrics`)](#85-observability-endpoints-get-metrics)
9. [Tier 9: Benchmarking & Performance Evaluation](#9-tier-9-benchmarking--performance-evaluation)
   - [9.1 Microbenchmarks vs Workload Benchmarks](#91-microbenchmarks-vs-workload-benchmarks)
   - [9.2 Latency ($ns/op$) vs Throughput ($ops/sec$)](#92-latency-nsop-vs-throughput-opssec)
   - [9.3 Memory Allocations ($B/op$, $allocs/op$)](#93-memory-allocations-bop-allocsop)
   - [9.4 Synthetic Workloads (Zipfian, Uniform, Scan)](#94-synthetic-workloads-zipfian-uniform-scan)
10. [Tier 10: Complete Step-by-Step Life of a Request](#10-tier-10-complete-step-by-step-life-of-a-request)
    - [10.1 Walkthrough of a `PUT` Request](#101-walkthrough-of-a-put-request)
    - [10.2 Walkthrough of a `GET` Request](#102-walkthrough-of-a-get-request)

---

## 1. Tier 1: In-Memory Caching & Concurrency Fundamentals

### 1.1 In-Memory Cache
- **What it means**: Storing frequently needed data in the computer's Random Access Memory (RAM) instead of reading it from a slow hard drive, SSD, or external SQL database every single time.
- **Why we need it**: Reading from RAM takes ~50–100 nanoseconds ($0.0001$ ms). Reading from a database over a network or disk takes ~5–50 milliseconds ($5,000,000$ ns)—which is **50,000x slower**.
- **How it works in our app**: Located in [`internal/cache/cache.go`](../internal/cache/cache.go). The cache holds data in a native Go hash map (`map[string]*entry`), giving near-instant $O(1)$ lookups.
- **Example in action**:
  ```go
  c, _ := cache.NewLRU(1000)
  c.Set("user:101", "Alice")
  val, found := c.Get("user:101") // Returns "Alice", true in ~30 nanoseconds
  ```

---

### 1.2 Key-Value Pair
- **What it means**: The simplest way to store information: a unique identifier string (the **Key**) that maps directly to some piece of data (the **Value**).
- **Why we need it**: Looking up items by primary key is instantaneous ($O(1)$) compared to searching through arrays or filtering database rows.
- **How it works in our app**: We define an `entry` struct in [`internal/cache/cache.go`](../internal/cache/cache.go):
  ```go
  type entry struct {
      key       string
      value     string
      expiresAt *time.Time
      // ... pointers for eviction lists
  }
  ```
- **Example in action**:
  - Key: `"session:token_abc123"`
  - Value: `"user_id=42,role=admin"`

---

### 1.3 Thread Safety & Mutex Locking
- **What it means**: A **Mutex** (Mutual Exclusion lock) acts like a single bathroom door key. If Goroutine A is updating a data structure, Goroutine B must wait outside until Goroutine A finishes and unlocks the door.
- **Why we need it**: Go web servers handle hundreds of requests at the exact same moment across multiple CPU threads. If two goroutines modify a Go map or update linked-list pointers simultaneously, the Go runtime crashes with `fatal error: concurrent map writes`.
- **How it works in our app**: Located in [`internal/cache/cache.go`](../internal/cache/cache.go):
  - In our cache, `Get` is **not** a read-only operation because it updates eviction recency (in LRU), increments frequency (in LFU), or purges expired items (TTL). Therefore, `Get`, `Set`, and `Delete` acquire an exclusive `c.mu.Lock()`.
  - Read-only queries like `Size()` and `Capacity()` acquire a shared `c.mu.RLock()`.
- **Example in action**:
  ```go
  func (c *Cache) Get(key string) (string, bool) {
      c.mu.Lock()         // 1. Acquire exclusive lock
      defer c.mu.Unlock() // 3. Ensure lock is released even if code returns early
      return c.engine.get(key) // 2. Safely read and update recency
  }
  ```

---

## 2. Tier 2: Cache Eviction Policies

### 2.1 Cache Eviction
- **What it means**: When a cache reaches its maximum memory capacity (e.g., 1,000 items) and you try to add item #1,001, the cache must pick an existing item and throw it away (**evict** it) to make room.
- **Why we need it**: Server RAM is finite. Without eviction limits, a cache will keep growing until the server runs out of memory (OOM) and the operating system kills the process.
- **How it works in our app**: Each cache engine checks `if size >= capacity` before inserting a new key. The chosen algorithm determines *which* key gets evicted.

---

### 2.2 LRU (Least Recently Used)
- **What it means**: Evict the item that has not been read or written to for the longest amount of time.
- **Why we need it**: Real-world computer programs follow the **Temporal Locality Principle**: if you accessed a piece of data recently, you are very likely to access it again soon.
- **How it works in our app**: Located in [`internal/cache/lru.go`](../internal/cache/lru.go). Implemented using a **Hash Map + Doubly-Linked List**:
  - `Head` = Most Recently Used (MRU).
  - `Tail` = Least Recently Used (LRU).
  - When a key is accessed (`Get`), its node is unlinked and moved to the `Head` in $O(1)$ time (~31 ns).
  - When capacity is full, the node at the `Tail` is unlinked and removed.
- **Example in action**:
  ```text
  Capacity: 3
  Set(A), Set(B), Set(C)  --> List: [C, B, A]
  Get(A)                  --> List: [A, C, B] (A promoted to front)
  Set(D)                  --> B is evicted! List: [D, A, C]
  ```

---

### 2.3 LFU (Least Frequently Used)
- **What it means**: Evict the item that has been accessed the fewest total number of times, regardless of when the access happened.
- **Why we need it**: In workloads with clear "hot keys" (e.g., a viral product page or the company homepage), an item might not have been requested in the last 10 seconds, but it has been requested 50,000 times today. LRU would evict it; LFU preserves it.
- **How it works in our app**: Located in [`internal/cache/lfu.go`](../internal/cache/lfu.go). Implemented using **Frequency Buckets** with an $O(1)$ `minFreq` pointer:
  - Each frequency count ($1, 2, 3...$) has its own doubly-linked list.
  - When a key is accessed, its counter increments and it hops to the next frequency bucket in $O(1)$ time.
  - If multiple keys share the lowest frequency, an internal LRU tie-breaker picks the oldest one.
- **Example in action**:
  - `item_A` accessed 10 times.
  - `item_B` accessed 2 times.
  - `item_C` accessed 100 times.
  - New item arrives: `item_B` is evicted because frequency 2 is the lowest.

---

### 2.4 2Q (Two-Queue / Scan-Resistant)
- **What it means**: An advanced policy that divides cache memory into two separate sub-queues:
  1. **A1 (FIFO Queue)**: A probation waiting room for newly arrived keys.
  2. **Am (LRU Queue)**: The permanent club for keys that have proven their popularity by being requested at least twice.
- **Why we need it**: Standard LRU suffers from **Cache Pollution**. If a database backup or batch report scans through 100,000 records once, standard LRU flushes out all your hot, valuable cached data!
- **How it works in our app**: Located in [`internal/cache/twoq.go`](../internal/cache/twoq.go):
  - Newly inserted keys enter `A1` (first queue).
  - If a key is accessed only once, it moves through `A1` and gets evicted without ever touching `Am`.
  - If a key is requested a **second time** while in `A1`, it gets promoted to `Am` (permanent LRU cache).
  - In our benchmarks, 2Q delivered the fastest read latency (**22–24 ns/op**) and highest throughput under concurrent reads.
- **Example in action**:
  ```text
  1. Set(A)                --> Placed in A1 (Probation)
  2. One-time scan 1..5000 --> Fills & drains A1; Am remains untouched!
  3. Get(A) while in A1    --> PROMOTED to Am! Now permanent.
  ```

---

### 2.5 Cache Pollution & Scan Resistance
- **What it means**: **Cache Pollution** occurs when temporary, one-time data floods the cache and evicts frequently reused data. **Scan Resistance** is the ability of a cache policy (like 2Q) to detect and neutralize this problem.
- **How it works in our app**: In [`internal/cache/workload_bench_test.go`](../internal/cache/workload_bench_test.go), `BenchmarkWorkloadScanPollution` tests this by accessing 50 hot keys, running a 5,000-key cold scan, and measuring how many original hot keys survived.

---

## 3. Tier 3: Time-To-Live (TTL) & Expiration

### 3.1 TTL (Time-To-Live)
- **What it means**: A timer attached to a cached key telling the cache when the data becomes stale and should no longer be returned.
- **Why we need it**: Data changes in the primary database. An authentication token might expire after 15 minutes, or a weather forecast becomes stale after 1 hour.
- **How it works in our app**: Located in [`internal/cache/cache.go`](../internal/cache/cache.go):
  ```go
  func (c *Cache) SetWithTTL(key, value string, ttl time.Duration) {
      c.mu.Lock()
      defer c.mu.Unlock()
      c.engine.setWithTTL(key, value, ttl)
  }
  ```
  An entry stores `expiresAt = time.Now().Add(ttl)`.

---

### 3.2 Lazy Expiration vs Active Expiration
- **What it means**:
  - **Active Expiration**: A background goroutine with an infinite ticker loop continuously scanning through all keys to delete expired ones.
  - **Lazy Expiration**: The cache does not run background loops. Instead, when a user queries `Get("key")`, the cache checks `if time.Now().After(*entry.expiresAt)`. If expired, it deletes the item on the spot and returns `404 Not Found`.
- **Why our app uses Lazy Expiration**:
  - Active background loops consume CPU cycles and acquire locks periodically, causing latency spikes for client requests.
  - Lazy expiration costs **0 background CPU overhead** and checks timestamps using nanosecond-fast monotonic clock comparisons.
- **Example in action**:
  ```go
  c.SetWithTTL("coupon", "SAVE20", 2*time.Second)
  time.Sleep(3 * time.Second)
  val, found := c.Get("coupon") // Detects expiration: purges key, increments `expired` metric, returns "", false
  ```

---

## 4. Tier 4: Cache Server Node & HTTP REST API

### 4.1 Cache Node
- **What it means**: An independent, self-contained server process running on a dedicated TCP port (e.g., `:8001`), with its own private cache engine and memory space.
- **Why we need it**: In distributed systems, a single server cannot hold all the world's data or handle all incoming network traffic. Dividing the workload into distinct nodes is the first step toward horizontal scaling.
- **How it works in our app**: Located in [`internal/node/node.go`](../internal/node/node.go). A `Node` instance wraps a unique `ID` (e.g., `"node-1"`), a host/port, an eviction policy, and a running HTTP server.
- **Example CLI command**:
  ```powershell
  go run ./cmd/cache-server --id node-1 --port 8001 --capacity 1000 --policy lru
  ```

---

### 4.2 HTTP REST API & Handlers
- **What it means**: Standard web endpoints allowing any client—regardless of programming language (Go, Python, JavaScript, Java, cURL)—to read, write, and delete cache keys using standard HTTP verbs (`GET`, `PUT`, `DELETE`).
- **How it works in our app**: Located in [`internal/server/handlers.go`](../internal/server/handlers.go):

| HTTP Method | Route | Purpose | Example cURL |
| :--- | :--- | :--- | :--- |
| **`PUT`** | `/cache/{key}` | Insert or update a key | `curl -X PUT http://localhost:8001/cache/user:101 -d '{"value":"Alice"}' -H "Content-Type: application/json"` |
| **`GET`** | `/cache/{key}` | Retrieve a key | `curl http://localhost:8001/cache/user:101` |
| **`DELETE`** | `/cache/{key}` | Remove a key | `curl -X DELETE http://localhost:8001/cache/user:101` |
| **`GET`** | `/cache` | Inspect cache size & capacity | `curl http://localhost:8001/cache` |
| **`GET`** | `/health` | Node health & uptime | `curl http://localhost:8001/health` |
| **`GET`** | `/metrics` | JSON telemetry statistics | `curl http://localhost:8001/metrics` |

---

### 4.3 Health Check (`/health`)
- **What it means**: A lightweight endpoint that returns `200 OK` and `{"status":"ok"}` if the server is alive and responding.
- **Why we need it**: In production environments (like Kubernetes, Docker, or AWS ALB), automated load balancers ping `/health` every 5 seconds. If a node crashes, the load balancer stops routing user traffic to it.
- **How it works in our app**: Located in [`internal/server/handlers.go`](../internal/server/handlers.go). Returns node ID, status, and system health in under $100$ microseconds.

---

## 5. Tier 5: Multi-Node Clustering

### 5.1 Horizontal Scaling vs Vertical Scaling
- **Vertical Scaling (Scale Up)**: Buying a bigger server with more RAM and more CPU cores. (Expensive and hits hardware ceilings).
- **Horizontal Scaling (Scale Out)**: Running multiple smaller, independent servers working together as a cluster. (Cheaper, highly resilient, and virtually limitless).
- **How our app uses this**: Rather than running one huge 32GB server, we run three independent 2GB nodes (`node-1` on `:8001`, `node-2` on `:8002`, `node-3` on `:8003`).

---

### 5.2 Port & Memory Isolation
- **What it means**: Each cache node runs as an independent operating system process listening on its own port. They do **not** share RAM, pointers, or variables.
- **Why we need it**: Strict isolation guarantees that if `node-2` runs out of memory or crashes, `node-1` and `node-3` continue running without interruption.
- **How it works in our app**: Verified in [`internal/node/node_test.go`](../internal/node/node_test.go). Writing `"key-A"` to `:8001` will never accidentally appear on `:8002`.

---

## 6. Tier 6: Distributed Router (Reverse Proxy)

### 6.1 Distributed Router
- **What it means**: A central traffic cop that sits between client applications and the backend cache cluster.
- **Why we need it**: Clients should not need to know which of the 100 cache nodes holds `"user:101"`. Clients send every request to the Router at port `:9000`, and the Router decides which cache node owns that key and forwards the request transparently.
- **How it works in our app**: Located in [`internal/router/router.go`](../internal/router/router.go).

```text
               Client
                 │
                 ▼
         Router (:9000)
                 │
        ┌────────┴────────┐
        ▼                 ▼
  Node 1 (:8001)    Node 2 (:8002)
```

---

### 6.2 Reverse Proxying & Request Forwarding
- **What it means**: When the router receives an HTTP request for key `K`:
  1. It reads the incoming request body and HTTP headers.
  2. It identifies the target backend node (`http://localhost:8002`).
  3. It constructs an outbound HTTP request, sends it to the backend node, waits for the response, and mirrors the response back to the client.
- **How it works in our app**: Located in [`internal/router/router.go`](../internal/router/router.go):
  - Uses an optimized long-lived `http.Client` with connection pooling.
  - Uses Go's `http.NewRequestWithContext(req.Context(), ...)` to propagate timeouts. If the user disconnects, the router automatically cancels the outbound request to save backend CPU.
  - If a backend node is offline, the router returns a clean `502 Bad Gateway` instead of crashing.

---

### 6.3 Node Registry
- **What it means**: The router's internal directory listing all available physical cache nodes (IDs and addresses).
- **How it works in our app**: Located in [`internal/router/registry.go`](../internal/router/registry.go). Parses CLI configurations such as:
  ```powershell
  --nodes "node-1=http://localhost:8001,node-2=http://localhost:8002,node-3=http://localhost:8003"
  ```
  Validates URL schemes, hostnames, and duplicate node IDs on startup.

---

## 7. Tier 7: Consistent Hashing & The Hash Ring

### 7.1 Modulo Hashing & Why It Fails
- **What it means**: A naive way to pick a node for a key using simple remainder division:
  $$\text{Node Index} = \text{hash}(\text{key}) \pmod N$$
  *(where $N$ is the number of nodes in the cluster).*
- **Why it is disastrous in production**:
  - Suppose you have $N = 3$ nodes. A key hashes to $10$. $10 \pmod 3 = \text{Node } 1$.
  - Now you scale your cluster to $N = 4$ nodes. $10 \pmod 4 = \text{Node } 2$!
  - **The Catastrophe**: Changing $N$ changes the formula for almost every single key in the database! In our empirical test ([`internal/router/redistribution_test.go`](../internal/router/redistribution_test.go)), adding a 4th node remapped **75.07% of all keys**.
  - This results in a massive **Cache Stampede**: almost all subsequent requests miss the cache, crashing the primary database under sudden load.

---

### 7.2 The Hash Ring
- **What it means**: Instead of dividing by $N$, we map both **nodes** and **keys** onto a continuous mathematical circle (a ring) with values ranging from $0$ to $2^{32}-1$ ($4,294,967,295$).
- **How it works**:
  - The ring connects circularly: position $2^{32}-1$ wraps back around to position $0$.
  - Nodes are placed at specific points on the ring.
  - Keys are placed on the ring based on their hash value.

```text
                  Node A (pos: 1,000,000)
                         │
                   +-----+-----+
                  /             \
    Node C       /               \       Node B
(pos: 3,000,000) \               / (pos: 2,000,000)
                  +-------------+
                         │
                     Hash Ring
```

---

### 7.3 Hash Function (FNV-1a / SHA-256)
- **What it means**: A mathematical algorithm that takes an arbitrary string (like `"user:42"`) and transforms it into a deterministic 32-bit integer.
- **Why we need it**:
  - **Deterministic**: The same key always produces the exact same integer every time.
  - **Uniform**: Keys are scattered evenly across the entire 0 to 4.2 billion range with zero clustering.
- **How it works in our app**: Located in [`internal/router/ring.go`](../internal/router/ring.go) using standard FNV-1a hashing.

---

### 7.4 Clockwise Node Lookup ($O(\log N)$)
- **What it means**: When routing key $K$:
  1. Hash the key to find its position on the ring (e.g., position $1,500,000$).
  2. Travel **clockwise** along the circle.
  3. The **first physical node encountered** is the owner of that key!
  4. If the key's position is beyond the highest node on the ring, it wraps around to the very first node (at the beginning of the ring).
- **How it works in our app**: Located in [`internal/router/ring.go`](../internal/router/ring.go):
  - All node positions are stored in a sorted array `ringPositions []uint32`.
  - The lookup uses Go's `sort.Search` (Binary Search), finding the owner in $O(\log R)$ time (~28 nanoseconds in benchmarks).

---

### 7.5 Virtual Nodes (Replicas) & Load Distribution
- **What it means**: If you place only 3 physical nodes on a ring, they might land close together, causing one node to own 80% of the circle while others own 10% (hot spots).
- **The Solution**: Each physical node is given **150 virtual personas (replicas)** scattered randomly across the ring:
  - `node-1#0`, `node-1#1`, `node-1#2` ... `node-1#149`
  - `node-2#0`, `node-2#1`, `node-2#2` ... `node-2#149`
- **Why we need it**: With 450 total virtual points on the circle, statistical probability ensures that load is divided evenly across all physical machines (balanced within $\pm 2\%$).

---

### 7.6 Minimal Key Redistribution
- **What it means**: When adding a new node to the cluster:
  - In Modulo Hashing: **~75%** of keys are remapped.
  - In Consistent Hashing: Only keys located between the new node and its counter-clockwise predecessor move to the new node. Keys elsewhere on the ring are untouched!
- **Empirical result from our project**:
  - When expanding from 3 to 4 nodes across 10,000 keys:
    - Modulo Hashing remapped: **7,507 keys (75.07%)**.
    - Consistent Hashing remapped: only **1,337 keys (13.37%)**.
    - **100%** of those 1,337 keys went directly to the new node; **0 keys** were shuffled between old nodes!

---

## 8. Tier 8: Telemetry, Observability & Metrics

### 8.1 Telemetry & Observability
- **What it means**: Instrumentation that reports what is happening inside the system in real time (e.g., how many requests per second, error rates, cache hit rates, average latency).
- **The Golden Rule**:
  > **Correctness** tells us if the cache works.  
  > **Observability** tells us how the cache behaves.

---

### 8.2 Lock-Free Atomic Metrics (`sync/atomic`)
- **What it means**: Incrementing counters using specialized CPU hardware instructions (`LOCK XADD` on x86/AMD64) instead of software mutex locks.
- **Why we need it**: If 100 goroutines acquire a mutex lock just to increment a counter, the counter becomes a bottleneck. Atomic counters take **~1 nanosecond** and never block other threads.
- **How it works in our app**: Located in [`internal/metrics/metrics.go`](../internal/metrics/metrics.go):
  ```go
  type CacheMetrics struct {
      hits      atomic.Uint64
      misses    atomic.Uint64
      sets      atomic.Uint64
      evictions atomic.Uint64
  }
  func (m *CacheMetrics) IncHits() { m.hits.Add(1) }
  ```

---

### 8.3 Metric Ownership Separation
- **What it means**: Strictly separating which layer records which data.
- **Why we need it**: The Router does not know if a key was an LRU eviction or a cache hit—it only proxies bytes. The Cache Engine does not know what HTTP status code was returned over TCP.
- **Delineation in our app**:
  - **Router Metrics**: Total proxy requests, 2xx successes, 4xx/5xx errors, average proxy latency.
  - **Node Metrics**: Node-level HTTP requests and errors.
  - **Cache Engine Metrics**: Hits, misses, hit rate, sets, deletes, capacity evictions, and lazy TTL expirations.

---

### 8.4 Hit Rate
- **What it means**: The percentage of lookup requests that successfully found valid data in the cache:
  $$\text{Hit Rate} = \frac{\text{Hits}}{\text{Hits} + \text{Misses}}$$
- **Why it matters**: A hit rate of $95\%$ means that $95\%$ of database queries were eliminated, reducing database load by $20\text{x}$.

---

### 8.5 Observability Endpoints (`GET /metrics`)
- **What it means**: Dedicated HTTP endpoints returning point-in-time snapshots of system telemetry formatted as JSON.
- **How it works in our app**:
  - `GET http://localhost:8001/metrics` (Node telemetry)
  - `GET http://localhost:9000/metrics` (Router telemetry)
  ```json
  {
    "hits": 1200,
    "misses": 300,
    "hit_rate": 0.80,
    "sets": 1500,
    "evictions": 50,
    "requests": 1600,
    "avg_latency_ms": 1.42
  }
  ```

---

## 9. Tier 9: Benchmarking & Performance Evaluation

### 9.1 Microbenchmarks vs Workload Benchmarks
- **Microbenchmarks**: Measure a single isolated function in memory under optimal conditions (e.g., `BenchmarkGetExisting` measures how many nanoseconds a single `c.Get()` call takes).
- **Workload Benchmarks**: Measure system behavior under complex, multi-step scenarios simulating realistic user traffic (e.g., 50,000 operations following a skewed power-law distribution).
- **How it works in our app**:
  - Microbenchmarks: [`internal/cache/benchmark_test.go`](../internal/cache/benchmark_test.go)
  - Workload suite: [`internal/cache/workload_bench_test.go`](../internal/cache/workload_bench_test.go)

---

### 9.2 Latency ($ns/op$) vs Throughput ($ops/sec$)
- **Latency ($\text{ns/op}$)**: How long a single operation takes from start to finish (Lower is better).
  - *Example*: LRU Get takes **$37.57\text{ ns}$**.
- **Throughput ($\text{ops/sec}$)**: How many total operations the system can complete in one second (Higher is better):
  $$\text{Throughput} \approx \frac{1,000,000,000}{\text{Latency (ns)}}$$
  - *Example*: $24\text{ ns/op} \approx \mathbf{41,000,000\text{ operations per second}}$.

---

### 9.3 Memory Allocations ($B/op$, $allocs/op$)
- **What it means**: How much heap memory a function allocates (`B/op`) and how many times it invoked Go's garbage collection allocator (`allocs/op`).
- **Why it matters**: In high-performance Go programming, heap allocations trigger garbage collection (GC) pauses.
- **Our project's achievement**: All cache `Get` operations and `HashRing.GetNode` operations execute with **$0\text{ B/op}$ and $0\text{ allocs/op}$**—zero garbage collector overhead!

---

### 9.4 Synthetic Workloads (Zipfian, Uniform, Scan)
- **Zipfian / Hot-Key**: Power-law distribution where key #1 is requested $10\text{x}$ more than key #10, and $1,000\text{x}$ more than key #1,000. (Models real internet traffic).
- **Uniform Random**: Every key has an identical probability of being requested. (Models randomized UUID lookups with no locality).
- **Sequential Scan**: Continuous stream of unique keys (`1, 2, 3, 4 ...`). (Models database backup scans).

---

## 10. Tier 10: Complete Step-by-Step Life of a Request

To see how all components work together seamlessly, let us trace two requests through the entire system:

### 10.1 Walkthrough of a `PUT` Request

```text
Client: PUT http://localhost:9000/cache/user:101
Body:   {"value": "Alice", "ttl_seconds": 300}
```

1. **Client $\to$ Router**: Client sends HTTP PUT request to the Router on port `:9000`.
2. **Router Interception**: The Router starts a stopwatch, increments `router.requests`, and extracts key `"user:101"`.
3. **Consistent Hash Ring Selection**:
   - Router hashes `"user:101"` using FNV-1a $\to$ yields position `2,145,091,822`.
   - Router queries `HashRing.GetNode("user:101")`.
   - Binary search (`sort.Search`) scans clockwise $\to$ identifies virtual node `node-2#47`, which maps to physical **Node 2** (`http://localhost:8002`).
4. **Outbound Reverse Proxying**:
   - The Router constructs an HTTP PUT request targeting `http://localhost:8002/cache/user:101`.
   - Sends the request over a pooled TCP connection.
5. **Node 2 Processing**:
   - Node 2's HTTP server receives the request and parses the JSON payload.
   - Node 2 calls its cache engine: `Cache.SetWithTTL("user:101", "Alice", 300*time.Second)`.
6. **Cache Engine Storage & Eviction**:
   - Engine acquires `c.mu.Lock()`.
   - If cache capacity is full, the chosen policy (e.g., 2Q or LRU) identifies the victim node, unlinks it, and increments `metrics.evictions`.
   - Stores `"user:101"` with expiration timestamp `time.Now() + 300s`.
   - Engine increments `metrics.sets`.
   - Engine releases `c.mu.Unlock()`.
7. **Response Propagation**:
   - Node 2 records request duration in `node.metrics` and returns `200 OK` with `{"status":"stored"}`.
   - Router receives `200 OK`, records proxy duration in `router.metrics`, and mirrors the response to the client.

---

### 10.2 Walkthrough of a `GET` Request

```text
Client: GET http://localhost:9000/cache/user:101
```

1. **Client $\to$ Router**: Client sends HTTP GET request to `:9000`.
2. **Hash Ring Route**: Key `"user:101"` is hashed $\to$ routes deterministically to **Node 2** (`http://localhost:8002`).
3. **Node 2 Lookup**:
   - Node 2 calls `Cache.Get("user:101")`.
   - Engine acquires `c.mu.Lock()`.
   - Looks up `"user:101"` in internal hash map.
4. **TTL Expiration Check (Lazy Eviction)**:
   - Cache checks `expiresAt`. If `time.Now()` is past expiration, it deletes the entry, increments `metrics.expired` and `metrics.misses`, and returns `404 Not Found`.
   - If valid, engine promotes the key (MRU position in LRU, increments frequency in LFU, or moves to `Am` in 2Q).
   - Increments `metrics.hits`.
   - Releases `c.mu.Unlock()`.
5. **Response Delivery**:
   - Node 2 returns `200 OK` with `{"key":"user:101","value":"Alice"}`.
   - Router proxies response back to client in $\approx 200\text{ microseconds}$.
