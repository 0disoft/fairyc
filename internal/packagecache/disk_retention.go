package packagecache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Collection visits only compiler-owned, current-format namespaces. Unknown
// versions and unrelated files are preserved; this is not recursive cleanup.
func (store *DiskStore) scan(clean bool) ([]diskGeneration, error) {
	var result []diskGeneration
	for _, namespace := range []string{".", "generic-v0", "storage-transfer-v0"} {
		if namespace != "." {
			info, err := store.retentionRoot.Lstat(namespace)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("inspect cache namespace: %w", err)
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				continue
			}
		}
		versionPath := filepath.Join(namespace, diskLayoutDirectory)
		info, err := store.retentionRoot.Lstat(versionPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect cache format directory: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		generations, err := store.scanNamespace(versionPath, clean)
		if err != nil {
			return nil, err
		}
		result = append(result, generations...)
	}
	return result, nil
}

func (store *DiskStore) scanNamespace(versionPath string, clean bool) ([]diskGeneration, error) {
	keyDirectories, err := readCacheDirectory(store.retentionRoot, versionPath)
	if err != nil {
		return nil, fmt.Errorf("scan package cache root: %w", err)
	}
	now := time.Now()
	var result []diskGeneration
	for _, keyDirectory := range keyDirectories {
		if !keyDirectory.IsDir() || !validDigestDirectory(keyDirectory.Name()) {
			continue
		}
		directoryPath := filepath.Join(versionPath, keyDirectory.Name())
		entries, err := readCacheDirectory(store.retentionRoot, directoryPath)
		if errors.Is(err, os.ErrNotExist) {
			continue // Another collector removed this empty key directory.
		}
		if err != nil {
			return nil, fmt.Errorf("scan package cache key directory: %w", err)
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("inspect package cache file: %w", err)
			}
			if !info.Mode().IsRegular() {
				continue
			}
			path := filepath.Join(directoryPath, entry.Name())
			if strings.HasPrefix(entry.Name(), ".tmp-") {
				if clean && now.Sub(info.ModTime()) >= store.limits.TempMaxAge {
					if err := store.retentionRoot.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
						return nil, fmt.Errorf("remove expired cache temporary file: %w", err)
					}
				}
				continue
			}
			if !validGenerationName(entry.Name()) {
				continue
			}
			// MaxRecordBytes is a reader bound, not a reason to erase a sibling
			// store's larger record. Every generation still counts toward retention.
			result = append(result, diskGeneration{path: path, name: entry.Name(), size: uint64(info.Size()), modTime: info.ModTime()})
		}
		if clean {
			// Never recurse: unknown files, live temporaries and concurrent
			// publications keep their directory. Failure is harmless.
			removeEmptyKeyDirectory(store.retentionRoot, directoryPath)
		}
	}
	return result, nil
}
