package packageartifact

import "github.com/0disoft/fairyc/internal/packagecache"

const (
	StorageProcessCodecSchema    = "fairyc.package-mir-storage-process/v0"
	StorageProcessCodecVersion   = "fairyc-package-mir-storage-process/1"
	storageProcessIdentityPrefix = "fyproc-storage0-"
)

// NewStorageTransferProcessCodec creates an isolated private transfer codec.
// Its process identity is not authentication or current-source validation.
func NewStorageTransferProcessCodec(maxBytes uint64) (*ProcessCodec, error) {
	codec, err := NewProcessCodec(maxBytes)
	if err != nil {
		return nil, err
	}
	codec.storageTransfers = true
	return codec, nil
}

func (codec *ProcessCodec) contract() (string, string, func(Artifact) error) {
	if codec.storageTransfers {
		return StorageProcessCodecSchema, StorageProcessCodecVersion, VerifyStorageTransfers
	}
	return ProcessCodecSchema, ProcessCodecVersion, Verify
}

// NewStorageTransferCache reuses the bounded process-local LRU with transfer-only
// codec selection. Callers still own current key construction and whole-program
// reassembly; a cache hit alone does not authorize executable publication.
func NewStorageTransferCache(limits packagecache.Limits, maxArtifactBytes uint64) (*GenericCache, error) {
	return newGenericCache(limits, maxArtifactBytes, true)
}
