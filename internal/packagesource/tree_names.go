package packagesource

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/directoryscan"
	"github.com/0disoft/fairyc/internal/durablefile"
)

// An index lives only for one installed-tree verification. It never replaces
// per-file kind/size/opened-identity checks. Two bounded directory passes avoid
// F repeated sibling scans while rejecting changes to the spelling snapshot.
type treeNameIndex struct {
	root *os.Root
	dirs map[string]treeDirectoryNames
}

type treeDirectoryNames struct {
	identity os.FileInfo
	names    map[string]fs.FileMode
}

func (index *treeNameIndex) remember(name string, identity os.FileInfo, entries []fs.DirEntry) {
	names := make(map[string]fs.FileMode, len(entries))
	for _, entry := range entries {
		names[entry.Name()] = entry.Type()
	}
	index.dirs[name] = treeDirectoryNames{identity: identity, names: names}
}

func (index *treeNameIndex) identity(name string) error {
	previous, found := index.dirs[name]
	current, err := index.root.Lstat(name)
	if err != nil {
		return err
	}
	if !found || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(previous.identity, current) {
		return fmt.Errorf("existing package directory changed during verification")
	}
	return nil
}

func (index *treeNameIndex) exactName(parent, name string) error {
	if err := index.identity(parent); err != nil {
		return err
	}
	if _, found := index.dirs[parent].names[name]; !found {
		return fmt.Errorf("noncanonical path case")
	}
	return nil
}

func (index *treeNameIndex) directory(name string) error {
	if path.Clean(name) != name || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") || name == ".." || strings.HasPrefix(name, "../") {
		return fmt.Errorf("invalid contained directory")
	}
	if name != "." {
		parent := "."
		for _, part := range strings.Split(name, "/") {
			if err := index.exactName(parent, part); err != nil {
				return err
			}
			parent = path.Join(parent, part)
		}
	}
	return index.identity(name)
}

func (index *treeNameIndex) verifyStable(ctx context.Context, maxEntries int) error {
	budget := &directoryscan.Budget{Context: ctx, Remaining: maxEntries - 1}
	names := make([]string, 0, len(index.dirs))
	for name := range index.dirs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := index.directory(name); err != nil {
			return err
		}
		directory, err := durablefile.OpenDirectory(index.root, name)
		if err != nil {
			return err
		}
		identity, statErr := directory.Stat()
		entries, readErr := directoryscan.ReadSorted(directory, budget)
		closeErr := directory.Close()
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return err
		}
		previous := index.dirs[name]
		if !os.SameFile(previous.identity, identity) || len(entries) != len(previous.names) {
			return fmt.Errorf("existing package directory changed during verification")
		}
		for _, entry := range entries {
			kind, found := previous.names[entry.Name()]
			if !found || kind != entry.Type() {
				return fmt.Errorf("existing package directory changed during verification")
			}
		}
	}
	return nil
}
