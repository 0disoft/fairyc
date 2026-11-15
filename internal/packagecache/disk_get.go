package packagecache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

var errDiskGenerationCorrupt = errors.New("package cache generation is corrupt")

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
	var policy DiskGetOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	ctx := policy.Context
	if ctx.Err() != nil {
		return miss()
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
	keyDirectory := store.keyDirectoryName(key)
	budget := store.newScanBudget()
	budget.Context = ctx
	entries, err := policy.ReadDirectory(keyDirectory, budget)
	if err != nil || ctx.Err() != nil {
		return miss()
	}
	slices.SortFunc(entries, func(left, right os.DirEntry) int { return strings.Compare(left.Name(), right.Name()) })
	var candidate Record
	validCount := 0
	for _, entry := range entries {
		if ctx.Err() != nil {
			return miss()
		}
		if entry.IsDir() || !validGenerationName(entry.Name()) {
			continue
		}
		path := filepath.Join(keyDirectory, entry.Name())
		record, readErr := policy.ReadGeneration(ctx, path, entry.Name(), key)
		if ctx.Err() != nil {
			return miss()
		}
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue // A concurrent collector removed this generation.
			}
			if !errors.Is(readErr, errDiskGenerationCorrupt) {
				return miss() // An unread sibling may hide a producer conflict.
			}
			store.recordRejected()
			_ = policy.Remove(path)
			continue
		}
		validCount++
		if validCount == 1 {
			candidate = record
		} else {
			// Validate/count later generations without retaining extra payloads.
			candidate = Record{}
		}
	}
	if ctx.Err() != nil {
		return miss()
	}
	if validCount != 1 {
		if validCount > 1 {
			store.recordRejectedBy(uint64(validCount))
			for _, entry := range entries {
				if ctx.Err() != nil {
					return miss()
				}
				if !entry.IsDir() && validGenerationName(entry.Name()) {
					_ = policy.Remove(filepath.Join(keyDirectory, entry.Name()))
				}
			}
		}
		removeEmptyKeyDirectoryUsing(keyDirectory, ctx, store.root.Lstat, policy.Remove)
		return miss()
	}
	store.recordHit()
	return candidate, true
}
