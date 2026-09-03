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

* LRU
* LFU
* 2Q

## Planned Architecture

Client → Router → Cache Nodes

## Status

🚧 In active development — Phase 1 complete

## Phase 1 — Basic In-Memory Cache

Implemented:

- SET
- GET
- DELETE
- Size tracking
- Concurrent access protection
- Unit tests
