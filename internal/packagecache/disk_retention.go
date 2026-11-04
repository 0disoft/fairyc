package packagecache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

type diskScanResult struct {
	generations        []diskGeneration
	expiredTemps       []string
	cleanupDirectories []string
}

// Collection visits only compiler-owned, current-format namespaces. Unknown
// versions and unrelated files are preserved; this is not recursive cleanup.
func (store *DiskStore) scan(clean bool) (diskScanResult, error) {
	budget := store.newScanBudget()
	var result diskScanResult
	for _, namespace := range []string{".", "generic-v0", "storage-transfer-v0"} {
		if namespace != "." {
			info, err := store.retentionRoot.Lstat(namespace)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return diskScanResult{}, fmt.Errorf("inspect cache namespace: %w", err)
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
			return diskScanResult{}, fmt.Errorf("inspect cache format directory: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		namespaceResult, err := store.scanNamespace(versionPath, clean, budget)
		if err != nil {
			return diskScanResult{}, err
		}
		result.generations = append(result.generations, namespaceResult.generations...)
		result.expiredTemps = append(result.expiredTemps, namespaceResult.expiredTemps...)
		result.cleanupDirectories = append(result.cleanupDirectories, namespaceResult.cleanupDirectories...)
	}
	return result, nil
}

func (store *DiskStore) scanNamespace(versionPath string, clean bool, budget *directoryscan.Budget) (diskScanResult, error) {
	keyDirectories, err := readCacheDirectoryWithBudget(store.retentionRoot, versionPath, budget)
	if err != nil {
		return diskScanResult{}, fmt.Errorf("scan package cache root: %w", err)
	}
	now := time.Now()
	var result diskScanResult
	for _, keyDirectory := range keyDirectories {
		if !keyDirectory.IsDir() || !validDigestDirectory(keyDirectory.Name()) {
			continue
		}
		directoryPath := filepath.Join(versionPath, keyDirectory.Name())
		entries, err := readCacheDirectoryWithBudget(store.retentionRoot, directoryPath, budget)
		if errors.Is(err, os.ErrNotExist) {
			continue // Another collector removed this empty key directory.
		}
		if err != nil {
			return diskScanResult{}, fmt.Errorf("scan package cache key directory: %w", err)
		}
		for _, entry := range entries {
			path := filepath.Join(directoryPath, entry.Name())
			info, err := store.retentionRoot.Lstat(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return diskScanResult{}, fmt.Errorf("inspect package cache file: %w", err)
			}
			if !info.Mode().IsRegular() {
				continue
			}
			if strings.HasPrefix(entry.Name(), ".tmp-") {
				if clean && now.Sub(info.ModTime()) >= store.limits.TempMaxAge {
					result.expiredTemps = append(result.expiredTemps, path)
				}
				continue
			}
			if !validGenerationName(entry.Name()) {
				continue
			}
			// MaxRecordBytes is a reader bound, not a reason to erase a sibling
			// store's larger record. Every generation still counts toward retention.
			result.generations = append(result.generations, diskGeneration{path: path, name: entry.Name(), size: uint64(info.Size()), modTime: info.ModTime()})
		}
		if clean {
			result.cleanupDirectories = append(result.cleanupDirectories, directoryPath)
		}
	}
	return result, nil
}
