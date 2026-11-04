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
)

func verifyTree(ctx context.Context, root *os.Root, name string, b bundle) error {
	if err := directory(root, name, false); err != nil {
		return err
	}
	child, err := root.OpenRoot(name)
	if err != nil {
		return err
	}
	defer child.Close()
	files, entries := 0, 0
	err = fs.WalkDir(child.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > MaxFiles*2+1 {
			return fmt.Errorf("existing package entry limit exceeded")
		}
		if name == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !portableEntry(name) {
			return fmt.Errorf("existing package link or nonportable entry")
		}
		if entry.IsDir() {
			return nil
		}
		expected, found := b.files[name]
		if !found {
			return fmt.Errorf("existing package contains additional file")
		}
		matches, err := readRegularMatches(ctx, child, name, MaxFileBytes, expected)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("existing package differs from pinned archive")
		}
		files++
		return nil
	})
	if err != nil {
		return err
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
