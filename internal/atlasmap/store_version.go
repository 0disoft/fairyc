package atlasmap

import (
	"fmt"
	"os"
)

type mapStoreVersion uint8

const (
	mapStoreV1 mapStoreVersion = iota
	mapStoreV2
	mapStoreV3
)

// NewMapStoreV3 opens only aggregate-aware generations in fymap-v3.
func NewMapStoreV3(root string, limits MapStoreLimits) (*MapStore, error) {
	return newMapStore(root, limits, mapStoreV3)
}

// NewMapStoreV3InRoot retains the caller's opened-root boundary for v3 I/O.
func NewMapStoreV3InRoot(boundary *os.Root, root string, limits MapStoreLimits) (*MapStore, error) {
	return newMapStoreInRoot(boundary, root, limits, mapStoreV3)
}

func (version mapStoreVersion) layoutDirectory() (string, error) {
	switch version {
	case mapStoreV1:
		return mapStoreLayoutDirectory, nil
	case mapStoreV2:
		return mapStoreLayoutDirectoryV2, nil
	case mapStoreV3:
		return mapStoreLayoutDirectoryV3, nil
	default:
		return "", fmt.Errorf("unsupported .fymap store version")
	}
}

func (store *MapStore) transportVersion() (schema, encoding, logical string) {
	switch store.version {
	case mapStoreV2:
		return TransportSchemaV2, TransportEncodingV2, LogicalSchemaV2
	case mapStoreV3:
		return TransportSchemaV3, TransportEncodingV3, LogicalSchemaV3
	}
	return TransportSchema, TransportEncoding, LogicalSchema
}
