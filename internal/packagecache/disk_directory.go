package packagecache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/durablefile"
)

func removeEmptyKeyDirectory(root *os.Root, name string) {
	if !validDigestDirectory(filepath.Base(name)) {
		return
	}
	info, err := root.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return
	}
	_ = root.Remove(name) // Nonrecursive; a concurrent temporary keeps it nonempty.
}

func (store *DiskStore) prepareKeyDirectory(name string) error {
	if err := store.root.Mkdir(name, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create package cache key directory: %w", err)
		}
		info, err := store.root.Lstat(name)
		if err != nil {
			return fmt.Errorf("inspect package cache key directory: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("package cache key path is not a real directory")
		}
		return nil
	}
	return durablefile.SyncDirectory(store.root, ".")
}

// Collection can remove an empty key directory between Mkdir and temporary-file
// creation. Retry that race once; a live temporary or published record prevents
// the collector's nonrecursive Remove from deleting the directory.
func (store *DiskStore) createKeyTemporary(name string, beforeCreate func(string)) (file *os.File, path string, err error) {
	for range 2 {
		err = store.prepareKeyDirectory(name)
		if err == nil {
			if beforeCreate != nil {
				beforeCreate(store.absolutePath(name))
				beforeCreate = nil
			}
			file, path, err = durablefile.CreateTemporary(store.root, name)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return
		}
	}
	return
}
