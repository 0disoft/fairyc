package packageartifact

import (
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/packagecache"
)

const storageStableCacheSchema = "fairyc.storage-transfer-stable-cache/v0"

func (cache *GenericStableCache) schema() string {
	if cache.storageTransfers {
		return storageStableCacheSchema
	}
	return genericStableSchema
}

// NewStorageTransferStableCache selects private stable transfer artifacts with
// their own directory, envelope and storage-address domains. The bounded opened-
// root store still owns publication, retention and generation-safe quarantine.
func NewStorageTransferStableCache(root string, limits packagecache.DiskLimits, maxArtifactBytes uint64) (*GenericStableCache, error) {
	return newGenericStableCache(nil, root, limits, maxArtifactBytes, true)
}

func NewStorageTransferStableCacheInRoot(boundary *os.Root, root string, limits packagecache.DiskLimits, maxArtifactBytes uint64) (*GenericStableCache, error) {
	if boundary == nil {
		return nil, fmt.Errorf("storage transfer cache filesystem boundary must be initialized")
	}
	return newGenericStableCache(boundary, root, limits, maxArtifactBytes, true)
}
