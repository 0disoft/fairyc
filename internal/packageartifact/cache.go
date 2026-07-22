package packageartifact

import (
	"fmt"
	"sync"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

const (
	LookupHit      = "hit"
	LookupMiss     = "miss"
	LookupRejected = "rejected"
)

// Lookup is one verified process-cache result. Rejected entries are quarantined and deliberately
// returned without an artifact so the caller can fall back to ordinary compilation.
type Lookup struct {
	Status   string
	Artifact Artifact
	Reason   string
}

// CacheStats separates verified reuse from raw byte-store hits. Store.Hits can include a payload
// later rejected by the process codec or public-interface check.
type CacheStats struct {
	Store        packagecache.Stats
	VerifiedHits uint64
	Misses       uint64
	Rejected     uint64
}

// Cache owns a process-bound codec and the byte-bounded LRU containing its complete payloads.
type Cache struct {
	codec *ProcessCodec
	store *packagecache.MemoryStore

	mu           sync.Mutex
	verifiedHits uint64
	misses       uint64
	rejected     uint64
}

// NewCache creates one independently isolated cache. Both record and artifact byte bounds are
// mandatory; Stage 0 intentionally has no hidden defaults.
func NewCache(limits packagecache.Limits, maxArtifactBytes uint64) (*Cache, error) {
	store, err := packagecache.NewMemoryStore(limits)
	if err != nil {
		return nil, err
	}
	codec, err := NewProcessCodec(maxArtifactBytes)
	if err != nil {
		return nil, err
	}
	return &Cache{codec: codec, store: store}, nil
}

// Identity returns the process-private compatibility value that callers must include in cache-key
// compiler-build identity.
func (cache *Cache) Identity() string {
	if cache == nil || cache.codec == nil {
		return ""
	}
	return cache.codec.Identity()
}

// Put verifies and encodes artifact before atomically inserting its complete bytes into the LRU.
func (cache *Cache) Put(key packagecache.Key, publicInterface moduleinterface.Hash, artifact Artifact) error {
	if cache == nil || cache.codec == nil || cache.store == nil {
		return fmt.Errorf("process-local package artifact cache is not initialized")
	}
	encoded, err := cache.codec.Encode(artifact)
	if err != nil {
		return err
	}
	return cache.store.Put(packagecache.Record{
		Key: key, PublicInterface: publicInterface,
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactMIR, Data: encoded}},
	})
}

// Get returns only a fully decoded and verified artifact with the expected output interface. A
// malformed, foreign-process, or contradictory entry becomes a quarantined safe miss.
func (cache *Cache) Get(key packagecache.Key, expectedInterface moduleinterface.Hash) Lookup {
	if cache == nil || cache.codec == nil || cache.store == nil {
		return Lookup{Status: LookupRejected, Reason: "process-local package artifact cache is not initialized"}
	}
	record, found := cache.store.Get(key)
	if !found {
		cache.recordMiss()
		return Lookup{Status: LookupMiss}
	}
	if record.PublicInterface != expectedInterface {
		return cache.reject(record, "cached public interface does not match the expected package interface")
	}
	if len(record.Artifacts) != 1 || record.Artifacts[0].Kind != packagecache.ArtifactMIR || len(record.Artifacts[0].Data) == 0 {
		return cache.reject(record, "cached package artifact set is not exactly one MIR payload")
	}
	artifact, err := cache.codec.Decode(record.Artifacts[0].Data)
	if err != nil {
		return cache.reject(record, err.Error())
	}
	cache.mu.Lock()
	cache.verifiedHits++
	cache.mu.Unlock()
	return Lookup{Status: LookupHit, Artifact: artifact}
}

// Stats returns a race-free observational snapshot. Store counters are independently synchronized
// by MemoryStore, so concurrent operations may land on either side of the two snapshots; verified
// counters describe completed Cache.Get outcomes.
func (cache *Cache) Stats() CacheStats {
	if cache == nil || cache.store == nil {
		return CacheStats{}
	}
	cache.mu.Lock()
	stats := CacheStats{VerifiedHits: cache.verifiedHits, Misses: cache.misses, Rejected: cache.rejected}
	cache.mu.Unlock()
	stats.Store = cache.store.Stats()
	return stats
}

func (cache *Cache) reject(record packagecache.Record, reason string) Lookup {
	cache.store.DeleteIfMatch(record)
	cache.mu.Lock()
	cache.rejected++
	cache.mu.Unlock()
	return Lookup{Status: LookupRejected, Reason: reason}
}

func (cache *Cache) recordMiss() {
	cache.mu.Lock()
	cache.misses++
	cache.mu.Unlock()
}
