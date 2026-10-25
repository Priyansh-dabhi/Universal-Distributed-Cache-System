export interface Concept {
  id: string;
  title: string;
  tagline: string;
  what: string;
  why: string;
  how: string;
  sourceFile: string;
  codeSnippet: string;
  interactiveLink?: {
    tabPath: string;
    label: string;
  };
}

export interface ArchitectureTier {
  tierNumber: number;
  title: string;
  summary: string;
  concepts: Concept[];
}

export const ARCHITECTURE_TIERS: ArchitectureTier[] = [
  {
    tierNumber: 1,
    title: 'In-Memory Caching & Concurrency',
    summary: 'Sub-microsecond RAM storage, key-value lookup, and Go RWMutex thread safety.',
    concepts: [
      {
        id: 'in-memory-cache',
        title: '1.1 In-Memory RAM Storage',
        tagline: '50,000× faster than disks and network databases',
        what: 'Storing frequently accessed key-value pairs directly in Random Access Memory (RAM) rather than issuing expensive disk reads or SQL queries.',
        why: 'RAM read latency is ~50–100 nanoseconds (0.0001 ms). Reading from an external relational database across network or disk takes ~5–50 ms—which is 50,000x slower.',
        how: 'Implemented via a Go hash map (`map[string]*entry`) providing near-instant O(1) lookups.',
        sourceFile: 'internal/cache/cache.go',
        codeSnippet: `// internal/cache/cache.go
c, _ := cache.NewLRU(1000)
c.Set("user:101", "Alice")
val, found := c.Get("user:101") // Returns "Alice", true in ~30 nanoseconds`,
        interactiveLink: { tabPath: '/playground', label: 'Try in CRUD Studio' },
      },
      {
        id: 'mutex-locking',
        title: '1.2 Thread Safety & Mutex Locking',
        tagline: 'Preventing concurrent map write panics under massive multi-threading',
        what: 'Mutual Exclusion (sync.RWMutex) ensuring only one goroutine mutates cache structures or eviction linked-list pointers at any given moment.',
        why: 'Go web servers serve hundreds of concurrent goroutines. Without mutex synchronization, concurrent writes corrupt map memory and crash the runtime with fatal error: concurrent map writes.',
        how: 'In our cache, Get acquires an exclusive lock (c.mu.Lock()) because it mutates recency/frequency pointers. Size and Capacity acquire read locks (c.mu.RLock()).',
        sourceFile: 'internal/cache/cache.go',
        codeSnippet: `func (c *Cache) Get(key string) (string, bool) {
    c.mu.Lock()         // Acquire exclusive lock
    defer c.mu.Unlock() // Ensure lock is released on return
    return c.engine.get(key)
}`,
        interactiveLink: { tabPath: '/playground', label: 'Test Concurrent Ops' },
      },
    ],
  },
  {
    tierNumber: 2,
    title: 'Cache Eviction Policies (LRU, LFU, 2Q)',
    summary: 'Algorithm engines that discard items when capacity is reached to prevent OOM crashes.',
    concepts: [
      {
        id: 'lru-policy',
        title: '2.1 LRU (Least Recently Used)',
        tagline: 'Exploiting temporal locality with a doubly linked list',
        what: 'Evicts the item that has not been read or written to for the longest duration.',
        why: 'Programs exhibit temporal locality: if an item was accessed recently, it is highly probable to be accessed again soon.',
        how: 'Uses a doubly linked list with sentinel Head (MRU) and Tail (LRU). Items accessed move to Head in O(1). The Tail item is evicted on overflow.',
        sourceFile: 'internal/cache/lru.go',
        codeSnippet: `// internal/cache/lru.go
if len(l.items) >= l.capacity {
    if lru := l.removeLRU(); lru != nil {
        delete(l.items, lru.key)
    }
}
l.addToFront(newNode)`,
        interactiveLink: { tabPath: '/eviction', label: 'Inspect LRU Visualizer' },
      },
      {
        id: 'lfu-policy',
        title: '2.2 LFU (Least Frequently Used)',
        tagline: 'Tracking access frequencies with LRU tie-breaking',
        what: 'Evicts the item with the lowest access count across its lifetime.',
        why: 'Accounts for long-term popularity: an item accessed 1,000 times will not be evicted simply because it was not touched in the last two seconds.',
        how: 'Maintains frequency buckets (Map[int]*lfuList). Accesses increment frequency and lift items to higher buckets. Evictions pop from minFreq.',
        sourceFile: 'internal/cache/lfu.go',
        codeSnippet: `// internal/cache/lfu.go
func (c *lfuCache) evict() {
    bucket := c.freqBuckets[c.minFreq]
    evicted := bucket.popTail()
    delete(c.items, evicted.key)
}`,
        interactiveLink: { tabPath: '/eviction', label: 'Inspect LFU Visualizer' },
      },
      {
        id: 'twoq-policy',
        title: '2.3 2Q (Two-Queue Scan Resistance)',
        tagline: 'Shielding valuable hot data from unindexed batch scans',
        what: 'Separates entries into two queues: A1 (FIFO probation) and Am (LRU main cache).',
        why: 'In standard LRU, an unindexed database query reading 10,000 records sweeps out all hot data (cache pollution). 2Q confines scans to A1.',
        how: 'New keys enter A1. Only when accessed a second time are they promoted to Am. Bulk scans pass through A1 without disturbing Am.',
        sourceFile: 'internal/cache/twoq.go',
        codeSnippet: `// internal/cache/twoq.go
if node.qType == queueA1 {
    // Second access: promote from probation A1 to main Am
    c.promoteToAm(node)
}`,
        interactiveLink: { tabPath: '/eviction', label: 'Inspect 2Q Visualizer' },
      },
    ],
  },
  {
    tierNumber: 3,
    title: 'Time-To-Live (TTL) & Expiration',
    summary: 'Stale data prevention via lazy evaluation and active TTL timestamps.',
    concepts: [
      {
        id: 'lazy-ttl',
        title: '3.1 Lazy Expiration Strategy',
        tagline: 'Zero background CPU overhead for expiration checks',
        what: 'Entries store an expiration timestamp. When a request accesses a key, its expiry is checked lazily.',
        why: 'Running active timer goroutines for thousands of keys wastes CPU and creates lock contention. Lazy expiration purges on-demand with zero background cost.',
        how: 'If time.Now() >= entry.expiresAt on Get(), the entry is immediately purged, expired counters increment, and a cache miss is returned.',
        sourceFile: 'internal/cache/cache.go',
        codeSnippet: `// Lazy check on Get()
if n.isExpired(time.Now()) {
    l.removeNode(n)
    delete(l.items, key)
    return "", false, true // expired = true
}`,
        interactiveLink: { tabPath: '/playground', label: 'Test TTL in Studio' },
      },
    ],
  },
  {
    tierNumber: 4,
    title: 'Distributed Router & Reverse Proxy',
    summary: 'Reverse-proxying client HTTP requests using consistent hash key routing.',
    concepts: [
      {
        id: 'reverse-proxy',
        title: '4.1 Reverse Proxying & Key Dispatch',
        tagline: 'Transparent HTTP proxying shielding backend nodes',
        what: 'The router acts as the single public entry point for clients, transparently proxying requests to the appropriate backend node.',
        why: 'Clients do not need to know cluster topology or node ports; they simply send HTTP requests to router:9000.',
        how: 'Router extracts the URL key, queries the consistent hash ring for the target node, and proxies the request via http.Client with connection pooling.',
        sourceFile: 'internal/router/router.go',
        codeSnippet: `targetNode, _ := r.Route(key)
targetURL := fmt.Sprintf("%s/cache/%s", targetNode.Address, url.PathEscape(key))
outboundReq, _ := http.NewRequestWithContext(ctx, req.Method, targetURL, req.Body)
resp, _ := r.client.Do(outboundReq)`,
        interactiveLink: { tabPath: '/playground', label: 'Inspect Proxy Hops' },
      },
    ],
  },
  {
    tierNumber: 5,
    title: 'Consistent Hashing & The Hash Ring',
    summary: 'Minimal key redistribution (1/(N+1)), virtual nodes, and O(log R) clockwise routing.',
    concepts: [
      {
        id: 'hash-ring',
        title: '5.1 Circular Hash Ring (2³² - 1)',
        tagline: 'Clockwise binary search routing around a circular perimeter',
        what: 'Treats the output range of a 32-bit hash function as an unbroken circle from 0 to 4,294,967,295.',
        why: 'Traditional modulo hashing (hash % N) breaks almost 100% of key mappings when a node is added or removed, causing catastrophic cache stampedes.',
        how: 'Keys are hashed and placed on the circle. The router searches clockwise for the nearest virtual node position using binary search in O(log R).',
        sourceFile: 'internal/router/ring.go',
        codeSnippet: `// Clockwise binary search in O(log R)
idx := sort.Search(len(r.ringPositions), func(i int) bool {
    return r.ringPositions[i] >= hashVal
})
if idx == len(r.ringPositions) {
    idx = 0 // Circular wrap-around
}`,
        interactiveLink: { tabPath: '/ring', label: 'Open 360° Ring' },
      },
      {
        id: 'virtual-nodes',
        title: '5.2 Virtual Nodes (Replicas)',
        tagline: 'Eliminating non-uniform clustering and hot spots',
        what: 'Creating multiple virtual token positions per physical node (e.g. node-1#0, node-1#1 ... node-1#99).',
        why: 'With few physical nodes, random hash placement leaves huge gaps where one node absorbs 80% of traffic. Virtual nodes disperse points uniformly.',
        how: 'Each physical node registers R virtual positions on the ring. The standard deviation of key ownership across nodes drops dramatically.',
        sourceFile: 'internal/router/ring.go',
        codeSnippet: `for i := 0; i < r.replicas; i++ {
    vnodeToken := fmt.Sprintf("%s#%d", id, i)
    pos := r.hasher.Hash(vnodeToken)
    r.ringMap[pos] = node
}`,
        interactiveLink: { tabPath: '/ring', label: 'Toggle Virtual Tokens' },
      },
      {
        id: 'minimal-redistribution',
        title: '5.3 Minimal Key Redistribution',
        tagline: 'Only 1/(N+1) keys moved on node addition vs 100% in modulo',
        what: 'When a new node joins, only keys falling on its clockwise arc segments are transferred. All other nodes retain their resident caches.',
        why: 'Enables safe zero-downtime horizontal scaling without overwhelming backend primary databases.',
        how: 'In a 4-node cluster growing to 5 nodes, only ~20% of keys are reassigned, leaving 80% untouched.',
        sourceFile: 'internal/router/ring.go',
        codeSnippet: `// Theoretical keys remapped:
// Consistent Hashing: 1 / (N + 1)
// Traditional Modulo: N / (N + 1)`,
        interactiveLink: { tabPath: '/ring', label: 'View Rebalancing Proof' },
      },
    ],
  },
  {
    tierNumber: 6,
    title: 'Atomic Telemetry & Workload Benchmarks',
    summary: 'Lock-free atomic performance counters, Zipfian distributions, and throughput profiling.',
    concepts: [
      {
        id: 'atomic-metrics',
        title: '6.1 Lock-Free Atomic Counters',
        tagline: 'Zero lock contention on hot telemetry paths',
        what: 'Using CPU atomic instructions (sync/atomic atomic.Uint64) for recording hits, misses, and requests.',
        why: 'Acquiring a mutex lock on every telemetry increment causes severe CPU core thrashing at 500,000+ operations/sec. Atomics execute in hardware nanoseconds.',
        how: 'Counters execute lock-free Add(1) and Load() without ever blocking request processing threads.',
        sourceFile: 'internal/metrics/metrics.go',
        codeSnippet: `type CacheMetrics struct {
    hits      atomic.Uint64
    misses    atomic.Uint64
    sets      atomic.Uint64
}
func (m *CacheMetrics) IncHits() { m.hits.Add(1) }`,
        interactiveLink: { tabPath: '/metrics', label: 'Inspect Telemetry Stream' },
      },
      {
        id: 'zipfian-workload',
        title: '6.2 Zipfian & Synthetic Workloads',
        tagline: 'Validating hit rates under Pareto 80/20 power-law distributions',
        what: 'Benchmarking the cache under realistic access patterns where a tiny fraction of items receive the overwhelming majority of queries.',
        why: 'Testing with uniform random traffic yields artificially low hit rates that do not reflect production user behavior.',
        how: 'Zipfian distributions power real-world tests where top items stay hot in cache, validating eviction efficiency.',
        sourceFile: 'internal/cache/workload_bench_test.go',
        codeSnippet: `// 80% requests target top 20% key space
if rand.Float64() < 0.80 {
    key = hotKeyPool[rand.Intn(len(hotKeyPool))]
}`,
        interactiveLink: { tabPath: '/metrics', label: 'Run Zipfian Workload' },
      },
    ],
  },
];
