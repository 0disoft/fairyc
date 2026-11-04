package packageartifact

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

// StableCacheBatch borrows its cache's filesystem handles. Close only collects
// pending records, leaving the cache usable by subsequent checks.
type StableCacheBatch struct {
	cache *StableCache
	disk  *packagecache.DiskBatch
}

func (cache *StableCache) BeginBatch() *StableCacheBatch {
	batch := &StableCacheBatch{cache: cache}
	if cache != nil && cache.store != nil {
		batch.disk = cache.store.BeginBatch()
	}
	return batch
}

func (batch *StableCacheBatch) Put(key packagecache.Key, publicInterface moduleinterface.Hash, artifact Artifact, stateBody contractstatemir.StableStateBodyArtifact) error {
	if batch == nil || batch.cache == nil || batch.disk == nil {
		return fmt.Errorf("stable package artifact batch is not initialized")
	}
	return batch.cache.put(key, publicInterface, artifact, stateBody, batch.disk.Put)
}

func (batch *StableCacheBatch) Close() error {
	if batch == nil || batch.disk == nil {
		return nil
	}
	if err := batch.disk.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrStableCacheStore, err)
	}
	return nil
}

// GenericStableCacheBatch applies the same publication/retention lifetime to
// generic and storage-transfer namespaces, preserving their address domains.
type GenericStableCacheBatch struct {
	cache *GenericStableCache
	disk  *packagecache.DiskBatch
}

func (cache *GenericStableCache) BeginBatch() *GenericStableCacheBatch {
	batch := &GenericStableCacheBatch{cache: cache}
	if cache != nil && cache.store != nil {
		batch.disk = cache.store.BeginBatch()
	}
	return batch
}

func (batch *GenericStableCacheBatch) Put(key packagecache.GenericKey, hash moduleinterface.GenericHash, artifact Artifact) error {
	if batch == nil || batch.cache == nil || batch.disk == nil {
		return fmt.Errorf("generic stable package artifact batch is not initialized")
	}
	return batch.cache.put(key, hash, artifact, batch.disk.Put)
}

func (batch *GenericStableCacheBatch) Close() error {
	if batch == nil || batch.disk == nil {
		return nil
	}
	if err := batch.disk.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrStableCacheStore, err)
	}
	return nil
}
