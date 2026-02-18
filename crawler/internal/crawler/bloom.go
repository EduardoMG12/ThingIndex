package crawler

import (
	"hash"
	"hash/fnv"
	"math"
	"sync"
)

// BloomFilter is a thread-safe Bloom filter implementation.
type BloomFilter struct {
	bitset []bool
	k      uint
	m      uint
	lock   sync.RWMutex
}

// NewBloomFilter creates a new Bloom filter with the given size (m) and number of hash functions (k).
// Optimal m = -n*ln(p) / (ln(2)^2)
// Optimal k = (m/n) * ln(2)
// Where n is the expected number of elements and p is the false positive probability.
// For n = 1,000,000 and p = 0.01:
// m = 9,585,058 bits (~1.2MB)
// k = 7
func NewBloomFilter(n uint, p float64) *BloomFilter {
	m := uint(math.Ceil(float64(n) * math.Log(p) / math.Log(1.0/math.Pow(2.0, math.Log(2.0)))))
	k := uint(math.Round((float64(m) / float64(n)) * math.Log(2.0)))

	return &BloomFilter{
		bitset: make([]bool, m),
		k:      k,
		m:      m,
	}
}

// Add adds a string to the Bloom filter.
func (bf *BloomFilter) Add(data string) {
	bf.lock.Lock()
	defer bf.lock.Unlock()

	h := fnv.New64a()
	for i := uint(0); i < bf.k; i++ {
		h.Reset()
		h.Write([]byte{byte(i)})
		h.Write([]byte(data))
		hash := h.Sum64()
		idx := uint(hash % uint64(bf.m))
		bf.bitset[idx] = true
	}
}

// Test checks if a string is present in the Bloom filter.
// Returns true if the string might be present, false if it is definitely not present.
func (bf *BloomFilter) Test(data string) bool {
	bf.lock.RLock()
	defer bf.lock.RUnlock()

	h := fnv.New64a()
	for i := uint(0); i < bf.k; i++ {
		h.Reset()
		h.Write([]byte{byte(i)})
		h.Write([]byte(data))
		hash := h.Sum64()
		idx := uint(hash % uint64(bf.m))
		if !bf.bitset[idx] {
			return false
		}
	}
	return true
}

// TestAndAdd checks if a string is present, and if not, adds it.
// Returns true if the string was already present (might be), false if it was added.
func (bf *BloomFilter) TestAndAdd(data string) bool {
	bf.lock.Lock()
	defer bf.lock.Unlock()

	present := true
	h := fnv.New64a()
	
	// We need to calculate all hashes to add it anyway, so we can check presence meanwhile
	// But to be atomic, we recalculate or store.
	// Simple approach: Check all bits. If any is 0, it wasn't present. Set all to 1.
	
	// Pre-calculate positions to minimize lock time holding if we were doing complex stuff, 
	// but here hashing is fast.
	
	positions := make([]uint, bf.k)
	
	for i := uint(0); i < bf.k; i++ {
		h.Reset()
		h.Write([]byte{byte(i)})
		h.Write([]byte(data))
		hash := h.Sum64()
		positions[i] = uint(hash % uint64(bf.m))
		if !bf.bitset[positions[i]] {
			present = false
		}
	}
	
	if !present {
		for _, pos := range positions {
			bf.bitset[pos] = true
		}
	}
	
	return present
}

// Hash interface reuse helper
var _ hash.Hash64 = fnv.New64a()
