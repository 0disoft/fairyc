package workspaceloader

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// Count all entries, including ignored files and empty directories. The source
// count/byte guards alone do not bound directory enumeration before filtering.
const MaxSourceEntries = 100_000
const sourceDirectoryBatch = 128

func walkSourceTree(root fs.FS, budget *limits, maximum int, visit fs.WalkDirFunc) error {
	if err := chargeSourceEntries(budget, maximum, 1); err != nil {
		return err
	}
	info, err := fs.Stat(root, ".")
	if err != nil {
		return visit(".", nil, err)
	}
	var walk func(string, fs.DirEntry) error
	walk = func(name string, entry fs.DirEntry) error {
		if err := visit(name, entry, nil); err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		entries, err := readSourceEntries(root, name, budget, maximum)
		if err != nil {
			return visit(name, entry, err)
		}
		for _, child := range entries {
			if err := walk(path.Join(name, child.Name()), child); err != nil {
				if err == fs.SkipDir {
					if child.IsDir() {
						continue
					}
					return nil
				}
				return err
			}
		}
		return nil
	}
	err = walk(".", fs.FileInfoToDirEntry(info))
	if err == fs.SkipAll || err == fs.SkipDir {
		return nil
	}
	return err
}

func readSourceEntries(root fs.FS, name string, budget *limits, maximum int) (entries []fs.DirEntry, err error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	directory, ok := file.(fs.ReadDirFile)
	if !ok {
		return nil, fmt.Errorf("source directory does not support bounded enumeration")
	}
	for {
		if budget.ctx != nil {
			if err := budget.ctx.Err(); err != nil {
				return nil, err
			}
		}
		batch, readErr := directory.ReadDir(min(sourceDirectoryBatch, max(1, maximum-budget.entries+1)))
		if err := chargeSourceEntries(budget, maximum, len(batch)); err != nil {
			return nil, err
		}
		entries = append(entries, batch...)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	// Match WalkDir's lexical diagnostic order, with a bounded entry catalog.
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func chargeSourceEntries(budget *limits, maximum, count int) error {
	if budget.ctx != nil {
		if err := budget.ctx.Err(); err != nil {
			return err
		}
	}
	if count > maximum-budget.entries {
		return &LoadError{Reason: ReasonSourceLimit, Detail: fmt.Sprintf("workspace exceeds %d filesystem entries", maximum)}
	}
	budget.entries += count
	return nil
}
