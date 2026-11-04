// Package directoryscan bounds filesystem enumeration before retaining or sorting entries.
package directoryscan

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"slices"
	"strings"
)

const BatchSize = 128
const DefaultMaxEntries = 100_000

var ErrLimit = errors.New("directory entry scan limit exceeded")

// Budget is local to one scan and may be shared by all of its directories.
type Budget struct {
	Context   context.Context
	Remaining int
}

// ReadSorted retains at most the remaining entry budget and reads at most one
// extra entry to detect overflow. Entries returned with an error are partial;
// callers must not use them to certify a complete directory or cache hit.
func ReadSorted(directory fs.ReadDirFile, budget *Budget) (entries []fs.DirEntry, err error) {
	if budget == nil || budget.Remaining < 0 {
		return nil, ErrLimit
	}
	defer func() {
		slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	}()
	for {
		if budget.Context != nil {
			if err := budget.Context.Err(); err != nil {
				return entries, err
			}
		}
		size := BatchSize
		if budget.Remaining < size {
			size = budget.Remaining + 1
		}
		batch, readErr := directory.ReadDir(size)
		admitted := min(len(batch), budget.Remaining)
		budget.Remaining -= admitted
		entries = append(entries, batch[:admitted]...)
		if admitted < len(batch) {
			return entries, ErrLimit
		}
		if errors.Is(readErr, io.EOF) {
			return entries, nil
		}
		if readErr != nil {
			return entries, readErr
		}
		if len(batch) == 0 {
			return entries, io.ErrNoProgress
		}
	}
}
