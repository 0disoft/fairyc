package packageartifact

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

type genericRecord struct {
	key           packagecache.GenericKey
	interfaceHash moduleinterface.GenericHash
	payload       []byte
	size          uint64
}

// GenericCache is an isolated, process-bound MIR cache. It accepts no stable v4
// keys or v5 interfaces. A verified hit still requires whole-program reassembly
// against the caller's current compilation before executable publication.
type GenericCache struct {
	codec   *ProcessCodec
	limits  packagecache.Limits
	mu      sync.Mutex
	entries map[packagecache.GenericKey]*list.Element
	recent  *list.List
	stats   CacheStats
}

func NewGenericCache(limits packagecache.Limits, maxArtifactBytes uint64) (*GenericCache, error) {
	return newGenericCache(limits, maxArtifactBytes, false)
}

func newGenericCache(limits packagecache.Limits, maxArtifactBytes uint64, transfers bool) (*GenericCache, error) {
	if limits.MaxEntries <= 0 || limits.MaxBytes == 0 {
		return nil, fmt.Errorf("generic cache requires positive entry and byte limits")
	}
	createCodec := NewProcessCodec
	if transfers {
		createCodec = NewStorageTransferProcessCodec
	}
	codec, err := createCodec(maxArtifactBytes)
	if err != nil {
		return nil, err
	}
	return &GenericCache{codec: codec, limits: limits, entries: map[packagecache.GenericKey]*list.Element{}, recent: list.New()}, nil
}

func (cache *GenericCache) Identity() string {
	if cache == nil || cache.codec == nil {
		return ""
	}
	return cache.codec.Identity()
}

func (cache *GenericCache) Put(key packagecache.GenericKey, interfaceHash moduleinterface.GenericHash, artifact Artifact) error {
	if cache == nil || cache.codec == nil || cache.recent == nil {
		return fmt.Errorf("generic cache is not initialized")
	}
	if key == (packagecache.GenericKey{}) || interfaceHash == (moduleinterface.GenericHash{}) {
		return fmt.Errorf("generic cache key and interface must be set")
	}
	payload, err := cache.codec.Encode(artifact)
	if err != nil {
		return err
	}
	// Byte accounting is the retained serialized payload plus its two full digests.
	size := uint64(len(payload)) + 64
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if size > cache.limits.MaxBytes {
		return packagecache.ErrRecordTooLarge
	}
	if previous := cache.entries[key]; previous != nil {
		cache.remove(previous, false)
	}
	for len(cache.entries) >= cache.limits.MaxEntries || cache.stats.Store.Bytes > cache.limits.MaxBytes-size {
		cache.remove(cache.recent.Back(), true)
	}
	record := &genericRecord{key: key, interfaceHash: interfaceHash, payload: payload, size: size}
	cache.entries[key] = cache.recent.PushFront(record)
	cache.stats.Store.Bytes += size
	return nil
}

func (cache *GenericCache) Get(key packagecache.GenericKey, expected moduleinterface.GenericHash) Lookup {
	if cache == nil || cache.codec == nil || cache.recent == nil {
		return Lookup{Status: LookupRejected, Reason: "generic cache is not initialized"}
	}
	cache.mu.Lock()
	element := cache.entries[key]
	if element == nil {
		cache.stats.Misses++
		cache.stats.Store.Misses++
		cache.mu.Unlock()
		return Lookup{Status: LookupMiss}
	}
	cache.stats.Store.Hits++
	cache.recent.MoveToFront(element)
	record := element.Value.(*genericRecord)
	cache.mu.Unlock()
	// Records are immutable after insertion; decoding runs outside the LRU lock.
	if record.interfaceHash != expected {
		return cache.rejectGeneric(record, "cached generic interface differs from current source interface")
	}
	artifact, err := cache.codec.Decode(record.payload)
	if err != nil {
		return cache.rejectGeneric(record, err.Error())
	}
	cache.mu.Lock()
	cache.stats.VerifiedHits++
	cache.mu.Unlock()
	return Lookup{Status: LookupHit, Artifact: artifact}
}

func (cache *GenericCache) Stats() CacheStats {
	if cache == nil {
		return CacheStats{}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	result := cache.stats
	result.Store.Entries = len(cache.entries)
	return result
}

func (cache *GenericCache) rejectGeneric(record *genericRecord, reason string) Lookup {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	// A rejection must not delete a replacement inserted while decoding ran.
	if current := cache.entries[record.key]; current != nil && current.Value.(*genericRecord) == record {
		cache.remove(current, false)
	}
	cache.stats.Rejected++
	return Lookup{Status: LookupRejected, Reason: reason}
}

func (cache *GenericCache) remove(element *list.Element, eviction bool) {
	record := element.Value.(*genericRecord)
	delete(cache.entries, record.key)
	cache.recent.Remove(element)
	cache.stats.Store.Bytes -= record.size
	if eviction {
		cache.stats.Store.Evictions++
	}
}
