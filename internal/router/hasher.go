package router

import (
	"hash/fnv"
)

// Hasher defines the contract for deterministic key hashing in the router.
type Hasher interface {
	Hash(key string) uint32
}

// FNVHasher implements Hasher using standard library 32-bit FNV-1a.
type FNVHasher struct{}

// NewFNVHasher creates a new FNVHasher.
func NewFNVHasher() *FNVHasher {
	return &FNVHasher{}
}

// Hash computes a deterministic 32-bit FNV-1a checksum of the key.
func (h *FNVHasher) Hash(key string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	return hasher.Sum32()
}
