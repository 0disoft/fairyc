package sourceedit

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const pathCaseBatch = 128

type pathCaseDirectory interface {
	ReadDir(int) ([]os.DirEntry, error)
	Close() error
}

// A listing belongs to one snapshot, never a Plan or a later Apply check.
// Only spelling is reused: readFile still Lstats each path segment and checks
// the final opened file's identity. The existing workspace entry ceiling bounds
// retained names as well as enumeration, including unrelated directory entries.
type pathCaseIndex struct {
	listings map[string]map[string]struct{}
	entries  int
	open     func(string) (pathCaseDirectory, error)
}

func newPathCaseIndex(root *os.Root) *pathCaseIndex {
	return &pathCaseIndex{
		listings: make(map[string]map[string]struct{}),
		open: func(name string) (pathCaseDirectory, error) {
			return root.Open(name)
		},
	}
}

func (index *pathCaseIndex) contains(parent, part string) (bool, error) {
	names, ok := index.listings[parent]
	if !ok {
		var err error
		names, err = index.read(parent)
		if err != nil {
			return false, err // Never retain a failed or incomplete listing.
		}
		index.listings[parent] = names
	}
	_, found := names[part]
	return found, nil
}

func (index *pathCaseIndex) read(parent string) (names map[string]struct{}, err error) {
	directory, err := index.open(parent)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	names = make(map[string]struct{})
	for {
		remaining := workspaceloader.MaxSourceEntries - index.entries
		entries, readErr := directory.ReadDir(min(pathCaseBatch, remaining+1))
		if len(entries) > remaining {
			return nil, fmt.Errorf("edit snapshot exceeds %d filesystem entries", workspaceloader.MaxSourceEntries)
		}
		index.entries += len(entries)
		for _, entry := range entries {
			names[entry.Name()] = struct{}{}
		}
		if readErr == io.EOF {
			return names, nil
		}
		if readErr != nil {
			return nil, readErr
		}
	}
}

func checkPathCase(root *os.Root, index *pathCaseIndex, parent, part string) error {
	if index != nil {
		found, err := index.contains(parent, part)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("noncanonical edit path case")
		}
		return nil
	}
	// A standalone body-file read needs one spelling, not a snapshot-wide index.
	directory, err := root.Open(parent)
	if err != nil {
		return err
	}
	defer directory.Close()
	for {
		entries, readErr := directory.ReadDir(pathCaseBatch)
		for _, entry := range entries {
			if entry.Name() == part {
				return nil
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				return readErr
			}
			return fmt.Errorf("noncanonical edit path case")
		}
	}
}
