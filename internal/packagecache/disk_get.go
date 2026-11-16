package packagecache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

var errDiskGenerationCorrupt = errors.New("package cache generation is corrupt")

// Any other verified generation count is a miss, never a selection by order.
const diskHitGenerationCount = 1

// DiskGetOptions supplies per-lookup I/O and Go's cancellation signal. Nil fields
// retain the opened root and Background. Callbacks use root-relative paths and
// must preserve the corresponding host operation's semantics. Only the built-in
// reader's verified corruption authorizes removal; other failures are safe misses.
// Cancellation cannot preempt a host syscall.
type DiskGetOptions struct {
	Context        context.Context
	ReadDirectory  func(string, *directoryscan.Budget) ([]os.DirEntry, error)
	ReadGeneration func(context.Context, string, string, Key) (Record, error)
	Remove         func(string) error
}

// GetWithOptions keeps Get's safe-miss result while allowing deterministic I/O
// failures and cancellation without changing the store's shared configuration.
func (store *DiskStore) GetWithOptions(key Key, options ...DiskGetOptions) (Record, bool) {
	if store == nil || store.root == nil || isZero(key[:]) {
		return Record{}, false
	}
	miss := func() (Record, bool) {
		store.recordMiss()
		return Record{}, false
	}
	if len(options) > 1 {
		return miss()
	}
	policy := store.diskGetPolicy(options)
	ctx := policy.Context
	if ctx.Err() != nil {
		return miss()
	}
	keyDirectory := store.keyDirectoryName(key)
	budget := store.newScanBudget()
	budget.Context = ctx
	entries, err := policy.ReadDirectory(keyDirectory, budget)
	if err != nil || ctx.Err() != nil {
		return miss()
	}
	names := diskGenerationCandidates(entries)
	candidate, validCount, complete := store.readDiskGenerationCandidates(keyDirectory, key, names, policy)
	if !complete {
		return miss()
	}
	if validCount == diskHitGenerationCount {
		store.recordHit()
		return candidate, true
	}
	if validCount > diskHitGenerationCount {
		store.recordRejectedBy(uint64(validCount))
		if !removeDiskGenerationCandidates(keyDirectory, names, policy) {
			return miss()
		}
	}
	removeEmptyKeyDirectoryUsing(keyDirectory, ctx, store.root.Lstat, policy.Remove)
	return miss()
}

func (store *DiskStore) diskGetPolicy(options []DiskGetOptions) DiskGetOptions {
	var policy DiskGetOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.ReadDirectory == nil {
		policy.ReadDirectory = func(name string, budget *directoryscan.Budget) ([]os.DirEntry, error) {
			return readCacheDirectoryWithBudget(store.root, name, budget)
		}
	}
	if policy.ReadGeneration == nil {
		policy.ReadGeneration = func(ctx context.Context, path, name string, key Key) (Record, error) {
			var sizeHint int64
			if info, err := store.root.Lstat(path); err == nil {
				sizeHint = info.Size()
			}
			if err := ctx.Err(); err != nil {
				return Record{}, err
			}
			return store.readGeneration(path, name, key, sizeHint)
		}
	}
	if policy.Remove == nil {
		policy.Remove = store.root.Remove
	}
	return policy
}

// Directory callbacks may share immutable snapshots. Retain and sort only
// generation names in our own array; unowned entries never enter the sort.
func diskGenerationCandidates(entries []os.DirEntry) []string {
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if validGenerationName(name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// Incomplete observations cannot authorize a hit or conflict cleanup.
func (store *DiskStore) readDiskGenerationCandidates(directory string, key Key, names []string, policy DiskGetOptions) (Record, int, bool) {
	ctx := policy.Context
	var candidate Record
	validCount := 0
	for _, name := range names {
		if ctx.Err() != nil {
			return Record{}, 0, false
		}
		path := filepath.Join(directory, name)
		record, readErr := policy.ReadGeneration(ctx, path, name, key)
		if ctx.Err() != nil {
			return Record{}, 0, false
		}
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue // A concurrent collector removed this generation.
			}
			if !errors.Is(readErr, errDiskGenerationCorrupt) {
				return Record{}, 0, false // An unread sibling may hide a producer conflict.
			}
			store.recordRejected()
			_ = policy.Remove(path)
			continue
		}
		validCount++
		if validCount == diskHitGenerationCount {
			candidate = record
		} else {
			// Validate/count later generations without retaining extra payloads.
			candidate = Record{}
		}
	}
	if ctx.Err() != nil {
		return Record{}, 0, false
	}
	return candidate, validCount, true
}

func removeDiskGenerationCandidates(directory string, names []string, policy DiskGetOptions) bool {
	for _, name := range names {
		if policy.Context.Err() != nil {
			return false
		}
		_ = policy.Remove(filepath.Join(directory, name))
	}
	return true
}
