package packagecache

import (
	"container/list"
	"errors"
	"fmt"
	"slices"
	"sync"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

const (
	ArtifactCompilation = "compilation"
	ArtifactTypedHIR    = "typed-hir"
	ArtifactMIR         = "mir"
	ArtifactObject      = "object"
)

var ErrRecordTooLarge = errors.New("package cache record exceeds the byte limit")

// Artifact is one compiler-private whole-package payload. Its bytes are disposable and exact-
// version only; this type does not create a stable HIR, MIR, or object exchange format.
type Artifact struct {
	Kind string
	Data []byte
}

// Record stores the cache key, the produced public interface, and ordered opaque artifacts.
type Record struct {
	Key             Key
	PublicInterface moduleinterface.Hash
	Artifacts       []Artifact
}

// Limits are mandatory caller policy. Stage 0 deliberately has no guessed universal defaults.
type Limits struct {
	MaxEntries int
	MaxBytes   uint64
}

// Stats reports logical retained bytes and deterministic process-local reuse behavior.
type Stats struct {
	Entries   int
	Bytes     uint64
	Hits      uint64
	Misses    uint64
	Evictions uint64
}

type storedRecord struct {
	record Record
	size   uint64
}

// MemoryStore is a concurrency-safe, byte- and entry-bounded LRU store. It is the first executable
// storage policy and intentionally does not imply persistent or remote cache compatibility.
type MemoryStore struct {
	mu      sync.Mutex
	limits  Limits
	entries map[Key]*list.Element
	recent  *list.List
	bytes   uint64
	hits    uint64
	misses  uint64
	evicted uint64
}

// NewMemoryStore rejects missing bounds instead of falling back to an unbounded cache.
func NewMemoryStore(limits Limits) (result *MemoryStore, err error) {
	defer func() { err = cacheError(ErrorConfiguration, err) }()
	if limits.MaxEntries <= 0 || limits.MaxBytes == 0 {
		return nil, fmt.Errorf("package cache limits require positive MaxEntries and MaxBytes")
	}
	return &MemoryStore{
		limits:  limits,
		entries: make(map[Key]*list.Element),
		recent:  list.New(),
	}, nil
}

// Put copies one valid record into the store and evicts least-recently-used records as needed.
// An oversized record is rejected without disturbing an existing record under the same key.
func (store *MemoryStore) Put(record Record) (err error) {
	defer func() { err = cacheError(ErrorRecord, err) }()
	size, err := validateRecord(record)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if size > store.limits.MaxBytes {
		return ErrRecordTooLarge
	}
	if existing, found := store.entries[record.Key]; found {
		store.remove(existing, false)
	}
	for len(store.entries) >= store.limits.MaxEntries || store.bytes+size > store.limits.MaxBytes {
		store.remove(store.recent.Back(), true)
	}
	entry := &storedRecord{record: cloneRecord(record), size: size}
	element := store.recent.PushFront(entry)
	store.entries[record.Key] = element
	store.bytes += size
	return nil
}

// Get returns a deep copy and promotes the record to most recently used.
func (store *MemoryStore) Get(key Key) (Record, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	element, found := store.entries[key]
	if !found {
		store.misses++
		return Record{}, false
	}
	store.hits++
	store.recent.MoveToFront(element)
	return cloneRecord(element.Value.(*storedRecord).record), true
}

// Delete removes a record without counting policy-driven eviction.
func (store *MemoryStore) Delete(key Key) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	element, found := store.entries[key]
	if !found {
		return false
	}
	store.remove(element, false)
	return true
}

// DeleteIfMatch removes key only when the complete currently stored record still equals expected.
// Cache consumers use this to quarantine a rejected payload without deleting a valid concurrent
// replacement that arrived after Get returned its deep copy.
func (store *MemoryStore) DeleteIfMatch(expected Record) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	element, found := store.entries[expected.Key]
	if !found || !recordsEqual(element.Value.(*storedRecord).record, expected) {
		return false
	}
	store.remove(element, false)
	return true
}

// Stats returns a consistent snapshot.
func (store *MemoryStore) Stats() Stats {
	store.mu.Lock()
	defer store.mu.Unlock()
	return Stats{
		Entries:   len(store.entries),
		Bytes:     store.bytes,
		Hits:      store.hits,
		Misses:    store.misses,
		Evictions: store.evicted,
	}
}

func (store *MemoryStore) remove(element *list.Element, eviction bool) {
	entry := element.Value.(*storedRecord)
	delete(store.entries, entry.record.Key)
	store.recent.Remove(element)
	store.bytes -= entry.size
	if eviction {
		store.evicted++
	}
}

func validateRecord(record Record) (uint64, error) {
	if isZero(record.Key[:]) {
		return 0, fmt.Errorf("package cache record key must be set")
	}
	if isZero(record.PublicInterface[:]) {
		return 0, fmt.Errorf("package cache record public interface must be set")
	}
	if len(record.Artifacts) == 0 {
		return 0, fmt.Errorf("package cache record has no artifacts")
	}
	var size uint64 = uint64(len(record.Key) + len(record.PublicInterface) + 8)
	for index, artifact := range record.Artifacts {
		if !validArtifactKind(artifact.Kind) || !utf8.ValidString(artifact.Kind) || len(artifact.Data) == 0 {
			return 0, fmt.Errorf("invalid package cache artifact %q", artifact.Kind)
		}
		if index > 0 && record.Artifacts[index-1].Kind >= artifact.Kind {
			return 0, fmt.Errorf("package cache artifacts must be sorted and unique by kind")
		}
		part := uint64(16 + len(artifact.Kind) + len(artifact.Data))
		if ^uint64(0)-size < part {
			return 0, ErrRecordTooLarge
		}
		size += part
	}
	return size, nil
}

func validArtifactKind(kind string) bool {
	switch kind {
	case ArtifactCompilation, ArtifactTypedHIR, ArtifactMIR, ArtifactObject:
		return true
	default:
		return false
	}
}

func cloneRecord(record Record) Record {
	result := Record{Key: record.Key, PublicInterface: record.PublicInterface, Artifacts: make([]Artifact, len(record.Artifacts))}
	for index, artifact := range record.Artifacts {
		result.Artifacts[index] = Artifact{Kind: artifact.Kind, Data: slices.Clone(artifact.Data)}
	}
	return result
}

func recordsEqual(left, right Record) bool {
	if left.Key != right.Key || left.PublicInterface != right.PublicInterface || len(left.Artifacts) != len(right.Artifacts) {
		return false
	}
	for index := range left.Artifacts {
		if left.Artifacts[index].Kind != right.Artifacts[index].Kind || !slices.Equal(left.Artifacts[index].Data, right.Artifacts[index].Data) {
			return false
		}
	}
	return true
}
