package atlasmap

import (
	"errors"
	"fmt"
)

// PublishBatch publishes immutable generations in order and collects once before
// returning. Unlike Publish, retention may temporarily exceed its bounds during
// the call. This is not a transaction: an error retains the completed prefix,
// whose checksums are returned, and still collects that prefix before returning.
func (store *MapStore) PublishBatch(partitions []Partition) ([]Checksum, error) {
	return store.publishBatch(partitions, store.Sweep)
}

func (store *MapStore) publishBatch(partitions []Partition, collect func() error) (checksums []Checksum, err error) {
	if store == nil || store.root == nil {
		return nil, fmt.Errorf(".fymap store is not initialized")
	}
	defer func() {
		if len(checksums) > 0 {
			if sweepErr := collect(); sweepErr != nil {
				err = errors.Join(err, fmt.Errorf("collect .fymap store after batch publication: %w", sweepErr))
			}
		}
	}()
	for _, partition := range partitions {
		checksum, publishErr := store.publishGeneration(partition, mapPublishHooks{})
		if publishErr != nil {
			return checksums, publishErr
		}
		checksums = append(checksums, checksum)
	}
	return checksums, nil
}
