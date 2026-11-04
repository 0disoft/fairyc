package packagecache

import "fmt"

const maxDiskBatchEntries = 256
const maxDiskBatchBytes uint64 = 16 << 20

// DiskBatch streams immutable records without retaining encoded bytes. It owns
// no filesystem handles and must be closed even on cancellation/error. Ordinary
// DiskStore.Put keeps its immediate collection guarantee. A batch guarantees
// collection on Close and at bounded entry/byte checkpoints; it has one caller.
type DiskBatch struct {
	store    *DiskStore
	entries  int
	bytes    uint64
	closed   bool
	closeErr error
	sweep    func() error
}

func (store *DiskStore) BeginBatch() *DiskBatch {
	batch := &DiskBatch{store: store}
	if store != nil {
		batch.sweep = store.Sweep
	}
	return batch
}

func (batch *DiskBatch) Put(record Record) (err error) {
	defer func() { err = cacheError(ErrorIO, err) }()
	if batch == nil || batch.store == nil || batch.closed {
		return fmt.Errorf("package cache batch is not open")
	}
	entryLimit := min(maxDiskBatchEntries, batch.store.limits.MaxEntries)
	byteLimit := min(maxDiskBatchBytes, batch.store.limits.MaxBytes)
	return batch.store.put(record, diskPutHooks{
		beforeEncodedPublish: func(size int) error {
			if batch.entries > 0 && (batch.entries >= entryLimit || uint64(size) > byteLimit-min(byteLimit, batch.bytes)) {
				return batch.flush()
			}
			return nil
		},
		afterEncodedPublish: func(size int) error {
			batch.entries++
			batch.bytes += uint64(size)
			if batch.entries >= entryLimit || batch.bytes >= byteLimit {
				return batch.flush()
			}
			return nil
		},
	})
}

func (batch *DiskBatch) flush() error {
	if batch.entries == 0 {
		return nil
	}
	if err := batch.sweep(); err != nil {
		return err
	}
	batch.entries, batch.bytes = 0, 0
	return nil
}

func (batch *DiskBatch) Close() error {
	if batch == nil {
		return nil
	}
	if !batch.closed {
		batch.closeErr = batch.flush()
		batch.closed = true
	}
	return batch.closeErr
}
