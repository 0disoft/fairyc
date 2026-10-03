package packageartifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const genericStableSchema = "fairyc.generic-stable-mir-cache/v0"

type genericStableEnvelope struct {
	Schema    string          `json:"schema"`
	Key       string          `json:"key"`
	Interface string          `json:"interface"`
	MIR       json.RawMessage `json:"mir"`
}

type GenericStableLookup struct {
	Status   string
	Artifact StableArtifact
	Reason   string
}

// GenericStableCache stores exact-version stable MIR, not process-codec bytes.
// Default constructors select generic-v0; transfer constructors select a separate
// namespace. Domain-separated addresses isolate both from ordinary compilation
// records. Hits still require current reassembly.
type GenericStableCache struct {
	store                  *packagecache.DiskStore
	maxArtifactBytes       uint64
	mu                     sync.Mutex
	hits, misses, rejected uint64
	storageTransfers       bool
}

func NewGenericStableCache(root string, limits packagecache.DiskLimits, maxArtifactBytes uint64) (*GenericStableCache, error) {
	return newGenericStableCache(nil, root, limits, maxArtifactBytes, false)
}

func NewGenericStableCacheInRoot(boundary *os.Root, root string, limits packagecache.DiskLimits, maxArtifactBytes uint64) (*GenericStableCache, error) {
	if boundary == nil {
		return nil, fmt.Errorf("generic stable cache filesystem boundary must be initialized")
	}
	return newGenericStableCache(boundary, root, limits, maxArtifactBytes, false)
}

func newGenericStableCache(boundary *os.Root, root string, limits packagecache.DiskLimits, maxArtifactBytes uint64, transfers bool) (*GenericStableCache, error) {
	if root == "" || maxArtifactBytes == 0 || maxArtifactBytes > limits.MaxRecordBytes {
		return nil, fmt.Errorf("generic stable cache requires a root and positive artifact bound within the record bound")
	}
	var store *packagecache.DiskStore
	var err error
	directory := "generic-v0"
	if transfers {
		directory = "storage-transfer-v0"
	}
	if boundary == nil {
		store, err = packagecache.NewNamespacedDiskStore(root, directory, limits)
	} else {
		store, err = packagecache.NewNamespacedDiskStoreInRoot(boundary, root, directory, limits)
	}
	if err != nil {
		return nil, err
	}
	return &GenericStableCache{store: store, maxArtifactBytes: maxArtifactBytes, storageTransfers: transfers}, nil
}

func (cache *GenericStableCache) Close() error {
	if cache == nil || cache.store == nil {
		return nil
	}
	return cache.store.Close()
}

// These are storage addresses only, never v4 compiler compatibility identities.
func stableDiskAddress(schema string, key packagecache.GenericKey) packagecache.Key {
	return sha256.Sum256([]byte(schema + ":address:" + key.String()))
}

func stableDiskInterface(schema string, hash moduleinterface.GenericHash) moduleinterface.Hash {
	return sha256.Sum256([]byte(schema + ":interface:" + hash.String()))
}

func (cache *GenericStableCache) Put(key packagecache.GenericKey, hash moduleinterface.GenericHash, artifact Artifact) error {
	if cache == nil || cache.store == nil {
		return fmt.Errorf("generic stable cache is not initialized")
	}
	if key == (packagecache.GenericKey{}) || hash == (moduleinterface.GenericHash{}) {
		return fmt.Errorf("generic stable cache key and interface must be set")
	}
	stable, err := describeStableArtifact(artifact, cache.maxArtifactBytes, cache.storageTransfers)
	if err != nil {
		return err
	}
	payload, err := encodeStableArtifact(stable, cache.maxArtifactBytes, cache.storageTransfers)
	if err != nil {
		return err
	}
	schema := cache.schema()
	encoded, err := json.Marshal(genericStableEnvelope{schema, key.String(), hash.String(), payload})
	if err != nil {
		return err
	}
	if uint64(len(encoded)) > cache.maxArtifactBytes {
		return ErrStableArtifactTooLarge
	}
	if err := cache.store.Put(packagecache.Record{
		Key: stableDiskAddress(schema, key), PublicInterface: stableDiskInterface(schema, hash),
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactMIR, Data: encoded}},
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrStableCacheStore, err)
	}
	return nil
}

func (cache *GenericStableCache) Get(key packagecache.GenericKey, hash moduleinterface.GenericHash, expectedPackage packagegraph.InstanceID, expectedCoordinate packagegraph.CoordinateID) GenericStableLookup {
	if cache == nil || cache.store == nil {
		return GenericStableLookup{Status: LookupRejected, Reason: "generic stable cache is not initialized"}
	}
	if key == (packagecache.GenericKey{}) || hash == (moduleinterface.GenericHash{}) || expectedPackage == (packagegraph.InstanceID{}) || expectedCoordinate == (packagegraph.CoordinateID{}) {
		return cache.genericResult(GenericStableLookup{Status: LookupRejected, Reason: "generic stable cache expectations must be set"})
	}
	schema := cache.schema()
	record, found := cache.store.Get(stableDiskAddress(schema, key))
	if !found {
		return cache.genericResult(GenericStableLookup{Status: LookupMiss})
	}
	reject := func(reason string) GenericStableLookup {
		cache.store.DeleteIfMatch(record)
		return cache.genericResult(GenericStableLookup{Status: LookupRejected, Reason: reason})
	}
	if record.PublicInterface != stableDiskInterface(schema, hash) || len(record.Artifacts) != 1 || record.Artifacts[0].Kind != packagecache.ArtifactMIR {
		return reject("generic stable cache record identity or payload kind differs")
	}
	encoded := record.Artifacts[0].Data
	if uint64(len(encoded)) > cache.maxArtifactBytes {
		return reject("generic stable cache payload exceeds artifact limit")
	}
	var envelope genericStableEnvelope
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		return reject(err.Error())
	}
	canonical, err := json.Marshal(envelope)
	if err != nil || !bytes.Equal(canonical, encoded) {
		return reject("generic stable cache envelope is not canonical")
	}
	if envelope.Schema != schema || envelope.Key != key.String() || envelope.Interface != hash.String() {
		return reject("generic stable cache envelope identity differs")
	}
	artifact, err := decodeStableArtifact(envelope.MIR, cache.maxArtifactBytes, cache.storageTransfers)
	if err != nil {
		return reject(err.Error())
	}
	if artifact.Package != expectedPackage || artifact.Coordinate != expectedCoordinate {
		return reject("generic stable cache package ownership differs")
	}
	return cache.genericResult(GenericStableLookup{Status: LookupHit, Artifact: artifact})
}

func (cache *GenericStableCache) genericResult(result GenericStableLookup) GenericStableLookup {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	switch result.Status {
	case LookupHit:
		cache.hits++
	case LookupMiss:
		cache.misses++
	case LookupRejected:
		cache.rejected++
	}
	return result
}

func (cache *GenericStableCache) Stats() (StableCacheStats, error) {
	if cache == nil || cache.store == nil {
		return StableCacheStats{}, fmt.Errorf("generic stable cache is not initialized")
	}
	store, err := cache.store.Stats()
	if err != nil {
		return StableCacheStats{}, err
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return StableCacheStats{Store: store, VerifiedHits: cache.hits, Misses: cache.misses, Rejected: cache.rejected}, nil
}
