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
* LFU (Planned)
* 2Q (Planned)

## Planned Architecture

Client → Router → Cache Nodes

## Status

🚧 In active development — Phase 1 & Phase 2 complete

- **Phase 1 — Basic In-Memory Cache**: Implemented
- **Phase 2 — LRU Eviction**: Implemented

---

## Phase 1 — Basic In-Memory Cache

Implemented:

- SET
- GET
- DELETE
- Size tracking
- Concurrent access protection
- Unit tests

---

## Phase 2 — LRU Eviction Policy

### What is LRU?
**LRU (Least Recently Used)** is a cache eviction algorithm that discards the least recently accessed items first when the cache reaches its configured maximum capacity. Every successful `GET` and `SET` operation marks the targeted item as the most recently used (MRU).

### Why an Eviction Policy is Needed
In-memory caches operate within bounded memory constraints. Without an eviction policy, continuously adding entries causes unbounded memory consumption and eventual out-of-memory (OOM) crashes. An eviction policy ensures the cache maintains a fixed working set of the most relevant items.

### Why HashMap + Doubly Linked List?
- **HashMap (`map[string]*node`)**: Provides $O(1)$ average-time lookup for any key.
- **Doubly Linked List**: Maintains recency ordering. Placing the MRU item at the head and the LRU item at the tail allows:
  - Moving an accessed/updated node to the front in $O(1)$ time.
  - Evicting the least recently used node from the tail in $O(1)$ time.
  - Removing a deleted node in $O(1)$ time without scanning.

### Time Complexity

| Operation | Complexity | Description |
| --------- | :--------: | ----------- |
| **GET**   |   $O(1)$   | Hash map lookup + move node to MRU head |
| **SET**   |   $O(1)$   | Hash map lookup/insert + move/add to MRU head + optional LRU tail eviction |
| **DELETE**|   $O(1)$   | Hash map deletion + unlink node from doubly linked list |
| **Eviction**| $O(1)$   | Unlink tail.prev node and delete from hash map |
| **Size**  |   $O(1)$   | Reading map length |
