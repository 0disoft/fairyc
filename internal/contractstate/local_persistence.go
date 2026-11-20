package contractstate

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/0disoft/fairyc/internal/durablefile"
)

var (
	ErrPersistenceBusy   = errors.New("local state persistence is busy")
	ErrPersistenceClosed = errors.New("local state persistence is closed")
)

const (
	localStateName       = "state-v1.bin"
	localLockName        = "state-v1.lock"
	maxLocalStateBytes   = 16 << 20
	maxLocalStateEntries = 65_536
)

// LocalPersistenceLimits bounds both decoding and complete-state publication.
// These are separate from transaction-overlay limits and have no defaults.
type LocalPersistenceLimits struct {
	MaxStateBytes int64
	MaxEntries    int
	MaxValueBytes int
}

func (limits LocalPersistenceLimits) validate() error {
	if limits.MaxStateBytes <= 0 || limits.MaxStateBytes > maxLocalStateBytes ||
		limits.MaxEntries <= 0 || limits.MaxEntries > maxLocalStateEntries ||
		limits.MaxValueBytes <= 0 || int64(limits.MaxValueBytes) > limits.MaxStateBytes {
		return fmt.Errorf("local state limits require 1..16MiB bytes, 1..65536 entries and a positive bounded value size")
	}
	return nil
}

// LocalPersistence implements the port for a trusted, existing local directory.
// Each operation takes a nonblocking OS file lock, rereads the exact state and
// atomically replaces it after syncing a complete staging file. Success means
// restart-visible publication, not certified power-loss durability. No fallible
// flush is performed after rename, so known publication is never reported as a
// generic failure. Network filesystems and hostile directory owners are excluded.
type LocalPersistence struct {
	mu            sync.Mutex
	root          *os.Root
	lock          *os.File
	limits        LocalPersistenceLimits
	closed        bool
	beforePublish func() error // private pre-publication fault control for tests
}

var _ PersistencePort = (*LocalPersistence)(nil)

// CreateLocalPersistence exclusively creates a new store. Existing state or lock
// files are never overwritten, including remnants of an interrupted creation.
func CreateLocalPersistence(directory string, initial Layout, limits LocalPersistenceLimits) (*LocalPersistence, error) {
	if err := limits.validate(); err != nil {
		return nil, err
	}
	record, err := encodeLocalState(localState{layout: initial}, limits)
	if err != nil {
		return nil, err
	}
	return openLocalPersistence(directory, initial, limits, record)
}

// OpenLocalPersistence opens an existing store. Missing authority files fail
// closed instead of silently resetting previously committed data.
func OpenLocalPersistence(directory string, initial Layout, limits LocalPersistenceLimits) (*LocalPersistence, error) {
	return openLocalPersistence(directory, initial, limits, nil)
}

func openLocalPersistence(directory string, initial Layout, limits LocalPersistenceLimits, initialRecord []byte) (*LocalPersistence, error) {
	if err := validateLayout(initial); err != nil {
		return nil, err
	}
	if err := limits.validate(); err != nil {
		return nil, err
	}
	root, err := openLocalRoot(directory)
	if err != nil {
		return nil, err
	}
	store := &LocalPersistence{root: root, limits: limits}
	create := initialRecord != nil
	flags := os.O_RDWR
	if create {
		if _, err := root.Lstat(localStateName); !errors.Is(err, os.ErrNotExist) {
			_ = root.Close()
			if err == nil {
				return nil, os.ErrExist
			}
			return nil, err
		}
		flags |= os.O_CREATE | os.O_EXCL
	}
	store.lock, err = store.openRegular(localLockName, flags)
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	err = store.withLock(func() error {
		current, err := store.load()
		if errors.Is(err, os.ErrNotExist) && create {
			return store.publishRecord(initialRecord)
		}
		if err != nil {
			return err
		}
		if create {
			return os.ErrExist
		}
		if current.layout.Hash() != initial.Hash() {
			return ErrLayoutMismatch
		}
		return nil
	})
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func openLocalRoot(directory string) (*os.Root, error) {
	if directory == "" {
		return nil, fmt.Errorf("local state directory is required")
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("local state root must be a real directory")
	}
	return os.OpenRoot(absolute)
}

func (store *LocalPersistence) Close() error {
	if store == nil {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed {
		return nil
	}
	store.closed = true
	return errors.Join(store.lock.Close(), store.root.Close())
}

func (store *LocalPersistence) withLock(body func() error) error {
	if store == nil {
		return ErrPersistenceClosed
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed {
		return ErrPersistenceClosed
	}
	pathInfo, err := store.root.Lstat(localLockName)
	if err != nil {
		return err
	}
	fileInfo, err := store.lock.Stat()
	if err != nil {
		return err
	}
	if !pathInfo.Mode().IsRegular() || !os.SameFile(pathInfo, fileInfo) {
		return fmt.Errorf("local state lock identity changed")
	}
	if err := lockLocalState(store.lock); err != nil {
		return err
	}
	defer unlockLocalState(store.lock)
	return body()
}

func (store *LocalPersistence) Snapshot(hash LayoutHash) (snapshot PersistentSnapshot, err error) {
	err = store.withLock(func() error {
		current, loadErr := store.load()
		if loadErr != nil {
			return loadErr
		}
		if current.layout.Hash() != hash {
			return ErrLayoutMismatch
		}
		snapshot = PersistentSnapshot{LayoutHash: hash, Version: current.version, Entries: current.entries}
		return nil
	})
	return snapshot, err
}

func (store *LocalPersistence) CompareAndPublish(hash LayoutHash, version uint64, writes []PersistentWrite) (receipt PersistentReceipt, err error) {
	if store == nil {
		return receipt, ErrPersistenceClosed
	}
	canonical, err := prepareLocalWrites(writes, store.limits)
	if err != nil {
		return receipt, err
	}
	err = store.withLock(func() error {
		current, err := store.load()
		if err != nil {
			return err
		}
		receipt = PersistentReceipt{LayoutHash: current.layout.Hash(), Version: current.version}
		if hash != current.layout.Hash() {
			return ErrLayoutMismatch
		}
		if version != current.version {
			return ErrConflict
		}
		if len(canonical) == 0 {
			return nil
		}
		if version == math.MaxUint64 {
			return fmt.Errorf("local state version exhausted")
		}
		current.entries = mergeLocalEntries(current.entries, canonical)
		current.version++
		if err := store.publish(current); err != nil {
			return err
		}
		receipt.Version, receipt.Writes = current.version, len(canonical)
		return nil
	})
	return receipt, err
}

func prepareLocalWrites(writes []PersistentWrite, limits LocalPersistenceLimits) ([]PersistentWrite, error) {
	if len(writes) > limits.MaxEntries {
		return nil, fmt.Errorf("local state mutation count limit exceeded")
	}
	var total int64
	for _, write := range writes {
		if len(write.Value) > limits.MaxValueBytes {
			return nil, ErrValueTooLarge
		}
		total += int64(len(write.Value))
		if total > limits.MaxStateBytes {
			return nil, fmt.Errorf("local state mutation bytes exceed file size limit")
		}
	}
	return canonicalPersistentWrites(writes)
}

// Both inputs have sorted unique keys. Their value bytes are owned by this
// publication attempt, so the result can borrow them until encoding completes.
func mergeLocalEntries(entries []Entry, writes []PersistentWrite) []Entry {
	merged := make([]Entry, 0, len(entries)+len(writes))
	for len(entries) > 0 && len(writes) > 0 {
		entry, write := entries[0], writes[0]
		order := slices.Compare(entry.Key[:], write.Key[:])
		if order < 0 {
			merged = append(merged, entry)
			entries = entries[1:]
			continue
		}
		if order == 0 {
			entries = entries[1:]
		}
		if !write.Deleted {
			merged = append(merged, Entry{Key: write.Key, Value: write.Value})
		}
		writes = writes[1:]
	}
	merged = append(merged, entries...)
	for _, write := range writes {
		if !write.Deleted {
			merged = append(merged, Entry{Key: write.Key, Value: write.Value})
		}
	}
	return merged
}

func (store *LocalPersistence) PublishLayout(hash LayoutHash, version uint64, next Layout) (receipt PersistentReceipt, err error) {
	if err := validateLayout(next); err != nil {
		return receipt, err
	}
	err = store.withLock(func() error {
		current, err := store.load()
		if err != nil {
			return err
		}
		receipt = PersistentReceipt{LayoutHash: current.layout.Hash(), Version: current.version}
		if hash != current.layout.Hash() {
			return ErrLayoutMismatch
		}
		if version != current.version {
			return ErrConflict
		}
		if hash == next.Hash() {
			return nil
		}
		if err := next.Extends(current.layout); err != nil {
			return err
		}
		if version == math.MaxUint64 {
			return fmt.Errorf("local state version exhausted")
		}
		current.layout, current.version = cloneLayout(next), version+1
		if err := store.publish(current); err != nil {
			return err
		}
		receipt.LayoutHash, receipt.Version = next.Hash(), current.version
		return nil
	})
	return receipt, err
}

func (store *LocalPersistence) openRegular(name string, flags int) (*os.File, error) {
	before, err := store.root.Lstat(name)
	if err != nil && !(errors.Is(err, os.ErrNotExist) && flags&os.O_CREATE != 0) {
		return nil, err
	}
	if err == nil && !before.Mode().IsRegular() {
		return nil, fmt.Errorf("local state %s must be a regular file", name)
	}
	file, err := store.root.OpenFile(name, flags, 0o600)
	if err != nil {
		return nil, err
	}
	opened, statErr := file.Stat()
	after, pathErr := store.root.Lstat(name)
	if statErr != nil || pathErr != nil || !opened.Mode().IsRegular() || !after.Mode().IsRegular() || !os.SameFile(opened, after) || before != nil && !os.SameFile(before, opened) {
		_ = file.Close()
		return nil, fmt.Errorf("local state %s identity changed or is not regular", name)
	}
	return file, nil
}

func (store *LocalPersistence) load() (localState, error) {
	state, _, err := store.readRecord()
	return state, err
}

func (store *LocalPersistence) readRecord() (localState, []byte, error) {
	file, err := store.openRegular(localStateName, os.O_RDONLY)
	if err != nil {
		return localState{}, nil, err
	}
	defer file.Close()
	encoded, err := readLocalStateBytes(file, store.limits)
	if err != nil {
		return localState{}, nil, err
	}
	state, err := decodeLocalState(encoded, store.limits)
	if err != nil {
		return localState{}, nil, err
	}
	return state, encoded, nil
}

func readLocalStateBytes(file *os.File, limits LocalPersistenceLimits) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > limits.MaxStateBytes {
		return nil, fmt.Errorf("local state exceeds file size limit")
	}
	return io.ReadAll(io.LimitReader(file, limits.MaxStateBytes+1))
}

func (store *LocalPersistence) publish(current localState) error {
	encoded, err := encodeLocalState(current, store.limits)
	if err != nil {
		return err
	}
	return store.publishRecord(encoded)
}

func (store *LocalPersistence) publishRecord(encoded []byte) error {
	temporary, err := stageLocalRecord(store.root, encoded)
	if err != nil {
		return err
	}
	defer store.root.Remove(temporary)
	if store.beforePublish != nil {
		if err := store.beforePublish(); err != nil {
			return err
		}
	}
	if info, err := store.root.Lstat(localStateName); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("local state destination is not regular")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return store.root.Rename(temporary, localStateName)
}

func stageLocalRecord(root *os.Root, encoded []byte) (string, error) {
	file, name, err := durablefile.CreateTemporary(root, ".")
	if err != nil {
		return "", err
	}
	_, writeErr := file.Write(encoded)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if err := errors.Join(writeErr, file.Close()); err != nil {
		_ = root.Remove(name)
		return "", err
	}
	return name, nil
}

// Keep snapshots independent from future callers even when entries are empty.
func cloneLocalEntries(entries []Entry) []Entry {
	result := make([]Entry, len(entries))
	for i, entry := range entries {
		result[i] = Entry{Key: entry.Key, Value: slices.Clone(entry.Value)}
	}
	return result
}
