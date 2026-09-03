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
Phase 5 — TTL                     Planned
```

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
| **GET** | $O(1)$ average | Map lookup + node recency/frequency/queue promotion update |
| **SET** | $O(1)$ average | Map lookup/insert + node insertion + optional eviction |
| **DELETE** | $O(1)$ average | Map removal + unlinking from linked list / frequency bucket / 2Q queue |
| **Promotion A1 → Am** | $O(1)$ | Unlink from A1 and prepend to Am head |
| **A1 eviction** | $O(1)$ | Unlink tail of A1 FIFO queue |
| **Am eviction** | $O(1)$ | Unlink tail of Am LRU queue |
| **Size** | $O(1)$ | Querying map length |

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
- Full test suite covering queue separation, promotions, LRU behavior, and concurrency
