package packagesource

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"

	"github.com/0disoft/fairyc/internal/directoryscan"
	"github.com/0disoft/fairyc/internal/durablefile"
)

func verifyTree(ctx context.Context, root *os.Root, name string, b bundle) error {
	return verifyTreeWithin(ctx, root, name, b, MaxFiles*2+1)
}

func verifyTreeWithin(ctx context.Context, root *os.Root, name string, b bundle, maxEntries int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if maxEntries < 1 {
		return fmt.Errorf("existing package entry limit exceeded")
	}
	if err := directory(root, name, false); err != nil {
		return err
	}
	before, err := root.Lstat(name)
	if err != nil {
		return err
	}
	child, err := root.OpenRoot(name)
	if err != nil {
		return err
	}
	defer child.Close()
	opened, err := child.Stat(".")
	if err != nil || !before.IsDir() || !os.SameFile(before, opened) {
		return fmt.Errorf("existing package directory changed during open")
	}
	budget := &directoryscan.Budget{Context: ctx, Remaining: maxEntries - 1}
	files := 0
	type pendingEntry struct {
		name  string
		entry fs.DirEntry
	}
	// Entries are admitted globally before retention. Close each directory
	// before descending so deep trees cannot exhaust the process handle limit.
	pending := []pendingEntry{{name: "."}}
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		item := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if item.entry != nil {
			if item.entry.Type()&os.ModeSymlink != 0 || !portableEntry(item.name) {
				return fmt.Errorf("existing package link or nonportable entry")
			}
			if !item.entry.IsDir() {
				expected, found := b.files[item.name]
				if !found {
					return fmt.Errorf("existing package contains additional file")
				}
				matches, err := readRegularMatches(ctx, child, item.name, MaxFileBytes, expected)
				if err != nil {
					return err
				}
				if !matches {
					return fmt.Errorf("existing package differs from pinned archive")
				}
				files++
				continue
			}
		}
		directory, err := durablefile.OpenDirectory(child, item.name)
		if err != nil {
			return err
		}
		entries, readErr := directoryscan.ReadSorted(directory, budget)
		closeErr := directory.Close()
		if errors.Is(readErr, directoryscan.ErrLimit) {
			return fmt.Errorf("existing package entry limit exceeded")
		}
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		for index := len(entries) - 1; index >= 0; index-- {
			entry := entries[index]
			pending = append(pending, pendingEntry{name: path.Join(item.name, entry.Name()), entry: entry})
		}
	}
	if files != len(b.files) {
		return fmt.Errorf("existing package is incomplete")
	}
	return nil
}

func publish(ctx context.Context, root *os.Root, pin Pin, b bundle) (installed bool, err error) {
	if _, err := root.Lstat(pin.Path); err == nil {
		return false, verifyTree(ctx, root, pin.Path, b)
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := directory(root, "vendor", true); err != nil {
		return false, err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return false, err
	}
	stage := "vendor/.fairyc-source-" + hex.EncodeToString(nonce[:])
	if err := root.Mkdir(stage, 0700); err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, root.RemoveAll(stage)) }()
	names := make([]string, 0, len(b.files))
	for name := range b.files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		destination := path.Join(stage, name)
		if err := directory(root, path.Dir(destination), true); err != nil {
			return false, err
		}
		file, err := root.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return false, err
		}
		_, writeErr := file.Write(b.files[name])
		syncErr := file.Sync()
		closeErr := file.Close()
		if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
			return false, err
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if _, err := root.Lstat(pin.Path); err == nil {
		return false, fmt.Errorf("package destination appeared during fetch")
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := root.Rename(stage, pin.Path); err != nil {
		return false, err
	}
	return true, nil
}
