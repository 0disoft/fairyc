package packageartifact

import (
	"fmt"
	"sync"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// StableLookup is one cross-process cache result. A hit has passed the disk record checks, stable
// artifact codec, expected public-interface check, and expected package-instance check.
type StableLookup struct {
	Status   string
	Artifact StableArtifact
	Reason   string
}

// StableCacheStats separates reusable stable artifacts from lower-level disk-record observations.
type StableCacheStats struct {
	Store        packagecache.DiskStats
	VerifiedHits uint64
	Misses       uint64
	Rejected     uint64
}

// StableCache connects exact-version stable package MIR bytes to one bounded disposable DiskStore.
// It owns no default path, lifetime, or size policy; every policy input comes from its caller.
type StableCache struct {
	store            *packagecache.DiskStore
	maxArtifactBytes uint64

	mu           sync.Mutex
	verifiedHits uint64
	misses       uint64
	rejected     uint64
}

// NewStableCache creates a cross-process cache adapter with mandatory record and artifact bounds.
func NewStableCache(root string, limits packagecache.DiskLimits, maxArtifactBytes uint64) (*StableCache, error) {
	if maxArtifactBytes == 0 {
		return nil, fmt.Errorf("stable package artifact byte limit must be positive")
	}
	if maxArtifactBytes > limits.MaxRecordBytes {
		return nil, fmt.Errorf("stable package artifact byte limit cannot exceed the disk record byte limit")
	}
	store, err := packagecache.NewDiskStore(root, limits)
	if err != nil {
		return nil, err
	}
	return &StableCache{store: store, maxArtifactBytes: maxArtifactBytes}, nil
}

// Put verifies a process-local artifact, projects it to stable identities, canonically encodes the
// result, and publishes one complete immutable record.
func (cache *StableCache) Put(key packagecache.Key, publicInterface moduleinterface.Hash, artifact Artifact) error {
	if cache == nil || cache.store == nil || cache.maxArtifactBytes == 0 {
		return fmt.Errorf("stable package artifact cache is not initialized")
	}
	stable, err := DescribeStableArtifact(artifact, cache.maxArtifactBytes)
	if err != nil {
		return err
	}
	encoded, err := EncodeStableArtifact(stable, cache.maxArtifactBytes)
	if err != nil {
		return err
	}
	return cache.store.Put(packagecache.Record{
		Key: key, PublicInterface: publicInterface,
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactMIR, Data: encoded}},
	})
}

// Get returns only a stable artifact for the exact expected interface and package instance. Every
// semantic rejection quarantines only the complete record generation that produced it.
func (cache *StableCache) Get(key packagecache.Key, expectedInterface moduleinterface.Hash, expectedPackage packagegraph.InstanceID) StableLookup {
	if cache == nil || cache.store == nil || cache.maxArtifactBytes == 0 {
		return StableLookup{Status: LookupRejected, Reason: "stable package artifact cache is not initialized"}
	}
	record, found := cache.store.Get(key)
	if !found {
		cache.recordStableMiss()
		return StableLookup{Status: LookupMiss}
	}
	if record.PublicInterface != expectedInterface {
		return cache.rejectStable(record, "cached public interface does not match the expected package interface")
	}
	if len(record.Artifacts) != 1 || record.Artifacts[0].Kind != packagecache.ArtifactMIR || len(record.Artifacts[0].Data) == 0 {
		return cache.rejectStable(record, "cached stable package artifact set is not exactly one MIR payload")
	}
	artifact, err := DecodeStableArtifact(record.Artifacts[0].Data, cache.maxArtifactBytes)
	if err != nil {
		return cache.rejectStable(record, err.Error())
	}
	if artifact.Package != expectedPackage {
		return cache.rejectStable(record, "cached stable package artifact belongs to a different package instance")
	}
	cache.mu.Lock()
	cache.verifiedHits++
	cache.mu.Unlock()
	return StableLookup{Status: LookupHit, Artifact: artifact}
}

// Stats returns current disk retention plus completed adapter outcomes.
func (cache *StableCache) Stats() (StableCacheStats, error) {
	if cache == nil || cache.store == nil {
		return StableCacheStats{}, fmt.Errorf("stable package artifact cache is not initialized")
	}
	store, err := cache.store.Stats()
	if err != nil {
		return StableCacheStats{}, err
	}
	cache.mu.Lock()
	stats := StableCacheStats{Store: store, VerifiedHits: cache.verifiedHits, Misses: cache.misses, Rejected: cache.rejected}
	cache.mu.Unlock()
	return stats, nil
}

func (cache *StableCache) rejectStable(record packagecache.Record, reason string) StableLookup {
	cache.store.DeleteIfMatch(record)
	cache.mu.Lock()
	cache.rejected++
	cache.mu.Unlock()
	return StableLookup{Status: LookupRejected, Reason: reason}
}

func (cache *StableCache) recordStableMiss() {
	cache.mu.Lock()
	cache.misses++
	cache.mu.Unlock()
}
