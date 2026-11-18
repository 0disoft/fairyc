package packagesource

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	names := &treeNameIndex{root: child, dirs: make(map[string]treeDirectoryNames)}
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
				matches, err := readRegularMatchesWithNames(ctx, child, item.name, MaxFileBytes, expected, names)
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
		identity, statErr := directory.Stat()
		closeErr := directory.Close()
		if errors.Is(readErr, directoryscan.ErrLimit) {
			return fmt.Errorf("existing package entry limit exceeded")
		}
		if err := errors.Join(readErr, statErr, closeErr); err != nil {
			return err
		}
		names.remember(item.name, identity, entries)
		for index := len(entries) - 1; index >= 0; index-- {
			entry := entries[index]
			pending = append(pending, pendingEntry{name: path.Join(item.name, entry.Name()), entry: entry})
		}
	}
	if files != len(b.files) {
		return fmt.Errorf("existing package is incomplete")
	}
	return names.verifyStable(ctx, maxEntries)
}

func publish(ctx context.Context, root *os.Root, pin Pin, b bundle, options ...sourcePublishOptions) (installed bool, err error) {
	policy, err := sourcePublicationPolicy(options)
	if err != nil {
		return false, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	defer func() { err = errors.Join(err, ctx.Err(), context.Cause(ctx)) }()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if root == nil {
		return false, fmt.Errorf("package publication requires an opened root")
	}
	if _, err := root.Lstat(pin.Path); err == nil {
		return false, verifyTree(ctx, root, pin.Path, b)
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := directory(root, "vendor", true); err != nil {
		return false, err
	}
	if err := rejectCaseAlias(root, path.Dir(pin.Path), path.Base(pin.Path)); err != nil {
		return false, err
	}
	var nonce [16]byte
	n, err := policy.Random(nonce[:])
	if err := errors.Join(err, ctx.Err()); err != nil {
		return false, err
	}
	if n != len(nonce) {
		return false, fmt.Errorf("package staging nonce: %w", io.ErrUnexpectedEOF)
	}
	stage := "vendor/.fairyc-source-" + hex.EncodeToString(nonce[:])
	if err := root.Mkdir(stage, 0700); err != nil {
		return false, err
	}
	identity, err := root.Lstat(stage)
	if err != nil {
		return false, err
	}
	owned := true
	defer func() {
		if owned {
			err = errors.Join(err, removeSourceStage(root, stage, identity, policy.RemoveAll))
		}
	}()
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
		if err := writeSourceStageFile(ctx, file, b.files[name], policy); err != nil {
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
	if err := rejectCaseAlias(root, path.Dir(pin.Path), path.Base(pin.Path)); err != nil {
		return false, err
	}
	if err := checkSourceStage(root, stage, identity); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := policy.Rename(root, stage, pin.Path); err != nil {
		return false, err
	}
	// Rename commits the installation. This name no longer belongs to us,
	// even when another directory appears there before the call returns.
	owned = false
	return true, nil
}
