package contractstate

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"
)

// PersistencePort is the minimum durable authority required by the transaction
// oracle. A production adapter must implement snapshot plus compare-and-publish
// atomically; a generic callback transaction is intentionally not sufficient.
type PersistencePort interface {
	Snapshot(expectedLayout LayoutHash) (PersistentSnapshot, error)
	CompareAndPublish(expectedLayout LayoutHash, expectedVersion uint64, writes []PersistentWrite) (PersistentReceipt, error)
	PublishLayout(expectedLayout LayoutHash, expectedVersion uint64, next Layout) (PersistentReceipt, error)
}

// PersistentSnapshot is one complete immutable observation used to start a
// transaction. Entries are sorted by binary physical key and deep-copied.
type PersistentSnapshot struct {
	LayoutHash LayoutHash
	Version    uint64
	Entries    []Entry
}

// PersistentWrite is one canonical durable overlay mutation. Deleted writes
// carry no value. Duplicate keys in one publish request are rejected.
type PersistentWrite struct {
	Key     Key
	Value   []byte
	Deleted bool
}

// PersistentReceipt identifies the durable state after one successful atomic
// publication. A no-op write publish preserves Version and reports zero writes.
type PersistentReceipt struct {
	LayoutHash LayoutHash
	Version    uint64
	Writes     int
}

// PersistenceFaultPoint names a pre-publication boundary in the deterministic
// reference adapter. Injected failures at these points must leave durable state
// byte-for-byte unchanged.
type PersistenceFaultPoint string

const (
	FaultBeforeCompare PersistenceFaultPoint = "before-compare"
	FaultBeforePublish PersistenceFaultPoint = "before-publish"
)

// PersistenceFault injects one deterministic backend failure. Returning nil
// continues publication. Production adapters do not need to expose fault hooks.
type PersistenceFault func(PersistenceFaultPoint) error

// ReferencePersistence is a deterministic in-memory reference adapter for the
// PersistencePort contract. It models durability boundaries and fault behavior;
// it is not a production database.
type ReferencePersistence struct {
	mu      sync.RWMutex
	layout  Layout
	version uint64
	values  map[Key][]byte
	fault   PersistenceFault
}

func NewReferencePersistence(layout Layout, fault PersistenceFault) (*ReferencePersistence, error) {
	if err := validateLayout(layout); err != nil {
		return nil, err
	}
	return &ReferencePersistence{
		layout: cloneLayout(layout),
		values: make(map[Key][]byte),
		fault:  fault,
	}, nil
}

func (store *ReferencePersistence) Snapshot(expectedLayout LayoutHash) (PersistentSnapshot, error) {
	if store == nil {
		return PersistentSnapshot{}, errors.New("contract state persistence is nil")
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	if expectedLayout != store.layout.hash {
		return PersistentSnapshot{}, fmt.Errorf("%w: snapshot %s, store %s", ErrLayoutMismatch, expectedLayout, store.layout.hash)
	}
	return PersistentSnapshot{
		LayoutHash: store.layout.hash,
		Version:    store.version,
		Entries:    sortedPersistentEntries(store.values),
	}, nil
}

func (store *ReferencePersistence) CompareAndPublish(expectedLayout LayoutHash, expectedVersion uint64, writes []PersistentWrite) (PersistentReceipt, error) {
	if store == nil {
		return PersistentReceipt{}, errors.New("contract state persistence is nil")
	}
	canonical, err := canonicalPersistentWrites(writes)
	if err != nil {
		return PersistentReceipt{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.inject(FaultBeforeCompare); err != nil {
		return PersistentReceipt{}, err
	}
	if expectedLayout != store.layout.hash {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, fmt.Errorf("%w: transaction %s, store %s", ErrLayoutMismatch, expectedLayout, store.layout.hash)
	}
	if expectedVersion != store.version {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, fmt.Errorf("%w: transaction version %d, store version %d", ErrConflict, expectedVersion, store.version)
	}
	if len(canonical) == 0 {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, nil
	}
	if err := store.inject(FaultBeforePublish); err != nil {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, err
	}

	next := clonePersistentValues(store.values)
	for _, write := range canonical {
		if write.Deleted {
			delete(next, write.Key)
			continue
		}
		next[write.Key] = slices.Clone(write.Value)
	}
	store.values = next
	store.version++
	return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version, Writes: len(canonical)}, nil
}

func (store *ReferencePersistence) PublishLayout(expectedLayout LayoutHash, expectedVersion uint64, next Layout) (PersistentReceipt, error) {
	if store == nil {
		return PersistentReceipt{}, errors.New("contract state persistence is nil")
	}
	if err := validateLayout(next); err != nil {
		return PersistentReceipt{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.inject(FaultBeforeCompare); err != nil {
		return PersistentReceipt{}, err
	}
	if expectedLayout != store.layout.hash {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, fmt.Errorf("%w: layout publisher %s, store %s", ErrLayoutMismatch, expectedLayout, store.layout.hash)
	}
	if expectedVersion != store.version {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, fmt.Errorf("%w: layout version %d, store version %d", ErrConflict, expectedVersion, store.version)
	}
	if next.hash == store.layout.hash {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, nil
	}
	if err := next.Extends(store.layout); err != nil {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, err
	}
	if err := store.inject(FaultBeforePublish); err != nil {
		return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, err
	}
	store.layout = cloneLayout(next)
	store.version++
	return PersistentReceipt{LayoutHash: store.layout.hash, Version: store.version}, nil
}

func (store *ReferencePersistence) inject(point PersistenceFaultPoint) error {
	if store.fault == nil {
		return nil
	}
	return store.fault(point)
}

func canonicalPersistentWrites(writes []PersistentWrite) ([]PersistentWrite, error) {
	result := make([]PersistentWrite, len(writes))
	for index, write := range writes {
		if write.Deleted && len(write.Value) != 0 {
			return nil, fmt.Errorf("deleted persistent write for %s carries a value", write.Key)
		}
		result[index] = PersistentWrite{Key: write.Key, Value: slices.Clone(write.Value), Deleted: write.Deleted}
	}
	sort.Slice(result, func(left, right int) bool {
		return slices.Compare(result[left].Key[:], result[right].Key[:]) < 0
	})
	for index := 1; index < len(result); index++ {
		if result[index-1].Key == result[index].Key {
			return nil, fmt.Errorf("persistent publish contains duplicate key %s", result[index].Key)
		}
	}
	return result, nil
}

func sortedPersistentEntries(values map[Key][]byte) []Entry {
	keys := make([]Key, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		return slices.Compare(keys[left][:], keys[right][:]) < 0
	})
	entries := make([]Entry, len(keys))
	for index, key := range keys {
		entries[index] = Entry{Key: key, Value: slices.Clone(values[key])}
	}
	return entries
}

func clonePersistentValues(values map[Key][]byte) map[Key][]byte {
	result := make(map[Key][]byte, len(values))
	for key, value := range values {
		result[key] = slices.Clone(value)
	}
	return result
}
