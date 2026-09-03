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
* 2Q (Planned)

## Planned Architecture

Client → Router → Cache Nodes

## Status

```text
Phase 1 — Basic In-Memory Cache       ✓
Phase 2 — LRU Eviction               ✓
Phase 3 — LFU Eviction               ✓
Phase 4 — 2Q                          Planned
```

---

## Eviction Policies Overview

### LRU (Least Recently Used)
Removes the item that has not been used recently. Every successful `GET` or `SET` moves the item to the MRU (Most Recently Used) position.

### LFU (Least Frequently Used)
Removes the item that has been used the fewest times across its lifetime in cache. Every access (`GET` or `SET` update) increments the item's access counter.

### LFU Tie-Breaking
When multiple entries share the same lowest access frequency, eviction tie-breaking is resolved using **LRU among equally frequent items**—the least recently accessed item within that frequency bucket is evicted first.

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

---

## Time Complexity

| Operation | Expected Complexity | Description |
| :--- | :---: | :--- |
| **GET** | $O(1)$ average | Map lookup + node recency/frequency update |
| **SET** | $O(1)$ average | Map lookup/insert + node insertion + optional eviction |
| **DELETE** | $O(1)$ average | Map removal + unlinking from linked list / frequency bucket |
| **Frequency update** | $O(1)$ average | Unlinking from bucket $f$ and prepending to bucket $f+1$ |
| **Eviction** | $O(1)$ average | Removing `tail.prev` of `freqBuckets[minFreq]` |
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
- Full test coverage for eviction, tie-breaking, capacity, and concurrency
