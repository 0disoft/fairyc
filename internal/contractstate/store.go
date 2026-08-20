package contractstate

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"
)

var (
	ErrClosed           = errors.New("contract state transaction is closed")
	ErrConflict         = errors.New("contract state transaction conflicts with a committed change")
	ErrLayoutMismatch   = errors.New("contract state layout does not match the store")
	ErrReadOnly         = errors.New("contract state transaction is read-only")
	ErrManagedLifecycle = errors.New("contract state transaction lifecycle is owned by Execute or View")
	ErrWriteLimit       = errors.New("contract state transaction exceeds the write-count limit")
	ErrWriteBytesLimit  = errors.New("contract state transaction exceeds the write-byte limit")
	ErrValueTooLarge    = errors.New("contract state value exceeds the per-value byte limit")
	ErrMapKeyTooLarge   = errors.New("contract state map key exceeds the encoded-key byte limit")
)

// Limits are mandatory execution policy. There are no hidden or guessed
// defaults because chain targets need explicit resource accounting.
type Limits struct {
	MaxWrites      int
	MaxWriteBytes  uint64
	MaxValueBytes  int
	MaxMapKeyBytes int
}

func (limits Limits) validate() error {
	if limits.MaxWrites <= 0 {
		return fmt.Errorf("contract state MaxWrites must be positive")
	}
	if limits.MaxWriteBytes == 0 {
		return fmt.Errorf("contract state MaxWriteBytes must be positive")
	}
	if limits.MaxValueBytes <= 0 {
		return fmt.Errorf("contract state MaxValueBytes must be positive")
	}
	if limits.MaxMapKeyBytes <= 0 {
		return fmt.Errorf("contract state MaxMapKeyBytes must be positive")
	}
	return nil
}

// Entry is one sorted deep-copied MemoryStore observation.
type Entry struct {
	Key   Key
	Value []byte
}

// MemoryStore is a process-local executable oracle. It provides snapshot reads
// and store-wide optimistic serialization; it is not a production database or
// chain state tree.
type MemoryStore struct {
	mu      sync.RWMutex
	layout  Layout
	version uint64
	values  map[Key][]byte
}

func NewMemoryStore(layout Layout) (*MemoryStore, error) {
	if err := validateLayout(layout); err != nil {
		return nil, err
	}
	return &MemoryStore{
		layout: cloneLayout(layout),
		values: make(map[Key][]byte),
	}, nil
}

func (store *MemoryStore) Layout() Layout {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return cloneLayout(store.layout)
}

func (store *MemoryStore) Version() uint64 {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.version
}

// Upgrade accepts the identical layout or one append-only extension. Existing
// keys remain stable; open transactions fail their later commit.
func (store *MemoryStore) Upgrade(next Layout) error {
	if err := validateLayout(next); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if next.hash == store.layout.hash {
		return nil
	}
	if err := next.Extends(store.layout); err != nil {
		return err
	}
	store.layout = cloneLayout(next)
	store.version++
	return nil
}

// Dump returns all committed entries sorted by their binary keys.
func (store *MemoryStore) Dump() []Entry {
	store.mu.RLock()
	defer store.mu.RUnlock()
	keys := make([]Key, 0, len(store.values))
	for key := range store.values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		return slices.Compare(keys[left][:], keys[right][:]) < 0
	})
	entries := make([]Entry, len(keys))
	for index, key := range keys {
		entries[index] = Entry{Key: key, Value: slices.Clone(store.values[key])}
	}
	return entries
}

func (store *MemoryStore) snapshot(layout Layout) (uint64, map[Key][]byte, error) {
	if store == nil {
		return 0, nil, fmt.Errorf("contract state store is nil")
	}
	if err := validateLayout(layout); err != nil {
		return 0, nil, err
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	if layout.hash != store.layout.hash {
		return 0, nil, fmt.Errorf("%w: transaction %s, store %s", ErrLayoutMismatch, layout.hash, store.layout.hash)
	}
	values := make(map[Key][]byte, len(store.values))
	for key, value := range store.values {
		values[key] = slices.Clone(value)
	}
	return store.version, values, nil
}

func (store *MemoryStore) commit(layoutHash LayoutHash, baseVersion uint64, writes map[Key]pendingWrite) (uint64, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if layoutHash != store.layout.hash {
		return store.version, fmt.Errorf("%w: transaction %s, store %s", ErrLayoutMismatch, layoutHash, store.layout.hash)
	}
	if baseVersion != store.version {
		return store.version, fmt.Errorf("%w: transaction version %d, store version %d", ErrConflict, baseVersion, store.version)
	}
	if len(writes) == 0 {
		return store.version, nil
	}
	keys := make([]Key, 0, len(writes))
	for key := range writes {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		return slices.Compare(keys[left][:], keys[right][:]) < 0
	})
	for _, key := range keys {
		write := writes[key]
		if write.deleted {
			delete(store.values, key)
			continue
		}
		store.values[key] = slices.Clone(write.value)
	}
	store.version++
	return store.version, nil
}

func cloneLayout(layout Layout) Layout {
	return Layout{contractUID: layout.contractUID, fields: slices.Clone(layout.fields), hash: layout.hash}
}
