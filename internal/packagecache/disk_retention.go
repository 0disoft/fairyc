package packagecache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/directoryscan"
)

type diskScanResult struct {
	generations        []diskGeneration
	expiredTemps       []diskTemporary
	cleanupDirectories []string
}

// Collection visits only compiler-owned, current-format namespaces. Unknown
// versions and unrelated files are preserved; this is not recursive cleanup.
func (store *DiskStore) scan(clean bool) (diskScanResult, error) {
	policy, err := store.sweepPolicy(nil)
	if err != nil {
		return diskScanResult{}, err
	}
	return store.scanWithPolicy(clean, policy)
}

func (store *DiskStore) scanWithPolicy(clean bool, policy sweepPolicy) (diskScanResult, error) {
	budget := store.newScanBudget()
	budget.Context = policy.Context
	var result diskScanResult
	for _, namespace := range []string{".", "generic-v0", "storage-transfer-v0"} {
		if err := policy.Context.Err(); err != nil {
			return diskScanResult{}, err
		}
		if namespace != "." {
			info, err := policy.Lstat(namespace)
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
		info, err := policy.Lstat(versionPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return diskScanResult{}, fmt.Errorf("inspect cache format directory: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		namespaceResult, err := store.scanNamespace(versionPath, clean, budget, policy)
		if err != nil {
			return diskScanResult{}, err
		}
		result.generations = append(result.generations, namespaceResult.generations...)
		result.expiredTemps = append(result.expiredTemps, namespaceResult.expiredTemps...)
		result.cleanupDirectories = append(result.cleanupDirectories, namespaceResult.cleanupDirectories...)
	}
	return result, nil
}

func (store *DiskStore) scanNamespace(versionPath string, clean bool, budget *directoryscan.Budget, policy sweepPolicy) (diskScanResult, error) {
	keyDirectories, err := policy.ReadDirectory(versionPath, budget)
	if err != nil {
		return diskScanResult{}, fmt.Errorf("scan package cache root: %w", err)
	}
	var result diskScanResult
	for _, keyDirectory := range keyDirectories {
		if err := policy.Context.Err(); err != nil {
			return diskScanResult{}, err
		}
		if !keyDirectory.IsDir() || !validDigestDirectory(keyDirectory.Name()) {
			continue
		}
		directoryPath := filepath.Join(versionPath, keyDirectory.Name())
		entries, err := policy.ReadDirectory(directoryPath, budget)
		if errors.Is(err, os.ErrNotExist) {
			continue // Another collector removed this empty key directory.
		}
		if err != nil {
			return diskScanResult{}, fmt.Errorf("scan package cache key directory: %w", err)
		}
		for _, entry := range entries {
			if err := policy.Context.Err(); err != nil {
				return diskScanResult{}, err
			}
			path := filepath.Join(directoryPath, entry.Name())
			info, err := policy.Lstat(path)
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
				if clean && policy.instant.Sub(info.ModTime()) >= store.limits.TempMaxAge {
					result.expiredTemps = append(result.expiredTemps, diskTemporary{path: path, identity: info})
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
