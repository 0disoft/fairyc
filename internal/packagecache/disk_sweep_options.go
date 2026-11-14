package packagecache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

// SweepOptions override dependencies for one sweep only. Nil fields retain the
// opened-root filesystem, current clock and background context. Context is Go's
// cancellation signal; a blocked filesystem syscall itself is not preemptible.
// Filesystem callbacks receive paths relative to the opened retention root and
// must retain the corresponding os.Root operation's semantics.
type SweepOptions struct {
	Context       context.Context
	Now           func() time.Time
	ReadDirectory func(string, *directoryscan.Budget) ([]os.DirEntry, error)
	Lstat         func(string) (os.FileInfo, error)
	Remove        func(string) error
}

type sweepPolicy struct {
	SweepOptions
	instant time.Time
}
type diskTemporary struct {
	path     string
	identity os.FileInfo
}

func (store *DiskStore) sweepPolicy(options []SweepOptions) (sweepPolicy, error) {
	if len(options) > 1 {
		return sweepPolicy{}, fmt.Errorf("at most one sweep options value is supported")
	}
	var selected SweepOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.Context == nil {
		selected.Context = context.Background()
	}
	if err := selected.Context.Err(); err != nil {
		return sweepPolicy{}, err
	}
	if selected.Now == nil {
		selected.Now = time.Now
	}
	if selected.Lstat == nil {
		selected.Lstat = store.retentionRoot.Lstat
	}
	if selected.Remove == nil {
		selected.Remove = store.retentionRoot.Remove
	}
	if selected.ReadDirectory == nil {
		selected.ReadDirectory = func(name string, budget *directoryscan.Budget) ([]os.DirEntry, error) {
			return readCacheDirectoryWithBudget(store.retentionRoot, name, budget)
		}
	}
	return sweepPolicy{SweepOptions: selected, instant: selected.Now()}, nil
}

func (store *DiskStore) removeExpiredTemporary(candidate diskTemporary, policy sweepPolicy) error {
	if err := policy.Context.Err(); err != nil {
		return err
	}
	current, err := policy.Lstat(candidate.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("recheck expired cache temporary file: %w", err)
	}
	if err := policy.Context.Err(); err != nil {
		return err
	}
	// A scan is an observation, not permission to delete a later file at this
	// name. Preserve replaced, refreshed, resized or no-longer-expired files.
	// Recheck and unlink are separate operations; this is not an atomic guard
	// against a writer changing the path again between these two syscalls.
	if !current.Mode().IsRegular() || !os.SameFile(candidate.identity, current) || current.Size() != candidate.identity.Size() || !current.ModTime().Equal(candidate.identity.ModTime()) || policy.instant.Sub(current.ModTime()) < store.limits.TempMaxAge {
		return nil
	}
	if err := policy.Remove(candidate.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove expired cache temporary file: %w", err)
	}
	return nil
}
