package packageartifact

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

var ErrStableCacheStore = errors.New("stable package artifact disk store failed")

// StableLookup is one cross-process cache result. A hit has passed the disk record checks, stable
// compilation codec, expected public-interface and package-instance checks, and exact expected
// contract-state body identity check.
type StableLookup struct {
	Status      string
	Compilation StableCompilationArtifact
	Reason      string
}

// StableCacheStats separates reusable stable artifacts from lower-level disk-record observations.
type StableCacheStats struct {
	Store        packagecache.DiskStats
	VerifiedHits uint64
	Misses       uint64
	Rejected     uint64
}

// StableCache connects exact-version stable package-compilation bytes to one bounded disposable
// DiskStore. It owns no default path, lifetime, or size policy; every policy input comes from its
// caller.
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
	return newStableCache(nil, root, limits, maxArtifactBytes)
}

// NewStableCacheInRoot creates the cache beneath an already-open filesystem boundary.
func NewStableCacheInRoot(boundary *os.Root, root string, limits packagecache.DiskLimits, maxArtifactBytes uint64) (*StableCache, error) {
	if boundary == nil {
		return nil, fmt.Errorf("stable package artifact filesystem boundary must be initialized")
	}
	return newStableCache(boundary, root, limits, maxArtifactBytes)
}

func newStableCache(boundary *os.Root, root string, limits packagecache.DiskLimits, maxArtifactBytes uint64) (*StableCache, error) {
	if maxArtifactBytes == 0 {
		return nil, fmt.Errorf("stable package artifact byte limit must be positive")
	}
	if maxArtifactBytes > limits.MaxRecordBytes {
		return nil, fmt.Errorf("stable package artifact byte limit cannot exceed the disk record byte limit")
	}
	var store *packagecache.DiskStore
	var err error
	if boundary == nil {
		store, err = packagecache.NewDiskStore(root, limits)
	} else {
		store, err = packagecache.NewDiskStoreInRoot(boundary, root, limits)
	}
	if err != nil {
		return nil, err
	}
	return &StableCache{store: store, maxArtifactBytes: maxArtifactBytes}, nil
}

// Close releases the persistent cache filesystem handle.
func (cache *StableCache) Close() error {
	if cache == nil || cache.store == nil {
		return nil
	}
	return cache.store.Close()
}

// Put verifies a process-local MIR artifact and independently described state body, binds them to
// one package coordinate, canonically encodes the result, and publishes one immutable record.
func (cache *StableCache) Put(key packagecache.Key, publicInterface moduleinterface.Hash, artifact Artifact, stateBody contractstatemir.StableStateBodyArtifact) error {
	if cache == nil || cache.store == nil || cache.maxArtifactBytes == 0 {
		return fmt.Errorf("stable package artifact cache is not initialized")
	}
	compilation, err := DescribeStableCompilationArtifact(artifact, stateBody, cache.maxArtifactBytes)
	if err != nil {
		return err
	}
	encoded, err := EncodeStableCompilationArtifact(compilation, cache.maxArtifactBytes)
	if err != nil {
		return err
	}
	if err := cache.store.Put(packagecache.Record{
		Key: key, PublicInterface: publicInterface,
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactCompilation, Data: encoded}},
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrStableCacheStore, err)
	}
	return nil
}

// Get returns only a stable compilation for the exact expected interface, package instance, and
// contract-state body hash. Every record-derived semantic rejection quarantines only the complete
// generation that produced it.
func (cache *StableCache) Get(key packagecache.Key, expectedInterface moduleinterface.Hash, expectedPackage packagegraph.InstanceID, expectedBody contractstatemir.StateBodyHash) StableLookup {
	if cache == nil || cache.store == nil || cache.maxArtifactBytes == 0 {
		return StableLookup{Status: LookupRejected, Reason: "stable package artifact cache is not initialized"}
	}
	if expectedBody == (contractstatemir.StateBodyHash{}) {
		cache.recordStableRejected()
		return StableLookup{Status: LookupRejected, Reason: "expected contract-state body hash must be set"}
	}
	record, found := cache.store.Get(key)
	if !found {
		cache.recordStableMiss()
		return StableLookup{Status: LookupMiss}
	}
	if record.PublicInterface != expectedInterface {
		return cache.rejectStable(record, "cached public interface does not match the expected package interface")
	}
	if len(record.Artifacts) != 1 || record.Artifacts[0].Kind != packagecache.ArtifactCompilation || len(record.Artifacts[0].Data) == 0 {
		return cache.rejectStable(record, "cached stable package artifact set is not exactly one compilation payload")
	}
	compilation, err := DecodeStableCompilationArtifact(record.Artifacts[0].Data, cache.maxArtifactBytes)
	if err != nil {
		return cache.rejectStable(record, err.Error())
	}
	if compilation.MIR.Package != expectedPackage {
		return cache.rejectStable(record, "cached stable package artifact belongs to a different package instance")
	}
	if compilation.StateBody.Hash != expectedBody {
		return cache.rejectStable(record, "cached contract-state body does not match the expected body hash")
	}
	cache.mu.Lock()
	cache.verifiedHits++
	cache.mu.Unlock()
	return StableLookup{Status: LookupHit, Compilation: compilation}
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
	cache.recordStableRejected()
	return StableLookup{Status: LookupRejected, Reason: reason}
}

func (cache *StableCache) recordStableRejected() {
	cache.mu.Lock()
	cache.rejected++
	cache.mu.Unlock()
}

func (cache *StableCache) recordStableMiss() {
	cache.mu.Lock()
	cache.misses++
	cache.mu.Unlock()
}
